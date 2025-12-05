// FILE: internal/dto/admin_dto.go
package dto

import (
	"time"

	"github.com/google/uuid"
)

type AdminDashboardStats struct {
	TotalUsers        int     `json:"total_users"`
	ActiveUsers       int     `json:"active_users"`
	TotalRevenue      float64 `json:"total_revenue"` // Estimated from subscriptions
	ActiveSubscribers int     `json:"active_subscribers"`
}

type UserListResponse struct {
	Id        uuid.UUID `json:"id"`
	Email     string    `json:"email"`
	FullName  string    `json:"full_name"`
	Role      string    `json:"role"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

type PaginationRequest struct {
	Page  int `query:"page"`
	Limit int `query:"limit"`
}

type UpdateUserStatusRequest struct {
	Status string `json:"status" validate:"required,oneof=active blocked pending"`
}