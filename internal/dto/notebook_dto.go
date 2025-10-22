package dto

import (
	"time"

	"github.com/google/uuid"
)

type CreateNotebookRequest struct {
	Name     string     `json:"name" validate:"required"`
	ParentId *uuid.UUID `json:"parent_id"`
}

type CreateNotebookResponse struct {
	Id uuid.UUID `json:"id"`
}

type ShowNotebookResponse struct {
	Id        uuid.UUID  `json:"id"`
	Name      string     `json:"name"`
	ParentId  *uuid.UUID `json:"parent_id"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt *time.Time `json:"updated_at"`
}

type UpdateNotebookRequest struct {
	Id   uuid.UUID
	Name string `json:"name" validate:"required"`
}

type UpdateNotebookResponse struct {
	Id uuid.UUID `json:"id"`
}

type MoveNotebookRequest struct {
	Id       uuid.UUID
	ParentId *uuid.UUID `json:"parent_id"`
}

type MoveNotebookResponse struct {
	Id uuid.UUID `json:"id"`
}

type GetAllNotebookResponseNote struct {
	Id        uuid.UUID  `json:"id"`
	Title     string     `json:"title"`
	Content   string     `json:"content"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt *time.Time `json:"updated_at"`
}

type GetAllNotebookResponse struct {
	Id        uuid.UUID  `json:"id"`
	Name      string     `json:"name"`
	ParentId  *uuid.UUID `json:"parent_id"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt *time.Time `json:"updated_at"`

	Notes []*GetAllNotebookResponseNote `json:"notes"`
}
