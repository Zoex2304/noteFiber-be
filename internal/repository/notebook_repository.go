// internal\repository\notebook_repository.go
package repository

import (
	"ai-notetaking-be/internal/entity"
	"ai-notetaking-be/internal/pkg/serverutils"
	"ai-notetaking-be/pkg/database"
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type INotebookRepository interface {
	UsingTx(ctx context.Context, tx database.DatabaseQueryer) INotebookRepository
	GetAll(ctx context.Context, userId uuid.UUID) ([]*entity.Notebook, error)
	Create(ctx context.Context, notebook *entity.Notebook) error
	GetById(ctx context.Context, id uuid.UUID, userId uuid.UUID) (*entity.Notebook, error)
	GetByIdGlobal(ctx context.Context, id uuid.UUID) (*entity.Notebook, error) // ✅ ADDED: For ConsumerService
	Update(ctx context.Context, notebook *entity.Notebook) error
	DeleteById(ctx context.Context, id uuid.UUID, userId uuid.UUID) error
	NullifyParentById(ctx context.Context, parentId uuid.UUID, userId uuid.UUID) error
	UpdateParentId(ctx context.Context, id uuid.UUID, parentId *uuid.UUID, userId uuid.UUID) error
}

type notebookRepository struct {
	db database.DatabaseQueryer
}

func (n *notebookRepository) UsingTx(ctx context.Context, tx database.DatabaseQueryer) INotebookRepository {
	return &notebookRepository{
		db: tx,
	}
}

// ✅ SCOPED: Filter by User ID
func (n *notebookRepository) GetAll(ctx context.Context, userId uuid.UUID) ([]*entity.Notebook, error) {
	rows, err := n.db.Query(
		ctx,
		`SELECT id, name, parent_id, user_id, created_at, updated_at FROM notebook WHERE is_deleted = false AND user_id = $1 ORDER BY name ASC`,
		userId,
	)
	if err != nil {
		return nil, err
	}

	result := make([]*entity.Notebook, 0)
	for rows.Next() {
		var notebook entity.Notebook
		err = rows.Scan(
			&notebook.Id,
			&notebook.Name,
			&notebook.ParentId,
			&notebook.UserId,
			&notebook.CreatedAt,
			&notebook.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		result = append([]*entity.Notebook{&notebook}, result...)
	}

	return result, nil
}

func (n *notebookRepository) Create(ctx context.Context, notebook *entity.Notebook) error {
	_, err := n.db.Exec(
		ctx,
		`INSERT INTO notebook (id, name, parent_id, user_id, created_at, updated_at, deleted_at, is_deleted) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		notebook.Id,
		notebook.Name,
		notebook.ParentId,
		notebook.UserId,
		notebook.CreatedAt,
		notebook.UpdatedAt,
		notebook.DeletedAt,
		notebook.IsDeleted,
	)
	if err != nil {
		return err
	}

	return nil
}

// ✅ SCOPED: Verify Ownership
func (n *notebookRepository) GetById(ctx context.Context, id uuid.UUID, userId uuid.UUID) (*entity.Notebook, error) {
	row := n.db.QueryRow(
		ctx,
		`SELECT id, name, parent_id, user_id, created_at, updated_at, deleted_at, is_deleted FROM notebook n WHERE n.is_deleted = false AND n.id = $1 AND n.user_id = $2`,
		id, userId,
	)
	var notebook entity.Notebook

	err := row.Scan(
		&notebook.Id,
		&notebook.Name,
		&notebook.ParentId,
		&notebook.UserId,
		&notebook.CreatedAt,
		&notebook.UpdatedAt,
		&notebook.DeletedAt,
		&notebook.IsDeleted,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, serverutils.ErrNotFound
		}
		return nil, err
	}

	return &notebook, nil
}

// ✅ ADDED: Global access for background workers
func (n *notebookRepository) GetByIdGlobal(ctx context.Context, id uuid.UUID) (*entity.Notebook, error) {
	row := n.db.QueryRow(
		ctx,
		`SELECT id, name, parent_id, user_id, created_at, updated_at, deleted_at, is_deleted FROM notebook n WHERE n.is_deleted = false AND n.id = $1`,
		id,
	)
	var notebook entity.Notebook

	err := row.Scan(
		&notebook.Id,
		&notebook.Name,
		&notebook.ParentId,
		&notebook.UserId,
		&notebook.CreatedAt,
		&notebook.UpdatedAt,
		&notebook.DeletedAt,
		&notebook.IsDeleted,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, serverutils.ErrNotFound
		}
		return nil, err
	}

	return &notebook, nil
}

func (n *notebookRepository) Update(ctx context.Context, notebook *entity.Notebook) error {
	_, err := n.db.Exec(
		ctx,
		`
		UPDATE notebook SET
			name = $1,
			parent_id = $2,
			updated_at = $3
		WHERE id = $4 AND user_id = $5
		`,
		notebook.Name,
		notebook.ParentId,
		notebook.UpdatedAt,
		notebook.Id,
		notebook.UserId,
	)
	if err != nil {
		return err
	}

	return nil
}

func (n *notebookRepository) DeleteById(ctx context.Context, id uuid.UUID, userId uuid.UUID) error {
	_, err := n.db.Exec(
		ctx,
		`
		UPDATE notebook SET is_deleted = true, deleted_at = $1 WHERE id = $2 AND user_id = $3
		`,
		time.Now(),
		id,
		userId,
	)
	if err != nil {
		return err
	}

	return nil
}

func (n *notebookRepository) NullifyParentById(ctx context.Context, parentId uuid.UUID, userId uuid.UUID) error {
	_, err := n.db.Exec(
		ctx,
		`
		UPDATE notebook SET parent_id = null, updated_at = $1 WHERE parent_id = $2 AND user_id = $3
		`,
		time.Now(),
		parentId,
		userId,
	)
	if err != nil {
		return err
	}

	return nil
}

func (n *notebookRepository) UpdateParentId(ctx context.Context, id uuid.UUID, parentId *uuid.UUID, userId uuid.UUID) error {
	_, err := n.db.Exec(
		ctx,
		`
		UPDATE notebook SET parent_id = $1, updated_at = $2 WHERE id = $3 AND user_id = $4
		`,
		parentId,
		time.Now(),
		id,
		userId,
	)
	if err != nil {
		return err
	}

	return nil
}

func NewNotebookRepository(db *pgxpool.Pool) INotebookRepository {
	return &notebookRepository{
		db: db,
	}
}