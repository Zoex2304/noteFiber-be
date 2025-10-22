package service

import (
	"ai-notetaking-be/internal/dto"
	"ai-notetaking-be/internal/entity"
	"ai-notetaking-be/internal/repository"
	"ai-notetaking-be/pkg/embedding"
	"context"
	"encoding/json"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type INoteService interface {
	Create(ctx context.Context, req *dto.CreateNoteRequest) (*dto.CreateNoteResponse, error)
	Show(ctx context.Context, id uuid.UUID) (*dto.ShowNoteResponse, error)
	Update(ctx context.Context, req *dto.UpdateNoteRequest) (*dto.UpdateNoteResponse, error)
	Delete(ctx context.Context, id uuid.UUID) error
	MoveNote(ctx context.Context, req *dto.MoveNoteRequest) (*dto.MoveNoteResponse, error)
	SemanticSearch(ctx context.Context, search string) ([]*dto.SemanticSearchResponse, error)
}

type noteService struct {
	noteRepository          repository.INoteRepository
	noteEmbeddingRepository repository.INoteEmbeddingRepository
	publisherService        IPublisherService
	db                      *pgxpool.Pool
}

func NewNoteService(
	noteRepository repository.INoteRepository,
	publisherService IPublisherService,
	noteEmbeddingRepository repository.INoteEmbeddingRepository,
	db *pgxpool.Pool,
) INoteService {
	return &noteService{
		noteRepository:          noteRepository,
		noteEmbeddingRepository: noteEmbeddingRepository,
		publisherService:        publisherService,
		db:                      db,
	}
}

func (c *noteService) Create(ctx context.Context, req *dto.CreateNoteRequest) (*dto.CreateNoteResponse, error) {
	note := entity.Note{
		Id:         uuid.New(),
		Title:      req.Title,
		Content:    req.Content,
		NotebookId: req.NotebookId,
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

func (c *noteService) Show(ctx context.Context, id uuid.UUID) (*dto.ShowNoteResponse, error) {
	note, err := c.noteRepository.GetById(ctx, id)
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

func (c *noteService) Update(ctx context.Context, req *dto.UpdateNoteRequest) (*dto.UpdateNoteResponse, error) {
	note, err := c.noteRepository.GetById(ctx, req.Id)
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

func (c *noteService) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := c.noteRepository.GetById(ctx, id)
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

	err = noteRepository.Delete(ctx, id)
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

func (c *noteService) MoveNote(ctx context.Context, req *dto.MoveNoteRequest) (*dto.MoveNoteResponse, error) {
	note, err := c.noteRepository.GetById(ctx, req.Id)
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

func (c *noteService) SemanticSearch(ctx context.Context, search string) ([]*dto.SemanticSearchResponse, error) {
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

	notes, err := c.noteRepository.GetByIds(ctx, ids)
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
