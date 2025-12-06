// FILE: internal/entity/billing_entity.go
package entity

import (
	"time"

	"github.com/google/uuid"
)

type BillingAddress struct {
	Id           uuid.UUID
	UserId       uuid.UUID
	FirstName    string
	LastName     string
	Email        string
	Phone        string
	AddressLine1 string
	AddressLine2 string
	City         string
	State        string
	PostalCode   string
	Country      string
	IsDefault    bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}