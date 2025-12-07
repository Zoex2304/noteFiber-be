package service

import (
	"ai-notetaking-be/internal/dto"
	"ai-notetaking-be/internal/entity"
	"ai-notetaking-be/internal/repository"
	"ai-notetaking-be/pkg/embedding"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type INoteService interface {
	Create(ctx context.Context, userId uuid.UUID, req *dto.CreateNoteRequest) (*dto.CreateNoteResponse, error)
	Show(ctx context.Context, userId uuid.UUID, id uuid.UUID) (*dto.ShowNoteResponse, error)
	Update(ctx context.Context, userId uuid.UUID, req *dto.UpdateNoteRequest) (*dto.UpdateNoteResponse, error)
	Delete(ctx context.Context, userId uuid.UUID, id uuid.UUID) error
	MoveNote(ctx context.Context, userId uuid.UUID, req *dto.MoveNoteRequest) (*dto.MoveNoteResponse, error)
	SemanticSearch(ctx context.Context, userId uuid.UUID, search string) ([]*dto.SemanticSearchResponse, error)
}

type noteService struct {
	noteRepository          repository.INoteRepository
	noteEmbeddingRepository repository.INoteEmbeddingRepository
	publisherService        IPublisherService
	db                      *pgxpool.Pool
	subRepo                 repository.ISubscriptionRepository
}

func NewNoteService(
	noteRepository repository.INoteRepository,
	publisherService IPublisherService,
	noteEmbeddingRepository repository.INoteEmbeddingRepository,
	db *pgxpool.Pool,
	subRepo repository.ISubscriptionRepository,
) INoteService {
	return &noteService{
		noteRepository:          noteRepository,
		noteEmbeddingRepository: noteEmbeddingRepository,
		publisherService:        publisherService,
		db:                      db,
		subRepo:                 subRepo,
	}
}

func (c *noteService) Create(ctx context.Context, userId uuid.UUID, req *dto.CreateNoteRequest) (*dto.CreateNoteResponse, error) {
	note := entity.Note{
		Id:         uuid.New(),
		Title:      req.Title,
		Content:    req.Content,
		NotebookId: req.NotebookId,
		UserId:     userId, // ✅ Set Owner
		CreatedAt:  time.Now(),
	}

	err := c.noteRepository.Create(ctx, &note)
	if err != nil {
		return nil, err
	}

	msgPayload := dto.PublishEmbedNoteMessage{
		NoteId: note.Id,
	}
	msgJson, err := json.Marshal(msgPayload)
	if err != nil {
		return nil, err
	}

	err = c.publisherService.Publish(ctx, msgJson)
	if err != nil {
		return nil, err
	}

	return &dto.CreateNoteResponse{
		Id: note.Id,
	}, nil
}

func (c *noteService) Show(ctx context.Context, userId uuid.UUID, id uuid.UUID) (*dto.ShowNoteResponse, error) {
	// ✅ FIX: Pass userId to repository
	note, err := c.noteRepository.GetById(ctx, id, userId)
	if err != nil {
		return nil, err
	}

	res := dto.ShowNoteResponse{
		Id:         note.Id,
		Title:      note.Title,
		Content:    note.Content,
		NotebookId: note.NotebookId,
		CreatedAt:  note.CreatedAt,
		UpdatedAt:  note.UpdatedAt,
	}

	return &res, nil
}

func (c *noteService) Update(ctx context.Context, userId uuid.UUID, req *dto.UpdateNoteRequest) (*dto.UpdateNoteResponse, error) {
	// ✅ FIX: Pass userId to repository to check ownership first
	note, err := c.noteRepository.GetById(ctx, req.Id, userId)
	if err != nil {
		return nil, err
	}

	now := time.Now()

	note.Title = req.Title
	note.Content = req.Content
	note.UpdatedAt = &now

	err = c.noteRepository.Update(ctx, note)
	if err != nil {
		return nil, err
	}

	payload := dto.PublishEmbedNoteMessage{
		NoteId: note.Id,
	}
	payloadJson, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	err = c.publisherService.Publish(ctx, payloadJson)
	if err != nil {
		return nil, err
	}

	return &dto.UpdateNoteResponse{
		Id: note.Id,
	}, nil
}

func (c *noteService) Delete(ctx context.Context, userId uuid.UUID, id uuid.UUID) error {
	// ✅ FIX: Pass userId to ensure user owns the note before deleting
	_, err := c.noteRepository.GetById(ctx, id, userId)
	if err != nil {
		return err
	}

	tx, err := c.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	noteRepository := c.noteRepository.UsingTx(ctx, tx)
	noteEmbeddingRepository := c.noteEmbeddingRepository.UsingTx(ctx, tx)

	// ✅ FIX: Pass userId to delete method
	err = noteRepository.Delete(ctx, id, userId)
	if err != nil {
		return err
	}

	err = noteEmbeddingRepository.DeleteByNoteId(ctx, id)
	if err != nil {
		return err
	}

	err = tx.Commit(ctx)
	if err != nil {
		return err
	}

	return nil
}

func (c *noteService) MoveNote(ctx context.Context, userId uuid.UUID, req *dto.MoveNoteRequest) (*dto.MoveNoteResponse, error) {
	// ✅ FIX: Check ownership
	note, err := c.noteRepository.GetById(ctx, req.Id, userId)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	note.NotebookId = req.NotebookId
	note.UpdatedAt = &now

	err = c.noteRepository.Update(ctx, note)
	if err != nil {
		return nil, err
	}

	payload := dto.PublishEmbedNoteMessage{
		NoteId: note.Id,
	}
	payloadJson, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	err = c.publisherService.Publish(ctx, payloadJson)
	if err != nil {
		return nil, err
	}

	return &dto.MoveNoteResponse{
		Id: note.Id,
	}, nil
}

func (c *noteService) SemanticSearch(ctx context.Context, userId uuid.UUID, search string) ([]*dto.SemanticSearchResponse, error) {
	// ✅ GUARD: Check Pro Plan / SemanticSearchEnabled
	sub, plan, err := c.subRepo.GetActiveByUserId(ctx, userId)
	if err != nil {
		return nil, err
	}
	// If no subscription OR feature disabled in plan
	if sub == nil || !plan.SemanticSearchEnabled {
		return nil, fmt.Errorf("feature requires pro plan")
	}

	embeddingRes, err := embedding.GetGeminiEmbedding(
		os.Getenv("GOOGLE_GEMINI_API_KEY"),
		search,
		"RETRIEVAL_QUERY",
	)
	if err != nil {
		return nil, err
	}

	noteEmbeddings, err := c.noteEmbeddingRepository.SemanticSearch(ctx, embeddingRes.Embedding.Values)
	if err != nil {
		return nil, err
	}

	ids := make([]uuid.UUID, 0)
	for _, noteEmbedding := range noteEmbeddings {
		ids = append(ids, noteEmbedding.NoteId)
	}

	// ✅ FIX: Use GetByIds with userId to ensure we ONLY retrieve notes owned by the user
	notes, err := c.noteRepository.GetByIds(ctx, ids, userId)
	if err != nil {
		return nil, err
	}

	response := make([]*dto.SemanticSearchResponse, 0)
	for _, noteEmbedding := range noteEmbeddings {
		for _, note := range notes {
			if noteEmbedding.NoteId == note.Id {
				response = append(response, &dto.SemanticSearchResponse{
					Id:         note.Id,
					Title:      note.Title,
					Content:    note.Content,
					NotebookId: note.NotebookId,
					CreatedAt:  note.CreatedAt,
					UpdatedAt:  note.UpdatedAt,
				})
			}
		}
	}

	return response, nil
}