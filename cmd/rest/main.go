package main

import (
	"ai-notetaking-be/internal/controller"
	"ai-notetaking-be/internal/pkg/serverutils"
	"ai-notetaking-be/internal/repository"
	"ai-notetaking-be/internal/service"
	"ai-notetaking-be/pkg/database"
	"context"
	"fmt"
	"log"
	"os"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/pubsub/gochannel"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load()
	app := fiber.New(fiber.Config{
		BodyLimit: 10 * 1024 * 1024,
	})

	app.Use(cors.New())

	app.Use(serverutils.ErrorHandlerMiddleware())

	db := database.ConnectDB(os.Getenv("DB_CONNECTION_STRING"))

	exampleRepository := repository.NewExampleRepository(db)
	notebookRepository := repository.NewNotebookRepository(db)
	noteRepository := repository.NewNoteRepository(db)
	noteEmbeddingRepository := repository.NewNoteEmbeddingRepository(db)
	chatSessionRepository := repository.NewChatSessionRepository(db)
	chatMessageRepository := repository.NewChatMessageRepository(db)
	chatMessageRawRepository := repository.NewChatMessageRawRepository(db)

	watermillLogger := watermill.NewStdLogger(false, false)
	pubSub := gochannel.NewGoChannel(
		gochannel.Config{},
		watermillLogger,
	)
	publisherService := service.NewPublisherService(
		os.Getenv("EMBED_NOTE_CONTENT_TOPIC_NAME"),
		pubSub,
	)
	consumerService := service.NewConsumerService(
		pubSub,
		os.Getenv("EMBED_NOTE_CONTENT_TOPIC_NAME"),
		noteRepository,
		noteEmbeddingRepository,
		notebookRepository,
		db,
	)

	exampleService := service.NewExampleService(exampleRepository)
	notebookService := service.NewNotebookService(
		notebookRepository,
		noteRepository,
		db,
		publisherService,
		noteEmbeddingRepository,
	)
	noteService := service.NewNoteService(noteRepository, publisherService, noteEmbeddingRepository, db)
	chatbotService := service.NewChatbotService(
		db,
		chatSessionRepository,
		chatMessageRepository,
		chatMessageRawRepository,
		noteEmbeddingRepository,
	)

	exampleController := controller.NewExampleController(exampleService)
	notebookController := controller.NewNotebookController(notebookService)
	noteController := controller.NewNoteController(noteService)
	chatbotController := controller.NewChatbotController(chatbotService)

	api := app.Group("/api")
	exampleController.RegisterRoutes(api)
	notebookController.RegisterRoutes(api)
	noteController.RegisterRoutes(api)
	chatbotController.RegisterRoutes(api)

	err := consumerService.Consume(context.Background())
	if err != nil {
		panic(err)
	}

	fmt.Println("Server is running")
	log.Fatal(app.Listen(":3000"))
}
