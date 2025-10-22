package dto

import (
	"time"

	"github.com/google/uuid"
)

type CreateNoteRequest struct {
	Title      string    `json:"title" validate:"required"`
	Content    string    `json:"content"`
	NotebookId uuid.UUID `json:"notebook_id" validate:"required"`
}

type CreateNoteResponse struct {
	Id uuid.UUID `json:"id"`
}

type ShowNoteResponse struct {
	Id         uuid.UUID  `json:"id"`
	Title      string     `json:"title"`
	Content    string     `json:"content"`
	NotebookId uuid.UUID  `json:"notebook_id"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  *time.Time `json:"updated_at"`
}

type UpdateNoteRequest struct {
	Id      uuid.UUID
	Title   string `json:"title" validate:"required"`
	Content string `json:"content"`
}

type UpdateNoteResponse struct {
	Id uuid.UUID `json:"id"`
}

type MoveNoteRequest struct {
	Id         uuid.UUID
	NotebookId uuid.UUID `json:"notebook_id" validate:"required"`
}

type MoveNoteResponse struct {
	Id uuid.UUID `json:"id"`
}

type SemanticSearchResponse struct {
	Id         uuid.UUID  `json:"id"`
	Title      string     `json:"title"`
	Content    string     `json:"content"`
	NotebookId uuid.UUID  `json:"notebook_id"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  *time.Time `json:"updated_at"`
}
