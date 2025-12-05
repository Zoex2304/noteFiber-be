// FILE: internal/entity/admin_entities.go
package entity

import (
	"time"

	"github.com/google/uuid"
)

type SystemLog struct {
	Id        uuid.UUID
	Level     string
	Module    string
	Message   string
	Details   map[string]interface{} // JSONB
	CreatedAt time.Time
}

// TransactionDetail is a projection for listing payments (Joined data)
type TransactionDetail struct {
	Id              uuid.UUID
	UserId          uuid.UUID
	UserEmail       string
	PlanName        string
	Amount          float64
	Status          string
	PaymentStatus   string
	MidtransOrderId *string
	CreatedAt       time.Time
}