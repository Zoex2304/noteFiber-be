// FILE: internal/repository/admin_log_repository.go
package repository

import (
	"ai-notetaking-be/internal/entity"
	"ai-notetaking-be/internal/pkg/serverutils"
	"ai-notetaking-be/pkg/database"
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// --- EXTEND IUserRepository ---

func (r *userRepository) SearchUsers(ctx context.Context, query string, limit, offset int) ([]*entity.User, error) {
	sql := `
		SELECT id, email, full_name, role, status, created_at 
		FROM users 
		WHERE email ILIKE $1 OR full_name ILIKE $1 
		ORDER BY created_at DESC 
		LIMIT $2 OFFSET $3
	`
	rows, err := r.db.Query(ctx, sql, "%"+query+"%", limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*entity.User
	for rows.Next() {
		var u entity.User
		var roleStr, statusStr string
		if err := rows.Scan(&u.Id, &u.Email, &u.FullName, &roleStr, &statusStr, &u.CreatedAt); err != nil {
			return nil, err
		}
		u.Role = entity.UserRole(roleStr)
		u.Status = entity.UserStatus(statusStr)
		users = append(users, &u)
	}
	return users, nil
}

func (r *userRepository) GetUserGrowth(ctx context.Context) ([]map[string]interface{}, error) {
	sql := `
		SELECT to_char(created_at, 'YYYY-MM-DD') as date, COUNT(*) as count 
		FROM users 
		WHERE created_at > NOW() - INTERVAL '30 days' 
		GROUP BY date 
		ORDER BY date ASC
	`
	rows, err := r.db.Query(ctx, sql)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []map[string]interface{}
	for rows.Next() {
		var date string
		var count int
		if err := rows.Scan(&date, &count); err != nil {
			return nil, err
		}
		stats = append(stats, map[string]interface{}{"date": date, "count": count})
	}
	return stats, nil
}

// --- NEW LogRepository ---

type ILogRepository interface {
	Create(ctx context.Context, level, module, message string, details map[string]interface{}) error
	GetAll(ctx context.Context, level string, limit, offset int) ([]*entity.SystemLog, error)
	GetById(ctx context.Context, id uuid.UUID) (*entity.SystemLog, error)
}

type logRepository struct {
	db database.DatabaseQueryer
}

func NewLogRepository(db *pgxpool.Pool) ILogRepository {
	return &logRepository{db: db}
}

func (r *logRepository) Create(ctx context.Context, level, module, message string, details map[string]interface{}) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO system_logs (level, module, message, details) VALUES ($1, $2, $3, $4)
	`, level, module, message, details)
	return err
}

func (r *logRepository) GetAll(ctx context.Context, level string, limit, offset int) ([]*entity.SystemLog, error) {
	baseQuery := `SELECT id, level, module, message, created_at FROM system_logs`
	var args []interface{}
	counter := 1

	if level != "" {
		baseQuery += fmt.Sprintf(" WHERE level = $%d", counter)
		args = append(args, level)
		counter++
	}

	baseQuery += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", counter, counter+1)
	args = append(args, limit, offset)

	rows, err := r.db.Query(ctx, baseQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []*entity.SystemLog
	for rows.Next() {
		var l entity.SystemLog
		if err := rows.Scan(&l.Id, &l.Level, &l.Module, &l.Message, &l.CreatedAt); err != nil {
			return nil, err
		}
		logs = append(logs, &l)
	}
	return logs, nil
}

func (r *logRepository) GetById(ctx context.Context, id uuid.UUID) (*entity.SystemLog, error) {
	var l entity.SystemLog
	err := r.db.QueryRow(ctx, `SELECT id, level, module, message, details, created_at FROM system_logs WHERE id = $1`, id).
		Scan(&l.Id, &l.Level, &l.Module, &l.Message, &l.Details, &l.CreatedAt)
	if err != nil {
		return nil, serverutils.ErrNotFound
	}
	return &l, nil
}

// REMOVED: GetTransactions implementation (It is now in subscription_repository.go)