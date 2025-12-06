// FILE: internal/dto/auth_payment_dto.go
package dto

import (
	"github.com/google/uuid"
)

// --- Auth DTOs ---

type RegisterRequest struct {
	FullName string `json:"full_name" validate:"required,min=3"`
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
}

type VerifyEmailRequest struct {
	Email string `json:"email" validate:"required,email"`
	Token string `json:"token" validate:"required,len=6"`
}

type RegisterResponse struct {
	Id    uuid.UUID `json:"id"`
	Email string    `json:"email"`
}

type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type LoginResponse struct {
	AccessToken string  `json:"access_token"`
	User        UserDTO `json:"user"`
}

type UserDTO struct {
	Id       uuid.UUID `json:"id"`
	Email    string    `json:"email"`
	FullName string    `json:"full_name"`
	Role     string    `json:"role"`
}

type ForgotPasswordRequest struct {
	Email string `json:"email" validate:"required,email"`
}

type ResetPasswordRequest struct {
	Token           string `json:"token" validate:"required"`
	NewPassword     string `json:"new_password" validate:"required,min=8"`
	ConfirmPassword string `json:"confirm_password" validate:"required,eqfield=NewPassword"`
}

// --- Payment DTOs ---

type PlanResponse struct {
	Id          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Slug        string    `json:"slug"`
	Price       float64   `json:"price"`
	Description string    `json:"description"`
	Features    []string  `json:"features"`
}

type CheckoutRequest struct {
	PlanId       uuid.UUID `json:"plan_id" validate:"required"`
	
	// Billing Details
	FirstName    string    `json:"first_name" validate:"required"`
	LastName     string    `json:"last_name" validate:"required"`
	Email        string    `json:"email" validate:"required,email"`
	Phone        string    `json:"phone" validate:"omitempty"`
	AddressLine1 string    `json:"address_line1" validate:"required"`
	AddressLine2 string    `json:"address_line2"`
	City         string    `json:"city" validate:"required"`
	State        string    `json:"state" validate:"required"`
	PostalCode   string    `json:"postal_code" validate:"required"`
	Country      string    `json:"country" validate:"required"`
}

type CheckoutResponse struct {
	SubscriptionId  uuid.UUID `json:"subscription_id"`
	SnapRedirectUrl string    `json:"snap_redirect_url"`
	SnapToken       string    `json:"snap_token"`
}

type MidtransWebhookRequest struct {
	TransactionStatus string `json:"transaction_status"`
	OrderId           string `json:"order_id"`
	FraudStatus       string `json:"fraud_status"`
}