// FILE: internal/repository/subscription_repository.go
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

type ISubscriptionRepository interface {
	GetAllPlans(ctx context.Context) ([]*entity.SubscriptionPlan, error)
	GetPlanById(ctx context.Context, id uuid.UUID) (*entity.SubscriptionPlan, error)
	CreateSubscription(ctx context.Context, sub *entity.UserSubscription) error
	GetSubscriptionById(ctx context.Context, id uuid.UUID) (*entity.UserSubscription, error)
	UpdateSubscriptionStatus(ctx context.Context, id uuid.UUID, status entity.SubscriptionStatus, paymentStatus entity.PaymentStatus) error
}

type subscriptionRepository struct {
	db database.DatabaseQueryer
}

func NewSubscriptionRepository(db *pgxpool.Pool) ISubscriptionRepository {
	return &subscriptionRepository{db: db}
}

func (r *subscriptionRepository) GetAllPlans(ctx context.Context) ([]*entity.SubscriptionPlan, error) {
	rows, err := r.db.Query(ctx, `SELECT id, name, slug, description, price, billing_period, max_notes, semantic_search_enabled, ai_chat_enabled FROM subscription_plans ORDER BY price ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var plans []*entity.SubscriptionPlan
	for rows.Next() {
		var p entity.SubscriptionPlan
		var bp string
		err := rows.Scan(&p.Id, &p.Name, &p.Slug, &p.Description, &p.Price, &bp, &p.MaxNotes, &p.SemanticSearchEnabled, &p.AiChatEnabled)
		if err != nil {
			return nil, err
		}
		p.BillingPeriod = entity.BillingPeriod(bp)
		plans = append(plans, &p)
	}
	return plans, nil
}

func (r *subscriptionRepository) GetPlanById(ctx context.Context, id uuid.UUID) (*entity.SubscriptionPlan, error) {
	row := r.db.QueryRow(ctx, `SELECT id, name, price, billing_period FROM subscription_plans WHERE id = $1`, id)
	var p entity.SubscriptionPlan
	var bp string
	err := row.Scan(&p.Id, &p.Name, &p.Price, &bp)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, serverutils.ErrNotFound
	}
	p.BillingPeriod = entity.BillingPeriod(bp)
	return &p, err
}

func (r *subscriptionRepository) CreateSubscription(ctx context.Context, sub *entity.UserSubscription) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO user_subscriptions (id, user_id, plan_id, status, current_period_start, current_period_end, payment_status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, sub.Id, sub.UserId, sub.PlanId, sub.Status, sub.CurrentPeriodStart, sub.CurrentPeriodEnd, sub.PaymentStatus, sub.CreatedAt, sub.UpdatedAt)
	return err
}

func (r *subscriptionRepository) GetSubscriptionById(ctx context.Context, id uuid.UUID) (*entity.UserSubscription, error) {
	row := r.db.QueryRow(ctx, `SELECT id, user_id, plan_id, status, payment_status FROM user_subscriptions WHERE id = $1`, id)
	var s entity.UserSubscription
	err := row.Scan(&s.Id, &s.UserId, &s.PlanId, &s.Status, &s.PaymentStatus)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, serverutils.ErrNotFound
	}
	return &s, err
}

func (r *subscriptionRepository) UpdateSubscriptionStatus(ctx context.Context, id uuid.UUID, status entity.SubscriptionStatus, paymentStatus entity.PaymentStatus) error {
	_, err := r.db.Exec(ctx, `UPDATE user_subscriptions SET status = $1, payment_status = $2, updated_at = $3 WHERE id = $4`, status, paymentStatus, time.Now(), id)
	return err
}