// internal\entity\notebook_entity.go
package entity

import (
	"time"

	"github.com/google/uuid"
)

type Notebook struct {
	Id        uuid.UUID
	Name      string
	ParentId  *uuid.UUID
	UserId    uuid.UUID // ✅ ADDED: Owner ID
	CreatedAt time.Time
	UpdatedAt *time.Time
	DeletedAt *time.Time
	IsDeleted bool
}
