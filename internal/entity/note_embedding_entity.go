package entity

import (
	"time"

	"github.com/google/uuid"
)

type NoteEmbedding struct {
	Id             uuid.UUID
	Document       string
	EmbeddingValue []float32
	NoteId         uuid.UUID
	CreatedAt      time.Time
	UpdatedAt      *time.Time
	DeletedAt      *time.Time
	IsDeleted      bool
}
