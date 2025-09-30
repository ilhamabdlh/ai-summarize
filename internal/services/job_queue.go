package services

import (
	"context"
	"fmt"
	"log"
	"time"

	"ai-cv-summarize/internal/config"
	"ai-cv-summarize/internal/models"
	"ai-cv-summarize/internal/repositories"

	"github.com/redis/go-redis/v9"
)

type JobQueue struct {
	redisClient       *redis.Client
	repository        *repositories.MongoDBRepository
	evaluationService *EvaluationService
	config            *config.Config
}

func NewJobQueue(redisClient *redis.Client, repository *repositories.MongoDBRepository, evaluationService *EvaluationService, config *config.Config) *JobQueue {
	return &JobQueue{
		redisClient:       redisClient,
		repository:        repository,
		evaluationService: evaluationService,
		config:            config,
	}
}

// AddJob adds a job to the queue
func (jq *JobQueue) AddJob(jobID string) error {
	ctx := context.Background()

	// Add job to Redis queue
	return jq.redisClient.LPush(ctx, "evaluation_queue", jobID).Err()
}

// ProcessJobs processes jobs from the queue
func (jq *JobQueue) ProcessJobs() {
	ctx := context.Background()

	for {
		// Block and wait for job
		result, err := jq.redisClient.BRPop(ctx, 0, "evaluation_queue").Result()
		if err != nil {
			log.Printf("Error waiting for job: %v", err)
			time.Sleep(5 * time.Second)
			continue
		}

		if len(result) < 2 {
			continue
		}

		jobID := result[1]
		log.Printf("Processing job: %s", jobID)

		// Process the job
		if err := jq.processJob(ctx, jobID); err != nil {
			log.Printf("Error processing job %s: %v", jobID, err)

			// Increment retry count
			if err := jq.repository.IncrementRetryCount(ctx, jobID); err != nil {
				log.Printf("Error incrementing retry count for job %s: %v", jobID, err)
			}
		}
	}
}

// processJob processes a single job
func (jq *JobQueue) processJob(ctx context.Context, jobID string) error {
	// Get job from database
	job, err := jq.repository.GetJobByID(ctx, jobID)
	if err != nil {
		return fmt.Errorf("failed to get job: %w", err)
	}

	// Check if job is already completed or failed
	if job.Status == models.StatusCompleted || job.Status == models.StatusFailed {
		return nil
	}

	// Check retry count
	if job.RetryCount >= jq.config.JobQueue.MaxRetries {
		return jq.repository.UpdateJobError(ctx, jobID, "Max retries exceeded")
	}

	// Update status to processing
	if err := jq.repository.UpdateJobStatus(ctx, jobID, models.StatusProcessing); err != nil {
		return fmt.Errorf("failed to update job status: %w", err)
	}

	// Run real AI evaluation using evaluation service
	if err := jq.evaluationService.EvaluateCandidate(ctx, jobID); err != nil {
		// Update job with error
		if updateErr := jq.repository.UpdateJobError(ctx, jobID, err.Error()); updateErr != nil {
			log.Printf("Error updating job error: %v", updateErr)
		}
		return fmt.Errorf("evaluation failed: %w", err)
	}

	log.Printf("Job %s completed successfully", jobID)
	return nil
}

// GetQueueStatus returns the current queue status
func (jq *JobQueue) GetQueueStatus() (map[string]interface{}, error) {
	ctx := context.Background()

	// Get queue length
	queueLength, err := jq.redisClient.LLen(ctx, "evaluation_queue").Result()
	if err != nil {
		return nil, err
	}

	// Get pending jobs from database
	pendingJobs, err := jq.repository.GetPendingJobs(ctx)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"queue_length": queueLength,
		"pending_jobs": len(pendingJobs),
		"status":       "running",
	}, nil
}

// ClearQueue clears all jobs from the queue
func (jq *JobQueue) ClearQueue() error {
	ctx := context.Background()
	return jq.redisClient.Del(ctx, "evaluation_queue").Err()
}

// GetJobFromQueue retrieves a job from the queue without removing it
func (jq *JobQueue) GetJobFromQueue() (string, error) {
	ctx := context.Background()

	result, err := jq.redisClient.LIndex(ctx, "evaluation_queue", -1).Result()
	if err != nil {
		return "", err
	}

	return result, nil
}

// RemoveJobFromQueue removes a job from the queue
func (jq *JobQueue) RemoveJobFromQueue(jobID string) error {
	ctx := context.Background()
	return jq.redisClient.LRem(ctx, "evaluation_queue", 0, jobID).Err()
}
