// FILE: internal/repository/user_repository.go
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

type IUserRepository interface {
	UsingTx(ctx context.Context, tx database.DatabaseQueryer) IUserRepository
	Create(ctx context.Context, user *entity.User) error
	GetByEmail(ctx context.Context, email string) (*entity.User, error)
	GetById(ctx context.Context, id uuid.UUID) (*entity.User, error)
	CreatePasswordResetToken(ctx context.Context, token *entity.PasswordResetToken) error
	GetPasswordResetToken(ctx context.Context, tokenString string) (*entity.PasswordResetToken, error)
	MarkTokenUsed(ctx context.Context, id uuid.UUID) error
	UpdatePassword(ctx context.Context, userId uuid.UUID, hash string) error
}

type userRepository struct {
	db database.DatabaseQueryer
}

func NewUserRepository(db *pgxpool.Pool) IUserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) UsingTx(ctx context.Context, tx database.DatabaseQueryer) IUserRepository {
	return &userRepository{db: tx}
}

func (r *userRepository) Create(ctx context.Context, user *entity.User) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO users (id, email, password_hash, full_name, role, status, email_verified, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, user.Id, user.Email, user.PasswordHash, user.FullName, user.Role, user.Status, user.EmailVerified, user.CreatedAt, user.UpdatedAt)
	return err
}

func (r *userRepository) GetByEmail(ctx context.Context, email string) (*entity.User, error) {
	row := r.db.QueryRow(ctx, `SELECT id, email, password_hash, full_name, role, status FROM users WHERE email = $1`, email)
	var user entity.User
	err := row.Scan(&user.Id, &user.Email, &user.PasswordHash, &user.FullName, &user.Role, &user.Status)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil // Return nil if not found, let service handle 404
	}
	return &user, err
}

func (r *userRepository) GetById(ctx context.Context, id uuid.UUID) (*entity.User, error) {
	row := r.db.QueryRow(ctx, `SELECT id, email, full_name, role FROM users WHERE id = $1`, id)
	var user entity.User
	err := row.Scan(&user.Id, &user.Email, &user.FullName, &user.Role)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, serverutils.ErrNotFound
	}
	return &user, err
}

func (r *userRepository) CreatePasswordResetToken(ctx context.Context, token *entity.PasswordResetToken) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO password_reset_tokens (id, user_id, token, expires_at, used, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, token.Id, token.UserId, token.Token, token.ExpiresAt, token.Used, token.CreatedAt)
	return err
}

func (r *userRepository) GetPasswordResetToken(ctx context.Context, tokenString string) (*entity.PasswordResetToken, error) {
	row := r.db.QueryRow(ctx, `SELECT id, user_id, token, expires_at, used FROM password_reset_tokens WHERE token = $1`, tokenString)
	var t entity.PasswordResetToken
	err := row.Scan(&t.Id, &t.UserId, &t.Token, &t.ExpiresAt, &t.Used)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, serverutils.ErrNotFound
	}
	return &t, err
}

func (r *userRepository) MarkTokenUsed(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `UPDATE password_reset_tokens SET used = true WHERE id = $1`, id)
	return err
}

func (r *userRepository) UpdatePassword(ctx context.Context, userId uuid.UUID, hash string) error {
	_, err := r.db.Exec(ctx, `UPDATE users SET password_hash = $1, updated_at = $2 WHERE id = $3`, hash, time.Now(), userId)
	return err
}