package repository

import (
	"ai-notetaking-be/internal/entity"
	"ai-notetaking-be/internal/pkg/serverutils"
	"ai-notetaking-be/pkg/database"
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type INoteRepository interface {
	UsingTx(ctx context.Context, tx database.DatabaseQueryer) INoteRepository
	Create(ctx context.Context, note *entity.Note) error
	GetById(ctx context.Context, id uuid.UUID) (*entity.Note, error)
	GetByNotebookIds(ctx context.Context, ids []uuid.UUID) ([]*entity.Note, error)
	Update(ctx context.Context, note *entity.Note) error
	Delete(ctx context.Context, id uuid.UUID) error
	DeleteByNotebookId(ctx context.Context, notebookId uuid.UUID) error
	GetByIds(ctx context.Context, ids []uuid.UUID) ([]*entity.Note, error)
}

type noteRepository struct {
	db database.DatabaseQueryer
}

func (n *noteRepository) UsingTx(ctx context.Context, tx database.DatabaseQueryer) INoteRepository {
	return &noteRepository{
		db: tx,
	}
}

func (n *noteRepository) Create(ctx context.Context, note *entity.Note) error {
	_, err := n.db.Exec(
		ctx,
		`INSERT INTO note (id, title, content, notebook_id, created_at, updated_at, deleted_at, is_deleted) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		note.Id,
		note.Title,
		note.Content,
		note.NotebookId,
		note.CreatedAt,
		note.UpdatedAt,
		note.DeletedAt,
		note.IsDeleted,
	)
	if err != nil {
		return err
	}

	return nil
}

func (n *noteRepository) GetById(ctx context.Context, id uuid.UUID) (*entity.Note, error) {
	row := n.db.QueryRow(
		ctx,
		`SELECT id, title, content, notebook_id, created_at, updated_at FROM note WHERE id = $1 AND is_deleted = false`,
		id,
	)

	var note entity.Note
	err := row.Scan(
		&note.Id,
		&note.Title,
		&note.Content,
		&note.NotebookId,
		&note.CreatedAt,
		&note.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, serverutils.ErrNotFound
		}
		return nil, err
	}

	return &note, nil
}

func (n *noteRepository) Update(ctx context.Context, note *entity.Note) error {
	_, err := n.db.Exec(
		ctx,
		`
		UPDATE note SET
			title = $1,
			content = $2,
			notebook_id = $3,
			updated_at = $4
		WHERE id = $5
		`,
		note.Title,
		note.Content,
		note.NotebookId,
		note.UpdatedAt,
		note.Id,
	)
	if err != nil {
		return err
	}

	return nil
}

func (n *noteRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := n.db.Exec(
		ctx,
		`
		UPDATE note SET
			deleted_at = $1,
			is_deleted = true
		WHERE id = $2
		`,
		time.Now(),
		id,
	)
	if err != nil {
		return err
	}

	return nil
}

func (n *noteRepository) DeleteByNotebookId(ctx context.Context, notebookId uuid.UUID) error {
	_, err := n.db.Exec(
		ctx,
		`UPDATE note SET deleted_at = $1, is_deleted = true WHERE notebook_id = $2`,
		time.Now(),
		notebookId,
	)
	if err != nil {
		return err
	}

	return nil
}

func (n *noteRepository) GetByNotebookIds(ctx context.Context, ids []uuid.UUID) ([]*entity.Note, error) {
	if len(ids) == 0 {
		return make([]*entity.Note, 0), nil
	}

	idStr := make([]string, 0)
	for _, id := range ids {
		idStr = append(idStr, fmt.Sprintf("'%s'", id.String()))
	}
	idSqlFormat := strings.Join(idStr, ", ")

	rows, err := n.db.Query(
		ctx,
		fmt.Sprintf(`SELECT id, title, content, notebook_id, created_at, updated_at FROM note WHERE notebook_id IN (%s) AND is_deleted = false`, idSqlFormat),
	)
	if err != nil {
		return nil, err
	}

	result := make([]*entity.Note, 0)
	for rows.Next() {
		var note entity.Note

		err = rows.Scan(
			&note.Id,
			&note.Title,
			&note.Content,
			&note.NotebookId,
			&note.CreatedAt,
			&note.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		result = append(result, &note)
	}

	return result, nil
}

func (n *noteRepository) GetByIds(ctx context.Context, ids []uuid.UUID) ([]*entity.Note, error) {
	if len(ids) == 0 {
		return make([]*entity.Note, 0), nil
	}

	idStr := make([]string, 0)
	for _, id := range ids {
		idStr = append(idStr, fmt.Sprintf("'%s'", id.String()))
	}
	idSqlFormat := strings.Join(idStr, ", ")

	rows, err := n.db.Query(
		ctx,
		fmt.Sprintf(`SELECT id, title, content, notebook_id, created_at, updated_at FROM note WHERE id IN (%s) AND is_deleted = false`, idSqlFormat),
	)
	if err != nil {
		return nil, err
	}

	result := make([]*entity.Note, 0)
	for rows.Next() {
		var note entity.Note

		err = rows.Scan(
			&note.Id,
			&note.Title,
			&note.Content,
			&note.NotebookId,
			&note.CreatedAt,
			&note.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		result = append(result, &note)
	}

	return result, nil
}

func NewNoteRepository(db *pgxpool.Pool) INoteRepository {
	return &noteRepository{
		db: db,
	}
}
