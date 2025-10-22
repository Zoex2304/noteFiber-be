package repository

import (
	"ai-notetaking-be/internal/entity"
	"ai-notetaking-be/pkg/database"
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type IChatMessageRepository interface {
	UsingTx(ctx context.Context, tx database.DatabaseQueryer) IChatMessageRepository
	Create(ctx context.Context, chatMessage *entity.ChatMessage) error
	GetByChatSessionId(ctx context.Context, chatSessionId uuid.UUID) ([]*entity.ChatMessage, error)
	DeleteByChatSessionId(ctx context.Context, chatSessionId uuid.UUID) error
}

type chatMessageRepository struct {
	db database.DatabaseQueryer
}

func (n *chatMessageRepository) UsingTx(ctx context.Context, tx database.DatabaseQueryer) IChatMessageRepository {
	return &chatMessageRepository{
		db: tx,
	}
}

func (cs *chatMessageRepository) Create(ctx context.Context, chatMessage *entity.ChatMessage) error {
	_, err := cs.db.Exec(
		ctx,
		`INSERT INTO chat_message (id, role, chat, chat_session_id, created_at, updated_at, deleted_at, is_deleted) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		chatMessage.Id,
		chatMessage.Role,
		chatMessage.Chat,
		chatMessage.ChatSessionId,
		chatMessage.CreatedAt,
		chatMessage.UpdatedAt,
		chatMessage.DeletedAt,
		chatMessage.IsDeleted,
	)
	if err != nil {
		return err
	}

	return nil
}

func (cs *chatMessageRepository) GetByChatSessionId(ctx context.Context, chatSessionId uuid.UUID) ([]*entity.ChatMessage, error) {
	rows, err := cs.db.Query(
		ctx,
		`SELECT id, role, chat, chat_session_id, created_at, updated_at, deleted_at, is_deleted FROM chat_message WHERE chat_session_id = $1 AND is_deleted = false ORDER BY created_at ASC`,
		chatSessionId,
	)
	if err != nil {
		return nil, err
	}

	res := make([]*entity.ChatMessage, 0)
	for rows.Next() {
		var chatMessage entity.ChatMessage

		err = rows.Scan(
			&chatMessage.Id,
			&chatMessage.Role,
			&chatMessage.Chat,
			&chatMessage.ChatSessionId,
			&chatMessage.CreatedAt,
			&chatMessage.UpdatedAt,
			&chatMessage.DeletedAt,
			&chatMessage.IsDeleted,
		)
		if err != nil {
			return nil, err
		}

		res = append(res, &chatMessage)
	}

	return res, nil
}

func (cs *chatMessageRepository) DeleteByChatSessionId(ctx context.Context, chatSessionId uuid.UUID) error {
	_, err := cs.db.Exec(
		ctx,
		`UPDATE chat_message SET is_deleted = true, deleted_at = $1 WHERE chat_session_id = $2`,
		time.Now(),
		chatSessionId,
	)
	if err != nil {
		return err
	}

	return nil
}

func NewChatMessageRepository(db *pgxpool.Pool) IChatMessageRepository {
	return &chatMessageRepository{
		db: db,
	}
}
