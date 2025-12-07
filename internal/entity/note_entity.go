package entity

import (
	"time"

	"github.com/google/uuid"
)

type Note struct {
	Id         uuid.UUID
	Title      string
	Content    string
	NotebookId uuid.UUID
	UserId     uuid.UUID
	CreatedAt  time.Time
	UpdatedAt  *time.Time
	DeletedAt  *time.Time
	IsDeleted  bool
}
