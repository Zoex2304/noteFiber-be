package dto

import (
	"time"

	"github.com/google/uuid"
)

type CreateSessionResponse struct {
	Id uuid.UUID `json:"id"`
}

type GetAllSessionsResponse struct {
	Id        uuid.UUID  `json:"id"`
	Title     string     `json:"title"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt *time.Time `json:"updated_at"`
}

type GetChatHistoryResponse struct {
	Id        uuid.UUID `json:"id"`
	Role      string    `json:"role"`
	Chat      string    `json:"chat"`
	CreatedAt time.Time `json:"created_at"`
}

type SendChatRequest struct {
	ChatSessionId uuid.UUID `json:"chat_session_id" validate:"required"`
	Chat          string    `json:"chat" validate:"required"`
}

type SendChatResponseChat struct {
	Id        uuid.UUID `json:"id"`
	Chat      string    `json:"chat"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
}

type SendChatResponse struct {
	ChatSessionId    uuid.UUID             `json:"chat_session_id"`
	ChatSessionTitle string                `json:"title"`
	Sent             *SendChatResponseChat `json:"sent"`
	Reply            *SendChatResponseChat `json:"reply"`
}

type DeleteSessionRequest struct {
	ChatSessionId uuid.UUID `json:"chat_session_id"`
}
