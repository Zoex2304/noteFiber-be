package repository

import (
	"ai-notetaking-be/internal/entity"
	"ai-notetaking-be/pkg/database"
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type IChatSessionRepository interface {
	UsingTx(ctx context.Context, tx database.DatabaseQueryer) IChatSessionRepository
	Create(ctx context.Context, chatSession *entity.ChatSession) error
	GetAll(ctx context.Context) ([]*entity.ChatSession, error)
	GetById(ctx context.Context, id uuid.UUID) (*entity.ChatSession, error)
	Update(ctx context.Context, chatSession *entity.ChatSession) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type chatSessionRepository struct {
	db database.DatabaseQueryer
}

func (n *chatSessionRepository) UsingTx(ctx context.Context, tx database.DatabaseQueryer) IChatSessionRepository {
	return &chatSessionRepository{
		db: tx,
	}
}

func (cs *chatSessionRepository) Create(ctx context.Context, chatSession *entity.ChatSession) error {
	_, err := cs.db.Exec(
		ctx,
		`INSERT INTO chat_session (id, title, created_at, updated_at, deleted_at, is_deleted) VALUES ($1, $2, $3, $4, $5, $6)`,
		chatSession.Id,
		chatSession.Title,
		chatSession.CreatedAt,
		chatSession.UpdatedAt,
		chatSession.DeletedAt,
		chatSession.IsDeleted,
	)
	if err != nil {
		return err
	}

	return nil
}

func (cs *chatSessionRepository) GetAll(ctx context.Context) ([]*entity.ChatSession, error) {
	rows, err := cs.db.Query(
		ctx,
		`SELECT id, title, created_at, updated_at, deleted_at, is_deleted FROM chat_session WHERE is_deleted = false ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, err
	}

	res := make([]*entity.ChatSession, 0)
	for rows.Next() {
		var chatSession entity.ChatSession

		err = rows.Scan(
			&chatSession.Id,
			&chatSession.Title,
			&chatSession.CreatedAt,
			&chatSession.UpdatedAt,
			&chatSession.DeletedAt,
			&chatSession.IsDeleted,
		)
		if err != nil {
			return nil, err
		}

		res = append(res, &chatSession)
	}

	return res, nil
}

func (cs *chatSessionRepository) GetById(ctx context.Context, id uuid.UUID) (*entity.ChatSession, error) {
	row := cs.db.QueryRow(
		ctx,
		`SELECT id, title, created_at, updated_at, deleted_at, is_deleted FROM chat_session WHERE id = $1 AND is_deleted = false`,
		id,
	)

	var result entity.ChatSession
	err := row.Scan(
		&result.Id,
		&result.Title,
		&result.CreatedAt,
		&result.UpdatedAt,
		&result.DeletedAt,
		&result.IsDeleted,
	)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

func (cs *chatSessionRepository) Update(ctx context.Context, chatSession *entity.ChatSession) error {
	_, err := cs.db.Exec(
		ctx,
		`UPDATE chat_session SET title = $1, updated_at = $2 WHERE id = $3`,
		chatSession.Title,
		chatSession.UpdatedAt,
		chatSession.Id,
	)
	if err != nil {
		return nil
	}

	return nil
}

func (cs *chatSessionRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := cs.db.Exec(
		ctx,
		`UPDATE chat_session SET deleted_at = $1, is_deleted = true WHERE id = $2`,
		time.Now(),
		id,
	)
	if err != nil {
		return nil
	}

	return nil
}

func NewChatSessionRepository(db *pgxpool.Pool) IChatSessionRepository {
	return &chatSessionRepository{
		db: db,
	}
}
