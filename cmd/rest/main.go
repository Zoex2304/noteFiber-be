// FILE: cmd/rest/main.go
package main

import (
	"ai-notetaking-be/internal/controller"
	"ai-notetaking-be/internal/pkg/logger"
	"ai-notetaking-be/internal/pkg/mailer"
	"ai-notetaking-be/internal/pkg/serverutils"
	"ai-notetaking-be/internal/repository"
	"ai-notetaking-be/internal/service"
	"ai-notetaking-be/pkg/database"
	"context"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/pubsub/gochannel"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system env")
	}

	app := fiber.New(fiber.Config{
		BodyLimit: 10 * 1024 * 1024,
	})

	// ✅ CORS Configuration - Using Environment Variable
	allowedOrigins := os.Getenv("CORS_ALLOWED_ORIGINS")
	if allowedOrigins == "" {
		allowedOrigins = "http://localhost:5173" // fallback default
		log.Println("⚠️  CORS_ALLOWED_ORIGINS not set, using default:", allowedOrigins)
	}

	app.Use(cors.New(cors.Config{
		AllowOrigins:     allowedOrigins,                                    // Read from .env
		AllowCredentials: true,                                              // Required for Authorization headers
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization",     // Must include Authorization
		AllowMethods:     "GET, POST, PUT, PATCH, DELETE, OPTIONS",         // Standard HTTP methods
		ExposeHeaders:    "Content-Length, Content-Type, Authorization",     // Allow frontend to read these headers
	}))

	log.Printf("✅ CORS enabled for: %s", allowedOrigins)

	app.Use(serverutils.ErrorHandlerMiddleware())

	// --- Serve Static Files (Uploads) ---
	// This enables http://localhost:3000/uploads/avatars/filename.jpg
	app.Static("/uploads", "./uploads")

	db := database.ConnectDB(os.Getenv("DB_CONNECTION_STRING"))

	// --- Initialize CSV Logger ---
	logFilePath := os.Getenv("LOG_FILE_PATH")
	if logFilePath == "" {
		logFilePath = "app.log.csv"
	}
	sysLogger := logger.NewCsvLogger(logFilePath)

	// --- Initialize Email Service ---
	smtpPort, _ := strconv.Atoi(os.Getenv("SMTP_PORT"))
	emailService := mailer.NewEmailService(
		os.Getenv("SMTP_HOST"),
		smtpPort,
		os.Getenv("SMTP_EMAIL"),
		os.Getenv("SMTP_PASSWORD"),
		os.Getenv("SMTP_SENDER_NAME"),
	)

	// --- Repositories ---
	exampleRepository := repository.NewExampleRepository(db)
	notebookRepository := repository.NewNotebookRepository(db)
	noteRepository := repository.NewNoteRepository(db)
	noteEmbeddingRepository := repository.NewNoteEmbeddingRepository(db)
	chatSessionRepository := repository.NewChatSessionRepository(db)
	chatMessageRepository := repository.NewChatMessageRepository(db)
	chatMessageRawRepository := repository.NewChatMessageRawRepository(db)
	
	// User, Sub, & Billing Repos
	userRepository := repository.NewUserRepository(db)
	subscriptionRepository := repository.NewSubscriptionRepository(db)
	billingRepository := repository.NewBillingRepository(db)

	// --- Event Bus ---
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

	// --- Services ---
	exampleService := service.NewExampleService(exampleRepository)
	notebookService := service.NewNotebookService(
		notebookRepository,
		noteRepository,
		db,
		publisherService,
		noteEmbeddingRepository,
	)
	
	// ✅ UPDATED: Inject subscriptionRepository
	noteService := service.NewNoteService(
		noteRepository, 
		publisherService, 
		noteEmbeddingRepository, 
		db,
		subscriptionRepository, // INJECTED
	)

	// ✅ UPDATED: Inject subscriptionRepository
	chatbotService := service.NewChatbotService(
		db,
		chatSessionRepository,
		chatMessageRepository,
		chatMessageRawRepository,
		noteEmbeddingRepository,
		subscriptionRepository, // INJECTED
	)

	// Business Services
	// Inject EmailService into AuthService
	authService := service.NewAuthService(userRepository, emailService)
	
	// Payment Service now requires Billing Repo
	paymentService := service.NewPaymentService(subscriptionRepository, userRepository, billingRepository)
	
	userService := service.NewUserService(userRepository)
	oauthService := service.NewOAuthService(userRepository)
	adminService := service.NewAdminService(userRepository, subscriptionRepository, sysLogger)

	locationService := service.NewLocationService(
		os.Getenv("GEOAPIFY_API_KEY"),
		os.Getenv("BINDERBYTE_API_KEY"),
	)

	// --- Controllers ---
	exampleController := controller.NewExampleController(exampleService)
	notebookController := controller.NewNotebookController(notebookService)
	noteController := controller.NewNoteController(noteService)
	chatbotController := controller.NewChatbotController(chatbotService)
	authController := controller.NewAuthController(authService)
	paymentController := controller.NewPaymentController(paymentService)
	userController := controller.NewUserController(userService)
	adminController := controller.NewAdminController(adminService)
	oauthController := controller.NewOAuthController(oauthService)
	locationController := controller.NewLocationController(locationService)

	// --- Routes ---
	api := app.Group("/api")
	exampleController.RegisterRoutes(api)
	notebookController.RegisterRoutes(api)
	noteController.RegisterRoutes(api)
	chatbotController.RegisterRoutes(api)
	authController.RegisterRoutes(api)
	paymentController.RegisterRoutes(api)
	userController.RegisterRoutes(api)
	adminController.RegisterRoutes(api)
	oauthController.RegisterRoutes(api)
	locationController.RegisterRoutes(api)

	// --- Background Consumers ---
	err := consumerService.Consume(context.Background())
	if err != nil {
		panic(err)
	}

	fmt.Println("✅ Server is running on http://localhost:3000")
	log.Fatal(app.Listen(":3000"))
}