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

	// OTP Methods
	CreateVerificationToken(ctx context.Context, token *entity.EmailVerificationToken) error
	GetVerificationToken(ctx context.Context, userId uuid.UUID, token string) (*entity.EmailVerificationToken, error)
	DeleteVerificationToken(ctx context.Context, id uuid.UUID) error
	ActivateUser(ctx context.Context, userId uuid.UUID) error

	// New Profile & Admin Methods
	Update(ctx context.Context, user *entity.User) error
	Delete(ctx context.Context, id uuid.UUID) error
	GetAll(ctx context.Context, limit, offset int) ([]*entity.User, error)
	Count(ctx context.Context) (int, error)
	CountByStatus(ctx context.Context, status entity.UserStatus) (int, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status entity.UserStatus) error

	// Search & Stats (Implemented in admin_log_repository.go)
	SearchUsers(ctx context.Context, query string, limit, offset int) ([]*entity.User, error)
	GetUserGrowth(ctx context.Context) ([]map[string]interface{}, error)
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

// ... Existing methods (Create, GetByEmail, GetById, OTP methods, etc.) are assumed to be here ...
// ... I am appending the new methods below ...

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
		return nil, nil
	}
	return &user, err
}

func (r *userRepository) GetById(ctx context.Context, id uuid.UUID) (*entity.User, error) {
	row := r.db.QueryRow(ctx, `SELECT id, email, full_name, role, status, ai_daily_usage, created_at FROM users WHERE id = $1`, id)
	var user entity.User
	err := row.Scan(&user.Id, &user.Email, &user.FullName, &user.Role, &user.Status, &user.AiDailyUsage, &user.CreatedAt)
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

func (r *userRepository) CreateVerificationToken(ctx context.Context, token *entity.EmailVerificationToken) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO email_verification_tokens (id, user_id, token, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`, token.Id, token.UserId, token.Token, token.ExpiresAt, token.CreatedAt)
	return err
}

func (r *userRepository) GetVerificationToken(ctx context.Context, userId uuid.UUID, token string) (*entity.EmailVerificationToken, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, user_id, token, expires_at, created_at 
		FROM email_verification_tokens 
		WHERE user_id = $1 AND token = $2
	`, userId, token)

	var t entity.EmailVerificationToken
	err := row.Scan(&t.Id, &t.UserId, &t.Token, &t.ExpiresAt, &t.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, serverutils.ErrNotFound
	}
	return &t, err
}

func (r *userRepository) DeleteVerificationToken(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM email_verification_tokens WHERE id = $1`, id)
	return err
}

func (r *userRepository) ActivateUser(ctx context.Context, userId uuid.UUID) error {
	_, err := r.db.Exec(ctx, `
		UPDATE users 
		SET status = 'active', email_verified = true, email_verified_at = $1, updated_at = $2 
		WHERE id = $3
	`, time.Now(), time.Now(), userId)
	return err
}

// --- NEW METHODS ---

func (r *userRepository) Update(ctx context.Context, user *entity.User) error {
	_, err := r.db.Exec(ctx, `UPDATE users SET full_name = $1, updated_at = $2 WHERE id = $3`, user.FullName, time.Now(), user.Id)
	return err
}

func (r *userRepository) Delete(ctx context.Context, id uuid.UUID) error {
	// Soft delete or Hard delete? Usually hard delete for "Delete Account" request unless regulated otherwise.
	// Schema doesn't have deleted_at for users, assuming Hard Delete.
	_, err := r.db.Exec(ctx, `DELETE FROM users WHERE id = $1`, id)
	return err
}

func (r *userRepository) GetAll(ctx context.Context, limit, offset int) ([]*entity.User, error) {
	rows, err := r.db.Query(ctx, `SELECT id, email, full_name, role, status, created_at FROM users ORDER BY created_at DESC LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*entity.User
	for rows.Next() {
		var u entity.User
		err := rows.Scan(&u.Id, &u.Email, &u.FullName, &u.Role, &u.Status, &u.CreatedAt)
		if err != nil {
			return nil, err
		}
		users = append(users, &u)
	}
	return users, nil
}

func (r *userRepository) Count(ctx context.Context) (int, error) {
	var count int
	err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM users`).Scan(&count)
	return count, err
}

func (r *userRepository) CountByStatus(ctx context.Context, status entity.UserStatus) (int, error) {
	var count int
	err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM users WHERE status = $1`, status).Scan(&count)
	return count, err
}

func (r *userRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status entity.UserStatus) error {
	_, err := r.db.Exec(ctx, `UPDATE users SET status = $1, updated_at = $2 WHERE id = $3`, status, time.Now(), id)
	return err
}