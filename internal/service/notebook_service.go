// internal\service\notebook_service.go
package service

import (
	"ai-notetaking-be/internal/dto"
	"ai-notetaking-be/internal/entity"
	"ai-notetaking-be/internal/repository"
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type INotebookService interface {
	GetAll(ctx context.Context, userId uuid.UUID) ([]*dto.GetAllNotebookResponse, error)
	Create(ctx context.Context, userId uuid.UUID, req *dto.CreateNotebookRequest) (*dto.CreateNotebookResponse, error)
	Show(ctx context.Context, userId uuid.UUID, id uuid.UUID) (*dto.ShowNotebookResponse, error)
	Update(ctx context.Context, userId uuid.UUID, req *dto.UpdateNotebookRequest) (*dto.UpdateNotebookResponse, error)
	Delete(ctx context.Context, userId uuid.UUID, id uuid.UUID) error
	MoveNotebook(ctx context.Context, userId uuid.UUID, req *dto.MoveNotebookRequest) (*dto.MoveNotebookResponse, error)
}

type notebookService struct {
	notebookRepository      repository.INotebookRepository
	noteRepository          repository.INoteRepository
	noteEmbeddingRepository repository.INoteEmbeddingRepository
	publisherService        IPublisherService
	db                      *pgxpool.Pool
}

func NewNotebookService(
	notebookRepository repository.INotebookRepository,
	noteRepository repository.INoteRepository,
	db *pgxpool.Pool,
	publisherService IPublisherService,
	noteEmbeddingRepository repository.INoteEmbeddingRepository,
) INotebookService {
	return &notebookService{
		notebookRepository:      notebookRepository,
		noteRepository:          noteRepository,
		db:                      db,
		publisherService:        publisherService,
		noteEmbeddingRepository: noteEmbeddingRepository,
	}
}

func (c *notebookService) GetAll(ctx context.Context, userId uuid.UUID) ([]*dto.GetAllNotebookResponse, error) {
	// ✅ FIX: Pass userId
	notebooks, err := c.notebookRepository.GetAll(ctx, userId)
	if err != nil {
		return nil, err
	}

	ids := make([]uuid.UUID, 0)
	result := make([]*dto.GetAllNotebookResponse, 0)
	for _, notebook := range notebooks {
		res := dto.GetAllNotebookResponse{
			Id:        notebook.Id,
			Name:      notebook.Name,
			ParentId:  notebook.ParentId,
			CreatedAt: notebook.CreatedAt,
			UpdatedAt: notebook.UpdatedAt,
			Notes:     make([]*dto.GetAllNotebookResponseNote, 0),
		}

		result = append([]*dto.GetAllNotebookResponse{&res}, result...)
		ids = append(ids, notebook.Id)
	}

	// ✅ FIX: Pass userId to fetch only user's notes
	notes, err := c.noteRepository.GetByNotebookIds(ctx, ids, userId)
	if err != nil {
		return nil, err
	}

	for i := 0; i < len(result); i++ {
		for j := 0; j < len(notes); j++ {
			if notes[j].NotebookId == result[i].Id {
				result[i].Notes = append(result[i].Notes, &dto.GetAllNotebookResponseNote{
					Id:        notes[j].Id,
					Title:     notes[j].Title,
					Content:   notes[j].Content,
					CreatedAt: notes[j].CreatedAt,
					UpdatedAt: notes[j].UpdatedAt,
				})
			}
		}
	}

	return result, nil
}

func (c *notebookService) Create(ctx context.Context, userId uuid.UUID, req *dto.CreateNotebookRequest) (*dto.CreateNotebookResponse, error) {
	notebook := entity.Notebook{
		Id:        uuid.New(),
		Name:      req.Name,
		ParentId:  req.ParentId,
		UserId:    userId, // ✅ Set Owner
		CreatedAt: time.Now(),
	}

	err := c.notebookRepository.Create(ctx, &notebook)
	if err != nil {
		return nil, err
	}

	return &dto.CreateNotebookResponse{
		Id: notebook.Id,
	}, nil
}

func (c *notebookService) Show(ctx context.Context, userId uuid.UUID, id uuid.UUID) (*dto.ShowNotebookResponse, error) {
	// ✅ FIX: Check ownership
	notebook, err := c.notebookRepository.GetById(ctx, id, userId)
	if err != nil {
		return nil, err
	}

	res := dto.ShowNotebookResponse{
		Id:        notebook.Id,
		Name:      notebook.Name,
		ParentId:  notebook.ParentId,
		CreatedAt: notebook.CreatedAt,
		UpdatedAt: notebook.UpdatedAt,
	}

	return &res, nil
}

func (c *notebookService) Update(ctx context.Context, userId uuid.UUID, req *dto.UpdateNotebookRequest) (*dto.UpdateNotebookResponse, error) {
	// ✅ FIX: Check ownership
	notebook, err := c.notebookRepository.GetById(ctx, req.Id, userId)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	notebook.Name = req.Name
	notebook.UpdatedAt = &now

	err = c.notebookRepository.Update(ctx, notebook)
	if err != nil {
		return nil, err
	}

	// ✅ FIX: Check ownership for notes
	notes, err := c.noteRepository.GetByNotebookIds(ctx, []uuid.UUID{notebook.Id}, userId)
	if err != nil {
		return nil, err
	}

	for _, note := range notes {
		msg := dto.PublishEmbedNoteMessage{
			NoteId: note.Id,
		}
		msgJson, err := json.Marshal(msg)
		if err != nil {
			return nil, err
		}
		err = c.publisherService.Publish(
			ctx,
			msgJson,
		)
		if err != nil {
			return nil, err
		}
	}

	return &dto.UpdateNotebookResponse{
		Id: notebook.Id,
	}, nil
}

func (c *notebookService) Delete(ctx context.Context, userId uuid.UUID, id uuid.UUID) error {
	// ✅ FIX: Check ownership
	_, err := c.notebookRepository.GetById(ctx, id, userId)
	if err != nil {
		return err
	}

	tx, err := c.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	notebookRepo := c.notebookRepository.UsingTx(ctx, tx)
	noteRepo := c.noteRepository.UsingTx(ctx, tx)
	noteEmbeddingRepo := c.noteEmbeddingRepository.UsingTx(ctx, tx)

	// ✅ FIX: Pass userId to delete
	err = notebookRepo.DeleteById(ctx, id, userId)
	if err != nil {
		return err
	}

	err = noteEmbeddingRepo.DeleteByNotebookId(ctx, id)
	if err != nil {
		return err
	}

	// ✅ FIX: Pass userId
	err = notebookRepo.NullifyParentById(ctx, id, userId)
	if err != nil {
		return err
	}

	// ✅ FIX: Pass userId
	err = noteRepo.DeleteByNotebookId(ctx, id, userId)
	if err != nil {
		return err
	}

	err = tx.Commit(ctx)
	if err != nil {
		return err
	}

	return nil
}

func (c *notebookService) MoveNotebook(ctx context.Context, userId uuid.UUID, req *dto.MoveNotebookRequest) (*dto.MoveNotebookResponse, error) {
	// ✅ FIX: Check ownership
	_, err := c.notebookRepository.GetById(ctx, req.Id, userId)
	if err != nil {
		return nil, err
	}
	if req.ParentId != nil {
		// ✅ FIX: Check ownership of parent too
		_, err = c.notebookRepository.GetById(ctx, *req.ParentId, userId)
		if err != nil {
			return nil, err
		}
	}

	// ✅ FIX: Update parent
	err = c.notebookRepository.UpdateParentId(ctx, req.Id, req.ParentId, userId)
	if err != nil {
		return nil, err
	}

	return &dto.MoveNotebookResponse{
		Id: req.Id,
	}, nil
}