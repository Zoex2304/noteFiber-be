package entity

import (
	"time"

	"github.com/google/uuid"
)

type Notebook struct {
	Id        uuid.UUID
	Name      string
	ParentId  *uuid.UUID
	CreatedAt time.Time
	UpdatedAt *time.Time
	DeletedAt *time.Time
	IsDeleted bool
}
