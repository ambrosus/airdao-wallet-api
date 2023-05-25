package main

import (
	"airdao-mobile-api/config"
	"airdao-mobile-api/pkg/firebase"
	cloudmessaging "airdao-mobile-api/pkg/firebase/cloud-messaging"
	"airdao-mobile-api/pkg/logger"
	"airdao-mobile-api/pkg/mongodb"
	"airdao-mobile-api/services/health"
	"airdao-mobile-api/services/watcher"
	"context"
	"errors"
	"log"
	"os/signal"
	"syscall"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"go.uber.org/zap"
)

func main() {
	cfg, err := config.GetConfig()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	// Init logger
	newLogger, err := logger.NewLogger(cfg.Environment)
	if err != nil {
		log.Fatalf("can't create logger: %v", err)
	}

	zapLogger, err := newLogger.SetupZapLogger()
	if err != nil {
		log.Fatalf("can't setup zap logger: %v", err)
	}
	defer func(zapLogger *zap.SugaredLogger) {
		err := zapLogger.Sync()
		if err != nil && !errors.Is(err, syscall.ENOTTY) {
			log.Fatalf("can't setup zap logger: %v", err)
		}
	}(zapLogger)

	// Connect to database
	db, ctx, cancel, err := mongodb.NewConnection(cfg)
	if err != nil {
		zapLogger.Fatalf("failed to connect to mongodb: %s", err)
	}
	defer mongodb.Close(db, ctx, cancel)

	// Ping db
	err = mongodb.Ping(db, ctx)
	if err != nil {
		zapLogger.Fatal(err)
	}
	zapLogger.Info("DB connected successfully")

	// Firebase
	firebaseClient, err := firebase.NewClient(cfg.Firebase.CredPath)
	if err != nil {
		zapLogger.Fatalf("failed to create firebase messaging client - %v", err)
	}

	cloudMessagingClient, err := firebaseClient.CreateCloudMessagingClient()
	if err != nil {
		zapLogger.Fatalf("failed to create firebase cloud messaging client - %v", err)
	}

	// Firebase message
	cloudMessagingService, err := cloudmessaging.NewCloudMessagingService(cloudMessagingClient, cfg.Firebase.AndroidChannelName)
	if err != nil {
		zapLogger.Fatalf("failed to create firebase message service - %v", err)
	}

	// Repository
	watcherRepository, err := watcher.NewRepository(db, cfg.MongoDb.MongoDbName, zapLogger)
	if err != nil {
		zapLogger.Fatalf("failed to create watcher repository - %v", err)
	}

	// Services
	watcherService, err := watcher.NewService(watcherRepository, cloudMessagingService, zapLogger)
	if err != nil {
		zapLogger.Fatalf("failed to create watcher service - %v", err)
	}

	if err := watcherService.Init(context.Background()); err != nil {
		zapLogger.Fatalf("failed to init watchers - %v", err)
	}

	// Handlers
	healthHandler := health.NewHandler()

	watcherHandler, err := watcher.NewHandler(watcherService)
	if err != nil {
		zapLogger.Fatalf("failed to create watcher handler - %v", err)
	}

	// Create config variable
	config := fiber.Config{
		ServerHeader: "AIRDAO-Mobile-Api", // add custom server header
	}

	// Create fiber app
	app := fiber.New(config)

	// Init cors
	app.Use(cors.New(cors.Config{
		AllowOrigins:     "*",
		AllowHeaders:     "Origin, Content-Type, Accept, Content-Length, Accept-Encoding, Authorization",
		AllowMethods:     "GET, POST, PUT, DELETE, OPTIONS",
		AllowCredentials: true,
	}))

	// Set-up Route
	app.Route("/api/v1", func(router fiber.Router) {
		healthHandler.SetupRoutes(router)
		watcherHandler.SetupRoutes(router)
	})

	// Handle 404 page
	app.Use(func(c *fiber.Ctx) error {
		return c.Status(fiber.StatusNotFound).JSON(map[string]string{"error": "page not found"})
	})

	// Start the server
	go func() {
		if err := app.Listen(":" + cfg.Port); err != nil {
			zapLogger.Fatal(err)
		}
	}()

	zapLogger.Infoln("Server started on port %v", cfg.Port)

	// Create a context that will be used to gracefully shut down the server
	ctx, cancel = signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// Wait for the termination signal
	<-ctx.Done()

	// Perform the graceful shutdown by closing the server
	if err := app.Shutdown(); err != nil {
		zapLogger.Fatal(err)
	}

	zapLogger.Info("Server gracefully stopped")
}
