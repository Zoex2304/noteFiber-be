// FILE: internal/repository/subscription_repository.go
package repository

import (
	"ai-notetaking-be/internal/entity"
	"ai-notetaking-be/internal/pkg/serverutils"
	"ai-notetaking-be/pkg/database"
	"context"
	"errors"
	"fmt"
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
	GetActiveByUserId(ctx context.Context, userId uuid.UUID) (*entity.UserSubscription, *entity.SubscriptionPlan, error)
	CancelSubscription(ctx context.Context, id uuid.UUID) error
	GetTotalRevenue(ctx context.Context) (float64, error)
	CountActiveSubscribers(ctx context.Context) (int, error)
    GetTransactions(ctx context.Context, status string, limit, offset int) ([]*entity.TransactionDetail, error)
}

type subscriptionRepository struct {
	db database.DatabaseQueryer
}

func NewSubscriptionRepository(db *pgxpool.Pool) ISubscriptionRepository {
	return &subscriptionRepository{db: db}
}

func (r *subscriptionRepository) GetAllPlans(ctx context.Context) ([]*entity.SubscriptionPlan, error) {
	// UPDATED: Added tax_rate to SELECT
	rows, err := r.db.Query(ctx, `SELECT id, name, slug, description, price, tax_rate, billing_period, max_notes, semantic_search_enabled, ai_chat_enabled, ai_daily_credit_limit FROM subscription_plans ORDER BY price ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var plans []*entity.SubscriptionPlan
	for rows.Next() {
		var p entity.SubscriptionPlan
		var bp string
		// UPDATED: Added &p.TaxRate to Scan
		err := rows.Scan(&p.Id, &p.Name, &p.Slug, &p.Description, &p.Price, &p.TaxRate, &bp, &p.MaxNotes, &p.SemanticSearchEnabled, &p.AiChatEnabled, &p.AiDailyCreditLimit)
		if err != nil {
			return nil, err
		}
		p.BillingPeriod = entity.BillingPeriod(bp)
		plans = append(plans, &p)
	}
	return plans, nil
}

func (r *subscriptionRepository) GetPlanById(ctx context.Context, id uuid.UUID) (*entity.SubscriptionPlan, error) {
	// UPDATED: Added tax_rate to SELECT
	row := r.db.QueryRow(ctx, `SELECT id, name, price, tax_rate, billing_period FROM subscription_plans WHERE id = $1`, id)
	var p entity.SubscriptionPlan
	var bp string
	// UPDATED: Added &p.TaxRate to Scan
	err := row.Scan(&p.Id, &p.Name, &p.Price, &p.TaxRate, &bp)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, serverutils.ErrNotFound
	}
	p.BillingPeriod = entity.BillingPeriod(bp)
	return &p, err
}

func (r *subscriptionRepository) CreateSubscription(ctx context.Context, sub *entity.UserSubscription) error {
	// UPDATED: Added billing_address_id field
	_, err := r.db.Exec(ctx, `
		INSERT INTO user_subscriptions (id, user_id, plan_id, billing_address_id, status, current_period_start, current_period_end, payment_status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`, sub.Id, sub.UserId, sub.PlanId, sub.BillingAddressId, sub.Status, sub.CurrentPeriodStart, sub.CurrentPeriodEnd, sub.PaymentStatus, sub.CreatedAt, sub.UpdatedAt)
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
	ct, err := r.db.Exec(ctx, `UPDATE user_subscriptions SET status = $1, payment_status = $2, updated_at = $3 WHERE id = $4`, status, paymentStatus, time.Now(), id)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("no subscription found with id %s to update", id)
	}
	return nil
}

func (r *subscriptionRepository) GetActiveByUserId(ctx context.Context, userId uuid.UUID) (*entity.UserSubscription, *entity.SubscriptionPlan, error) {
	query := `
		SELECT 
			us.id, us.status, us.current_period_end, 
			sp.name, sp.ai_daily_credit_limit 
		FROM user_subscriptions us
		JOIN subscription_plans sp ON us.plan_id = sp.id
		WHERE us.user_id = $1 AND us.status = 'active' AND us.current_period_end > NOW()
		ORDER BY us.created_at DESC
		LIMIT 1
	`
	row := r.db.QueryRow(ctx, query, userId)

	var sub entity.UserSubscription
	var plan entity.SubscriptionPlan
	
	err := row.Scan(&sub.Id, &sub.Status, &sub.CurrentPeriodEnd, &plan.Name, &plan.AiDailyCreditLimit)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil, nil
	}
	if err != nil {
		return nil, nil, err
	}
	return &sub, &plan, nil
}

func (r *subscriptionRepository) CancelSubscription(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `UPDATE user_subscriptions SET status = 'canceled', updated_at = $1 WHERE id = $2`, time.Now(), id)
	return err
}

func (r *subscriptionRepository) GetTotalRevenue(ctx context.Context) (float64, error) {
	query := `
		SELECT COALESCE(SUM(sp.price), 0)
		FROM user_subscriptions us
		JOIN subscription_plans sp ON us.plan_id = sp.id
		WHERE us.payment_status = 'success'
	`
	var revenue float64
	err := r.db.QueryRow(ctx, query).Scan(&revenue)
	return revenue, err
}

func (r *subscriptionRepository) CountActiveSubscribers(ctx context.Context) (int, error) {
	var count int
	err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM user_subscriptions WHERE status = 'active'`).Scan(&count)
	return count, err
}

func (r *subscriptionRepository) GetTransactions(ctx context.Context, status string, limit, offset int) ([]*entity.TransactionDetail, error) {
	sql := `
		SELECT us.id, us.user_id, u.email, sp.name, sp.price, us.status, us.payment_status, us.created_at, us.midtrans_transaction_id
		FROM user_subscriptions us
		JOIN users u ON us.user_id = u.id
		JOIN subscription_plans sp ON us.plan_id = sp.id
	`
	var args []interface{}
	counter := 1

	if status != "" {
		sql += fmt.Sprintf(" WHERE us.payment_status = $%d", counter)
		args = append(args, status)
		counter++
	}

	sql += fmt.Sprintf(" ORDER BY us.created_at DESC LIMIT $%d OFFSET $%d", counter, counter+1)
	args = append(args, limit, offset)

	rows, err := r.db.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var txs []*entity.TransactionDetail
	for rows.Next() {
		var t entity.TransactionDetail
		var statusStr, payStatusStr string
		if err := rows.Scan(&t.Id, &t.UserId, &t.UserEmail, &t.PlanName, &t.Amount, &statusStr, &payStatusStr, &t.CreatedAt, &t.MidtransOrderId); err != nil {
			return nil, err
		}
		t.Status = statusStr
		t.PaymentStatus = payStatusStr
		txs = append(txs, &t)
	}
	return txs, nil
}