package handlers

import (
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"ai-cv-summarize/internal/models"
	"ai-cv-summarize/internal/repositories"
	"ai-cv-summarize/internal/services"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type EvaluationHandler struct {
	repository        *repositories.MongoDBRepository
	evaluationService *services.EvaluationService
	jobQueue          *services.JobQueue
	fileService       *services.FileService
}

func NewEvaluationHandler(
	repository *repositories.MongoDBRepository,
	evaluationService *services.EvaluationService,
	jobQueue *services.JobQueue,
	fileService *services.FileService,
) *EvaluationHandler {
	return &EvaluationHandler{
		repository:        repository,
		evaluationService: evaluationService,
		jobQueue:          jobQueue,
		fileService:       fileService,
	}
}

// StartEvaluation starts the evaluation process
func (h *EvaluationHandler) StartEvaluation(c *gin.Context) {
	var (
		req   models.EvaluateRequest
		jobID interface{}
		err   error
	)
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	// Read content from files
	cvContent, err := h.readFileContent(req.CVFile)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read CV file: " + err.Error()})
		return
	}

	projectContent, err := h.readFileContent(req.ProjectFile)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read project file: " + err.Error()})
		return
	}

	// Create new evaluation job
	job := &models.EvaluationJob{
		Status:         models.StatusQueued,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
		CVFile:         req.CVFile,
		ProjectFile:    req.ProjectFile,
		CVContent:      cvContent,
		ProjectContent: projectContent,
		RetryCount:     0,
	}

	// Save job to database
	if jobID, err = h.repository.CreateJob(c.Request.Context(), job); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create evaluation job"})
		return
	}
	job.ID = jobID.(primitive.ObjectID)
	fmt.Println("Job created: ", job.ID.Hex())

	// Add job to queue
	if err := h.jobQueue.AddJob(job.ID.Hex()); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add job to queue"})
		return
	}

	// Return response
	response := models.EvaluateResponse{
		ID:     job.ID.Hex(),
		Status: string(job.Status),
	}

	c.JSON(http.StatusOK, response)
}

// readFileContent reads content from a file
func (h *EvaluationHandler) readFileContent(filename string) (string, error) {
	// Construct file path (assuming files are in uploads directory)
	filePath := filepath.Join("uploads", filename)

	// Extract text content from file
	content, err := h.fileService.ExtractTextFromFile(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to extract text from file %s: %w", filename, err)
	}

	// Validate content is not empty
	if strings.TrimSpace(content) == "" {
		return "", fmt.Errorf("file %s is empty or contains no readable text", filename)
	}

	return content, nil
}

// GetResult retrieves the evaluation result
func (h *EvaluationHandler) GetResult(c *gin.Context) {
	jobID := c.Param("id")
	if jobID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Job ID is required"})
		return
	}

	// Get job from database
	job, err := h.repository.GetJobByID(c.Request.Context(), jobID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Job not found"})
		return
	}

	// Prepare response
	response := models.ResultResponse{
		ID:     job.ID.Hex(),
		Status: string(job.Status),
		Result: job.Result,
		Error:  job.ErrorMessage,
	}

	// Return appropriate status code based on job status
	switch job.Status {
	case models.StatusQueued, models.StatusProcessing:
		c.JSON(http.StatusOK, response)
	case models.StatusCompleted:
		c.JSON(http.StatusOK, response)
	case models.StatusFailed:
		c.JSON(http.StatusInternalServerError, response)
	default:
		c.JSON(http.StatusOK, response)
	}
}

// GetJobStatus retrieves the current status of a job
func (h *EvaluationHandler) GetJobStatus(c *gin.Context) {
	jobID := c.Param("id")
	if jobID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Job ID is required"})
		return
	}

	// Get job from database
	job, err := h.repository.GetJobByID(c.Request.Context(), jobID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Job not found"})
		return
	}

	// Return status
	response := gin.H{
		"id":         job.ID.Hex(),
		"status":     string(job.Status),
		"created_at": job.CreatedAt,
		"updated_at": job.UpdatedAt,
	}

	if job.StartedAt != nil {
		response["started_at"] = job.StartedAt
	}

	if job.CompletedAt != nil {
		response["completed_at"] = job.CompletedAt
	}

	if job.ErrorMessage != "" {
		response["error"] = job.ErrorMessage
	}

	c.JSON(http.StatusOK, response)
}

// ListJobs retrieves all jobs (for admin purposes)
func (h *EvaluationHandler) ListJobs(c *gin.Context) {
	// Get query parameters
	status := c.Query("status")
	limit := c.DefaultQuery("limit", "10")
	offset := c.DefaultQuery("offset", "0")

	// Parse limit and offset
	limitInt := 10
	offsetInt := 0

	if limit != "" {
		if parsed, err := strconv.Atoi(limit); err == nil {
			limitInt = parsed
		}
	}

	if offset != "" {
		if parsed, err := strconv.Atoi(offset); err == nil {
			offsetInt = parsed
		}
	}

	// Get jobs from database
	jobs, err := h.repository.GetJobsWithFilters(c.Request.Context(), status, limitInt, offsetInt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve jobs"})
		return
	}

	// Prepare response
	var response []gin.H
	for _, job := range jobs {
		jobResponse := gin.H{
			"id":         job.ID.Hex(),
			"status":     string(job.Status),
			"created_at": job.CreatedAt,
			"updated_at": job.UpdatedAt,
		}

		if job.StartedAt != nil {
			jobResponse["started_at"] = job.StartedAt
		}

		if job.CompletedAt != nil {
			jobResponse["completed_at"] = job.CompletedAt
		}

		if job.Result != nil {
			jobResponse["result"] = job.Result
		}

		if job.ErrorMessage != "" {
			jobResponse["error"] = job.ErrorMessage
		}

		response = append(response, jobResponse)
	}

	c.JSON(http.StatusOK, gin.H{
		"jobs":   response,
		"total":  len(response),
		"limit":  limitInt,
		"offset": offsetInt,
	})
}
