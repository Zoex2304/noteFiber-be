package service

import (
	"ai-notetaking-be/internal/constant"
	"ai-notetaking-be/internal/dto"
	"ai-notetaking-be/internal/entity"
	"ai-notetaking-be/internal/repository"
	"ai-notetaking-be/pkg/chatbot"
	"ai-notetaking-be/pkg/embedding"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type IChatbotService interface {
	CreateSession(ctx context.Context) (*dto.CreateSessionResponse, error)
	GetAllSessions(ctx context.Context) ([]*dto.GetAllSessionsResponse, error)
	GetChatHistory(ctx context.Context, sessionId uuid.UUID) ([]*dto.GetChatHistoryResponse, error)
	SendChat(ctx context.Context, request *dto.SendChatRequest) (*dto.SendChatResponse, error)
	DeleteSession(ctx context.Context, request *dto.DeleteSessionRequest) error
}

type chatbotService struct {
	db                       *pgxpool.Pool
	chatSessionRepository    repository.IChatSessionRepository
	chatMessageRepository    repository.IChatMessageRepository
	chatMessageRawRepository repository.IChatMessageRawRepository
	noteEmbeddingRepository  repository.INoteEmbeddingRepository
}

func (cs *chatbotService) CreateSession(ctx context.Context) (*dto.CreateSessionResponse, error) {

	now := time.Now()
	chatSession := entity.ChatSession{
		Id:        uuid.New(),
		Title:     "Unnamed session",
		CreatedAt: now,
	}
	chatMessage := entity.ChatMessage{
		Id:            uuid.New(),
		Chat:          "Hi, how can I help you ?",
		Role:          constant.ChatMessageRoleModel,
		ChatSessionId: chatSession.Id,
		CreatedAt:     now,
	}
	chatMessageRawUser := entity.ChatMessageRaw{
		Id:            uuid.New(),
		Chat:          constant.ChatMessageRawInitialUserPromptV1,
		Role:          constant.ChatMessageRoleUser,
		ChatSessionId: chatSession.Id,
		CreatedAt:     now,
	}
	chatMessageRawModel := entity.ChatMessageRaw{
		Id:            uuid.New(),
		Chat:          constant.ChatMessageRawInitialModelPromptV1,
		Role:          constant.ChatMessageRoleModel,
		ChatSessionId: chatSession.Id,
		CreatedAt:     now.Add(1 * time.Second),
	}

	tx, err := cs.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	chatSessionRepository := cs.chatSessionRepository.UsingTx(ctx, tx)
	chatMessageRepository := cs.chatMessageRepository.UsingTx(ctx, tx)
	chatMessageRawRepository := cs.chatMessageRawRepository.UsingTx(ctx, tx)

	err = chatSessionRepository.Create(ctx, &chatSession)
	if err != nil {
		return nil, err
	}
	err = chatMessageRepository.Create(ctx, &chatMessage)
	if err != nil {
		return nil, err
	}
	err = chatMessageRawRepository.Create(ctx, &chatMessageRawUser)
	if err != nil {
		return nil, err
	}
	err = chatMessageRawRepository.Create(ctx, &chatMessageRawModel)
	if err != nil {
		return nil, err
	}

	err = tx.Commit(ctx)
	if err != nil {
		return nil, err
	}

	return &dto.CreateSessionResponse{
		Id: chatSession.Id,
	}, nil
}

func (cs *chatbotService) GetAllSessions(ctx context.Context) ([]*dto.GetAllSessionsResponse, error) {
	chatSessions, err := cs.chatSessionRepository.GetAll(ctx)
	if err != nil {
		return nil, err
	}

	response := make([]*dto.GetAllSessionsResponse, 0)
	for _, chatSession := range chatSessions {
		response = append(response, &dto.GetAllSessionsResponse{
			Id:        chatSession.Id,
			Title:     chatSession.Title,
			CreatedAt: chatSession.CreatedAt,
			UpdatedAt: chatSession.UpdatedAt,
		})
	}

	return response, nil
}

func (cs *chatbotService) GetChatHistory(ctx context.Context, sessionId uuid.UUID) ([]*dto.GetChatHistoryResponse, error) {
	_, err := cs.chatSessionRepository.GetById(ctx, sessionId)
	if err != nil {
		return nil, err
	}

	chatMessages, err := cs.chatMessageRepository.GetByChatSessionId(ctx, sessionId)
	if err != nil {
		return nil, err
	}

	response := make([]*dto.GetChatHistoryResponse, 0)
	for _, chatMessage := range chatMessages {
		response = append(response, &dto.GetChatHistoryResponse{
			Id:        chatMessage.Id,
			Role:      chatMessage.Role,
			Chat:      chatMessage.Chat,
			CreatedAt: chatMessage.CreatedAt,
		})
	}

	return response, nil
}

func (cs *chatbotService) SendChat(ctx context.Context, request *dto.SendChatRequest) (*dto.SendChatResponse, error) {
	tx, err := cs.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	chatSessionRepository := cs.chatSessionRepository.UsingTx(ctx, tx)
	chatMessageRepository := cs.chatMessageRepository.UsingTx(ctx, tx)
	chatMessageRawRepository := cs.chatMessageRawRepository.UsingTx(ctx, tx)
	noteEmbeddingRepository := cs.noteEmbeddingRepository.UsingTx(ctx, tx)

	chatSession, err := chatSessionRepository.GetById(ctx, request.ChatSessionId)
	if err != nil {
		return nil, err
	}

	existingRawChats, err := chatMessageRawRepository.GetByChatSessionId(ctx, request.ChatSessionId)
	if err != nil {
		return nil, err
	}
	updateSessionTitle := len(existingRawChats) == 2

	now := time.Now()

	chatMessage := entity.ChatMessage{
		Id:            uuid.New(),
		Chat:          request.Chat,
		Role:          constant.ChatMessageRoleUser,
		ChatSessionId: request.ChatSessionId,
		CreatedAt:     now,
	}

	embeddingRes, err := embedding.GetGeminiEmbedding(
		os.Getenv("GOOGLE_GEMINI_API_KEY"),
		request.Chat,
		"RETRIEVAL_QUERY",
	)
	if err != nil {
		return nil, err
	}

	decideUseRAGChatHistories := make([]*chatbot.ChatHistory, 0)
	for i, rawChat := range existingRawChats {
		if i == 0 {
			decideUseRAGChatHistories = append(decideUseRAGChatHistories, &chatbot.ChatHistory{
				Chat: constant.DecideUseRAGMessageRawInitialUserPromptV1,
				Role: constant.ChatMessageRoleUser,
			})
			continue
		} else if i == 1 {
			decideUseRAGChatHistories = append(decideUseRAGChatHistories, &chatbot.ChatHistory{
				Chat: constant.DecideUseRAGMessageRawInitialModelPromptV1,
				Role: constant.ChatMessageRoleModel,
			})
			continue
		}

		decideUseRAGChatHistories = append(decideUseRAGChatHistories, &chatbot.ChatHistory{
			Chat: rawChat.Chat,
			Role: rawChat.Role,
		})
	}

	useRag, err := chatbot.DecideToUseRAG(
		ctx,
		os.Getenv("GOOGLE_GEMINI_API_KEY"),
		decideUseRAGChatHistories,
	)
	if err != nil {
		return nil, err
	}

	strBuilder := strings.Builder{}
	if useRag {
		noteEmbeddings, err := noteEmbeddingRepository.SearchSimilarity(
			ctx,
			embeddingRes.Embedding.Values,
		)
		if err != nil {
			return nil, err
		}

		for i, noteEmbedding := range noteEmbeddings {
			strBuilder.WriteString(fmt.Sprintf("Reference %d\n", i+1))
			strBuilder.WriteString(noteEmbedding.Document)
			strBuilder.WriteString("\n\n")
		}
	}

	strBuilder.WriteString("User next question: ")
	strBuilder.WriteString(request.Chat)
	strBuilder.WriteString("\n\n")
	strBuilder.WriteString("Your answer ?")
	chatMessageRaw := entity.ChatMessageRaw{
		Id:            uuid.New(),
		Chat:          strBuilder.String(),
		Role:          constant.ChatMessageRoleUser,
		ChatSessionId: request.ChatSessionId,
		CreatedAt:     now,
	}

	existingRawChats = append(
		existingRawChats,
		&chatMessageRaw,
	)

	geminiReq := make([]*chatbot.ChatHistory, 0)
	for _, existingRawChat := range existingRawChats {
		geminiReq = append(geminiReq, &chatbot.ChatHistory{
			Chat: existingRawChat.Chat,
			Role: existingRawChat.Role,
		})
	}

	reply, err := chatbot.GetGeminiResponse(
		ctx,
		os.Getenv("GOOGLE_GEMINI_API_KEY"),
		geminiReq,
	)
	if err != nil {
		return nil, err
	}

	chatMessageModel := entity.ChatMessage{
		Id:            uuid.New(),
		Chat:          reply,
		Role:          constant.ChatMessageRoleModel,
		ChatSessionId: request.ChatSessionId,
		CreatedAt:     now.Add(1 * time.Millisecond),
	}
	chatMessageModelRaw := entity.ChatMessageRaw{
		Id:            uuid.New(),
		Chat:          reply,
		Role:          constant.ChatMessageRoleModel,
		ChatSessionId: request.ChatSessionId,
		CreatedAt:     now.Add(1 * time.Millisecond),
	}

	chatMessageRepository.Create(ctx, &chatMessage)
	chatMessageRepository.Create(ctx, &chatMessageModel)
	chatMessageRawRepository.Create(ctx, &chatMessageRaw)
	chatMessageRawRepository.Create(ctx, &chatMessageModelRaw)

	if updateSessionTitle {
		chatSession.Title = request.Chat
		chatSession.UpdatedAt = &now
		err = chatSessionRepository.Update(ctx, chatSession)
		if err != nil {
			return nil, err
		}
	}

	err = tx.Commit(ctx)
	if err != nil {
		return nil, err
	}

	return &dto.SendChatResponse{
		ChatSessionId:    chatSession.Id,
		ChatSessionTitle: chatSession.Title,
		Sent: &dto.SendChatResponseChat{
			Id:        chatMessage.Id,
			Chat:      chatMessage.Chat,
			Role:      chatMessage.Role,
			CreatedAt: chatMessage.CreatedAt,
		},
		Reply: &dto.SendChatResponseChat{
			Id:        chatMessageModel.Id,
			Chat:      chatMessageModel.Chat,
			Role:      chatMessageModel.Role,
			CreatedAt: chatMessageModel.CreatedAt,
		},
	}, nil
}

func (cs *chatbotService) DeleteSession(ctx context.Context, request *dto.DeleteSessionRequest) error {

	tx, err := cs.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	chatSessionRepository := cs.chatSessionRepository.UsingTx(ctx, tx)
	chatMessageRepository := cs.chatMessageRepository.UsingTx(ctx, tx)
	chatMessageRawRepository := cs.chatMessageRawRepository.UsingTx(ctx, tx)

	_, err = chatSessionRepository.GetById(ctx, request.ChatSessionId)
	if err != nil {
		return err
	}

	err = cs.chatSessionRepository.Delete(ctx, request.ChatSessionId)
	if err != nil {
		return err
	}

	err = chatMessageRepository.DeleteByChatSessionId(ctx, request.ChatSessionId)
	if err != nil {
		return err
	}

	err = chatMessageRawRepository.DeleteByChatSessionId(ctx, request.ChatSessionId)
	if err != nil {
		return err
	}

	err = tx.Commit(ctx)
	if err != nil {
		return err
	}

	return nil
}

func NewChatbotService(
	db *pgxpool.Pool,
	chatSessionRepository repository.IChatSessionRepository,
	chatMessageRepository repository.IChatMessageRepository,
	chatMessageRawRepository repository.IChatMessageRawRepository,
	noteEmbeddingRepository repository.INoteEmbeddingRepository,
) IChatbotService {
	return &chatbotService{
		db:                       db,
		chatSessionRepository:    chatSessionRepository,
		chatMessageRepository:    chatMessageRepository,
		chatMessageRawRepository: chatMessageRawRepository,
		noteEmbeddingRepository:  noteEmbeddingRepository,
	}
}
