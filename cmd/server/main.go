package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"ai-cv-summarize/internal/config"
	"ai-cv-summarize/internal/handlers"
	"ai-cv-summarize/internal/llm"
	"ai-cv-summarize/internal/rag"
	"ai-cv-summarize/internal/repositories"
	"ai-cv-summarize/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatal("Failed to load configuration:", err)
	}

	// Set Gin mode
	gin.SetMode(cfg.Server.GinMode)

	// Connect to MongoDB
	mongoClient, err := mongo.Connect(context.TODO(), options.Client().ApplyURI(cfg.MongoDB.URI))
	if err != nil {
		log.Fatal("Failed to connect to MongoDB:", err)
	}
	defer mongoClient.Disconnect(context.TODO())

	// Get database
	db := mongoClient.Database(cfg.MongoDB.Database)

	// Connect to Redis
	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379", // Default Redis address
	})
	defer redisClient.Close()

	// Test Redis connection
	if err := redisClient.Ping(context.TODO()).Err(); err != nil {
		log.Fatal("Failed to connect to Redis:", err)
	}

	// Initialize repositories
	repository := repositories.NewMongoDBRepository(db)

	// Initialize database with default data
	dbInitService := services.NewDatabaseInitService(repository)
	if err := dbInitService.InitializeDatabase(context.TODO()); err != nil {
		log.Printf("Warning: Failed to initialize database: %v", err)
	}

	// Initialize LLM client
	llmFactory := llm.NewLLMFactory()
	llmClient := llmFactory.CreateClient(&cfg.OpenAI, &cfg.OpenRouter)

	// Initialize services
	fileService := services.NewFileService(cfg.Upload.UploadDir, cfg.Upload.MaxFileSize)
	vectorStore := rag.NewVectorStore(llmClient, repository, &cfg.VectorDB)
	evaluationService := services.NewEvaluationService(llmClient, repository, vectorStore, cfg)
	jobQueue := services.NewJobQueue(redisClient, repository, evaluationService, cfg)

	// Initialize handlers
	uploadHandler := handlers.NewUploadHandler(fileService)
	evaluationHandler := handlers.NewEvaluationHandler(repository, evaluationService, jobQueue, fileService)

	// Setup routes
	router := setupRoutes(uploadHandler, evaluationHandler)

	// Start job queue processor in background
	go jobQueue.ProcessJobs()

	// Start server
	server := &http.Server{
		Addr:    ":" + cfg.Server.Port,
		Handler: router,
	}

	// Start server in background
	go func() {
		log.Printf("Server starting on port %s", cfg.Server.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("Failed to start server:", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	log.Println("Server exited")
}

func setupRoutes(uploadHandler *handlers.UploadHandler, evaluationHandler *handlers.EvaluationHandler) *gin.Engine {
	router := gin.Default()

	// CORS middleware
	router.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// API routes
	api := router.Group("/api/v1")
	{
		// Upload routes
		api.POST("/upload", uploadHandler.UploadFiles)
		api.POST("/upload-with-content", uploadHandler.UploadFilesWithContent)

		// Evaluation routes
		api.POST("/evaluate", evaluationHandler.StartEvaluation)
		api.GET("/result/:id", evaluationHandler.GetResult)
		api.GET("/job/:id", evaluationHandler.GetJobStatus)
		api.GET("/jobs", evaluationHandler.ListJobs)
	}

	return router
}
