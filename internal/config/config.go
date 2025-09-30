package config

import (
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Server     ServerConfig
	MongoDB    MongoDBConfig
	Redis      RedisConfig
	OpenAI     OpenAIConfig
	OpenRouter OpenRouterConfig
	VectorDB   VectorDBConfig
	Upload     UploadConfig
	JobQueue   JobQueueConfig
}

type ServerConfig struct {
	Port    string
	GinMode string
}

type MongoDBConfig struct {
	URI      string
	Database string
}

type RedisConfig struct {
	URL string
}

type OpenAIConfig struct {
	APIKey  string
	BaseURL string
	Model   string
}

type OpenRouterConfig struct {
	APIKey  string
	BaseURL string
	Model   string
}

type VectorDBConfig struct {
	URL        string
	Collection string
}

type UploadConfig struct {
	MaxFileSize int64
	UploadDir   string
}

type JobQueueConfig struct {
	Timeout    time.Duration
	MaxRetries int
}

func Load() (*Config, error) {
	// Load .env file if exists
	godotenv.Load()

	timeout, _ := strconv.Atoi(getEnv("JOB_TIMEOUT", "300"))
	maxRetries, _ := strconv.Atoi(getEnv("MAX_RETRIES", "3"))
	maxFileSize, _ := strconv.ParseInt(getEnv("MAX_FILE_SIZE", "10485760"), 10, 64)

	return &Config{
		Server: ServerConfig{
			Port:    getEnv("PORT", "8080"),
			GinMode: getEnv("GIN_MODE", "debug"),
		},
		MongoDB: MongoDBConfig{
			URI:      getEnv("MONGODB_URI", "mongodb://localhost:27017"),
			Database: getEnv("MONGODB_DATABASE", "ai_cv_summarize"),
		},
		Redis: RedisConfig{
			URL: getEnv("REDIS_URL", "redis://localhost:6379"),
		},
		OpenAI: OpenAIConfig{
			APIKey:  getEnv("OPENAI_API_KEY", ""),
			BaseURL: getEnv("OPENAI_BASE_URL", "https://api.openai.com/v1"),
			Model:   getEnv("OPENAI_MODEL", "gpt-4"),
		},
		OpenRouter: OpenRouterConfig{
			APIKey:  getEnv("OPENROUTER_API_KEY", ""),
			BaseURL: getEnv("OPENROUTER_BASE_URL", "https://openrouter.ai/api/v1"),
			Model:   getEnv("OPENROUTER_MODEL", "openai/gpt-4"),
		},
		VectorDB: VectorDBConfig{
			URL:        getEnv("VECTOR_DB_URL", "http://localhost:8000"),
			Collection: getEnv("VECTOR_DB_COLLECTION", "job_descriptions"),
		},
		Upload: UploadConfig{
			MaxFileSize: maxFileSize,
			UploadDir:   getEnv("UPLOAD_DIR", "./uploads"),
		},
		JobQueue: JobQueueConfig{
			Timeout:    time.Duration(timeout) * time.Second,
			MaxRetries: maxRetries,
		},
	}, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
