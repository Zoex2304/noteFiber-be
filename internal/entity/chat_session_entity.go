package entity

import (
	"time"

	"github.com/google/uuid"
)

type ChatSession struct {
	Id        uuid.UUID
	Title     string
	CreatedAt time.Time
	UpdatedAt *time.Time
	DeletedAt *time.Time
	IsDeleted bool
}
