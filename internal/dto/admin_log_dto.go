// FILE: internal/dto/admin_log_dto.go
package dto

import (
	"time"

	"github.com/google/uuid"
)

// --- Admin Transaction DTOs ---

type TransactionListResponse struct {
	Id                uuid.UUID `json:"id"`
	UserId            uuid.UUID `json:"user_id"`
	UserEmail         string    `json:"user_email"`
	PlanName          string    `json:"plan_name"`
	Amount            float64   `json:"amount"`
	Status            string    `json:"status"`        // active, inactive
	PaymentStatus     string    `json:"payment_status"` // paid, pending, failed
	TransactionDate   time.Time `json:"transaction_date"`
	MidtransOrderId   *string   `json:"midtrans_order_id"`
}

type TransactionDetailResponse struct {
	TransactionListResponse
	SnapRedirectUrl string `json:"snap_redirect_url,omitempty"`
}

// --- Admin Graph DTOs ---

type UserGrowthStats struct {
	Date  string `json:"date"`
	Count int    `json:"count"`
}

// --- System Log DTOs ---

type LogListResponse struct {
	Id        uuid.UUID `json:"id"`
	Level     string    `json:"level"`
	Module    string    `json:"module"`
	Message   string    `json:"message"`
	CreatedAt time.Time `json:"created_at"`
}

type LogDetailResponse struct {
	LogListResponse
	Details map[string]interface{} `json:"details"`
}

// --- OAuth DTOs ---

type OAuthLoginURLResponse struct {
	URL string `json:"url"`
}

type OAuthCallbackRequest struct {
	Code  string `json:"code" validate:"required"`
	State string `json:"state"`
}
