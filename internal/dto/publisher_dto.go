package dto

import "github.com/google/uuid"

type PublishEmbedNoteMessage struct {
	NoteId uuid.UUID `json:"note_id"`
}
