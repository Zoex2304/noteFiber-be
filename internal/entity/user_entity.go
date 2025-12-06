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
	AvatarURL             string     // ✅ NEW: Avatar URL from OAuth provider
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
	AvatarURL      string // Added field for architecture synchronization
	CreatedAt      time.Time
}

type EmailVerificationToken struct {
	Id        uuid.UUID
	UserId    uuid.UUID
	Token     string
	ExpiresAt time.Time
	CreatedAt time.Time
}

// NEW: Refresh Token Entity matching the new table
type UserRefreshToken struct {
	Id        uuid.UUID
	UserId    uuid.UUID
	TokenHash string
	ExpiresAt time.Time
	Revoked   bool
	CreatedAt time.Time
	IpAddress string // Optional
	UserAgent string // Optional
}