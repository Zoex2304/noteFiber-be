// FILE: internal/dto/user_dto.go
package dto

import (
	"time"

	"github.com/google/uuid"
)

type UserProfileResponse struct {
	Id           uuid.UUID `json:"id"`
	Email        string    `json:"email"`
	FullName     string    `json:"full_name"`
	Role         string    `json:"role"`
	Status       string    `json:"status"`
	AvatarURL    string    `json:"avatar_url,omitempty"` // ✅ From code lama: Avatar URL (omit if empty)
	AiDailyUsage int       `json:"ai_daily_usage"`
	CreatedAt    time.Time `json:"created_at"`
}

type UpdateProfileRequest struct {
	FullName string `json:"full_name" validate:"required,min=3"`
	Email    string `json:"email" validate:"omitempty,email"` // ✅ From code lama: Added optional email update
}

// ✅ NEW: Feature flags structure
type SubscriptionFeatures struct {
	AiChat         bool `json:"ai_chat"`
	SemanticSearch bool `json:"semantic_search"`
	MaxNotes       *int `json:"max_notes"`
}

// ✅ UPDATED: Include Features
type SubscriptionStatusResponse struct {
	PlanName           string               `json:"plan_name"`
	Status             string               `json:"status"`
	CurrentPeriodEnd   time.Time            `json:"current_period_end"`
	AiDailyCreditLimit int                  `json:"ai_daily_credit_limit"`
	IsActive           bool                 `json:"is_active"`
	Features           SubscriptionFeatures `json:"features"` // Added feature flags
}