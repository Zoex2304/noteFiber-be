// FILE: internal/repository/billing_repository.go
package repository

import (
	"ai-notetaking-be/internal/entity"
	"ai-notetaking-be/pkg/database"
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type IBillingRepository interface {
	Create(ctx context.Context, addr *entity.BillingAddress) error
}

type billingRepository struct {
	db database.DatabaseQueryer
}

func NewBillingRepository(db *pgxpool.Pool) IBillingRepository {
	return &billingRepository{db: db}
}

func (r *billingRepository) Create(ctx context.Context, addr *entity.BillingAddress) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO user_billing_addresses (
			id, user_id, first_name, last_name, email, phone, 
			address_line1, address_line2, city, state, postal_code, country, 
			is_default, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
	`, 
		addr.Id, addr.UserId, addr.FirstName, addr.LastName, addr.Email, addr.Phone,
		addr.AddressLine1, addr.AddressLine2, addr.City, addr.State, addr.PostalCode, addr.Country,
		addr.IsDefault, addr.CreatedAt, addr.UpdatedAt,
	)
	return err
}