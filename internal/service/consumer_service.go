package service

import (
	"ai-notetaking-be/internal/dto"
	"ai-notetaking-be/internal/entity"
	"ai-notetaking-be/internal/repository"
	"ai-notetaking-be/pkg/embedding"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/pubsub/gochannel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type IConsumerService interface {
	Consume(ctx context.Context) error
}

type consumerService struct {
	notebookRepository      repository.INotebookRepository
	noteRepository          repository.INoteRepository
	noteEmbeddingRepository repository.INoteEmbeddingRepository
	pubSub                  *gochannel.GoChannel
	topicName               string

	db *pgxpool.Pool
}

func (cs *consumerService) Consume(ctx context.Context) error {
	messages, err := cs.pubSub.Subscribe(ctx, cs.topicName)
	if err != nil {
		return err
	}

	go func() {
		for msg := range messages {
			cs.processMessage(ctx, msg)
		}
	}()

	return nil
}

func (cs *consumerService) processMessage(ctx context.Context, msg *message.Message) {
	// CRITICAL FIX: Proper error handling without defer Nack
	var payload dto.PublishEmbedNoteMessage
	err := json.Unmarshal(msg.Payload, &payload)
	if err != nil {
		log.Printf("[ERROR] Failed to unmarshal message: %v", err)
		msg.Ack() // Ack invalid messages to prevent infinite retry
		return
	}

	log.Printf("[INFO] Processing note embedding for NoteId: %s", payload.NoteId)

	note, err := cs.noteRepository.GetById(ctx, payload.NoteId)
	if err != nil {
		log.Printf("[ERROR] Failed to get note %s: %v", payload.NoteId, err)
		msg.Nack() // Nack for retriable errors
		return
	}

	notebook, err := cs.notebookRepository.GetById(ctx, note.NotebookId)
	if err != nil {
		log.Printf("[ERROR] Failed to get notebook %s: %v", note.NotebookId, err)
		msg.Nack()
		return
	}

	noteUpdatedAt := "-"
	if note.UpdatedAt != nil {
		noteUpdatedAt = note.UpdatedAt.Format(time.RFC3339)
	}
	content := fmt.Sprintf(`Note Title: %s
Notebook Title: %s

%s

Created At: %s
Updated At: %s`,
		note.Title,
		notebook.Name,
		note.Content,
		note.CreatedAt.Format(time.RFC3339),
		noteUpdatedAt,
	)

	log.Printf("[INFO] Generating embedding for note %s (content length: %d)", payload.NoteId, len(content))
	res, err := embedding.GetGeminiEmbedding(
		os.Getenv("GOOGLE_GEMINI_API_KEY"),
		content,
		"RETRIEVAL_DOCUMENT",
	)
	if err != nil {
		log.Printf("[ERROR] Failed to generate embedding for note %s: %v", payload.NoteId, err)
		msg.Nack()
		return
	}

	log.Printf("[INFO] Embedding generated successfully (dimensions: %d)", len(res.Embedding.Values))

	noteEmbedding := entity.NoteEmbedding{
		Id:             uuid.New(),
		Document:       content,
		EmbeddingValue: res.Embedding.Values,
		NoteId:         note.Id,
		CreatedAt:      time.Now(),
	}

	tx, err := cs.db.Begin(ctx)
	if err != nil {
		log.Printf("[ERROR] Failed to begin transaction: %v", err)
		msg.Nack()
		return
	}
	defer tx.Rollback(ctx)

	noteEmbeddingRepository := cs.noteEmbeddingRepository.UsingTx(ctx, tx)
	
	log.Printf("[INFO] Deleting old embeddings for note %s", payload.NoteId)
	err = noteEmbeddingRepository.DeleteByNoteId(ctx, note.Id)
	if err != nil {
		log.Printf("[ERROR] Failed to delete old embeddings: %v", err)
		msg.Nack()
		return
	}

	log.Printf("[INFO] Creating new embedding for note %s", payload.NoteId)
	err = noteEmbeddingRepository.Create(ctx, &noteEmbedding)
	if err != nil {
		log.Printf("[ERROR] Failed to create embedding: %v", err)
		msg.Nack()
		return
	}

	err = tx.Commit(ctx)
	if err != nil {
		log.Printf("[ERROR] Failed to commit transaction: %v", err)
		msg.Nack()
		return
	}

	log.Printf("[SUCCESS] Note embedding processed successfully for NoteId: %s", payload.NoteId)
	msg.Ack() // CRITICAL FIX: Only Ack on success
}

func NewConsumerService(
	pubSub *gochannel.GoChannel,
	topicName string,
	noteRepository repository.INoteRepository,
	noteEmbeddingRepository repository.INoteEmbeddingRepository,
	notebookRepository repository.INotebookRepository,
	db *pgxpool.Pool,
) IConsumerService {
	return &consumerService{
		pubSub:                  pubSub,
		topicName:               topicName,
		noteRepository:          noteRepository,
		noteEmbeddingRepository: noteEmbeddingRepository,
		notebookRepository:      notebookRepository,
		db:                      db,
	}
}