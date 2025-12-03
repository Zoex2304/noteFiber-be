// FILE: internal/entity/user_entity.go
package entity

import (
	"time"

	"github.com/google/uuid"
)

type UserRole string
type UserStatus string

const (
	UserRoleUser  UserRole = "user"
	UserRoleAdmin UserRole = "admin"

	UserStatusPending UserStatus = "pending"
	UserStatusActive  UserStatus = "active"
	UserStatusBlocked UserStatus = "blocked"
)

type User struct {
	Id                    uuid.UUID
	Email                 string
	PasswordHash          *string
	FullName              string
	Role                  UserRole
	Status                UserStatus
	EmailVerified         bool
	EmailVerifiedAt       *time.Time
	CreatedAt             time.Time
	UpdatedAt             time.Time
	AiDailyUsage          int
	AiDailyUsageLastReset time.Time
}

type PasswordResetToken struct {
	Id        uuid.UUID
	UserId    uuid.UUID
	Token     string
	ExpiresAt time.Time
	Used      bool
	CreatedAt time.Time
}

type UserProvider struct {
	Id             uuid.UUID
	UserId         uuid.UUID
	ProviderName   string
	ProviderUserId string
	CreatedAt      time.Time
}

// Add this struct
type EmailVerificationToken struct {
	Id        uuid.UUID
	UserId    uuid.UUID
	Token     string
	ExpiresAt time.Time
	CreatedAt time.Time
}