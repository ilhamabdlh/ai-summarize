package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// JobStatus represents the status of an evaluation job
type JobStatus string

const (
	StatusQueued     JobStatus = "queued"
	StatusProcessing JobStatus = "processing"
	StatusCompleted  JobStatus = "completed"
	StatusFailed     JobStatus = "failed"
)

// EvaluationJob represents a job in the evaluation queue
type EvaluationJob struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Status      JobStatus          `bson:"status" json:"status"`
	CreatedAt   time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt   time.Time          `bson:"updated_at" json:"updated_at"`
	StartedAt   *time.Time         `bson:"started_at,omitempty" json:"started_at,omitempty"`
	CompletedAt *time.Time         `bson:"completed_at,omitempty" json:"completed_at,omitempty"`

	// Input files
	CVFile         string `bson:"cv_file" json:"cv_file"`
	ProjectFile    string `bson:"project_file" json:"project_file"`
	CVContent      string `bson:"cv_content" json:"cv_content"`
	ProjectContent string `bson:"project_content" json:"project_content"`

	// Results
	Result       *EvaluationResult `bson:"result,omitempty" json:"result,omitempty"`
	ErrorMessage string            `bson:"error_message,omitempty" json:"error_message,omitempty"`
	RetryCount   int               `bson:"retry_count" json:"retry_count"`
}

// EvaluationResult represents the final evaluation result
type EvaluationResult struct {
	CVMatchRate     float64 `bson:"cv_match_rate" json:"cv_match_rate"`
	CVFeedback      string  `bson:"cv_feedback" json:"cv_feedback"`
	ProjectScore    float64 `bson:"project_score" json:"project_score"`
	ProjectFeedback string  `bson:"project_feedback" json:"project_feedback"`
	OverallSummary  string  `bson:"overall_summary" json:"overall_summary"`

	// Detailed scores
	CVScores      CVScores      `bson:"cv_scores" json:"cv_scores"`
	ProjectScores ProjectScores `bson:"project_scores" json:"project_scores"`
}

// CVScores represents detailed CV evaluation scores
type CVScores struct {
	TechnicalSkills float64 `bson:"technical_skills" json:"technical_skills"`
	ExperienceLevel float64 `bson:"experience_level" json:"experience_level"`
	Achievements    float64 `bson:"achievements" json:"achievements"`
	CulturalFit     float64 `bson:"cultural_fit" json:"cultural_fit"`
}

// ProjectScores represents detailed project evaluation scores
type ProjectScores struct {
	Correctness   float64 `bson:"correctness" json:"correctness"`
	CodeQuality   float64 `bson:"code_quality" json:"code_quality"`
	Resilience    float64 `bson:"resilience" json:"resilience"`
	Documentation float64 `bson:"documentation" json:"documentation"`
	Creativity    float64 `bson:"creativity" json:"creativity"`
}

// JobDescription represents a job description stored in vector DB
type JobDescription struct {
	ID           primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Title        string             `bson:"title" json:"title"`
	Description  string             `bson:"description" json:"description"`
	Requirements string             `bson:"requirements" json:"requirements"`
	Embedding    []float64          `bson:"embedding" json:"embedding"`
	CreatedAt    time.Time          `bson:"created_at" json:"created_at"`
}

// ScoringRubric represents the scoring rubric for project evaluation
type ScoringRubric struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Name        string             `bson:"name" json:"name"`
	Description string             `bson:"description" json:"description"`
	Criteria    []RubricCriteria   `bson:"criteria" json:"criteria"`
	CreatedAt   time.Time          `bson:"created_at" json:"created_at"`
}

// RubricCriteria represents individual criteria in the scoring rubric
type RubricCriteria struct {
	Name        string  `bson:"name" json:"name"`
	Description string  `bson:"description" json:"description"`
	Weight      float64 `bson:"weight" json:"weight"`
	MaxScore    float64 `bson:"max_score" json:"max_score"`
}

// UploadRequest represents the request for file upload
type UploadRequest struct {
	CVFile      string `json:"cv_file" binding:"required"`
	ProjectFile string `json:"project_file" binding:"required"`
}

// UploadResponse represents the response after file upload
type UploadResponse struct {
	Message     string `json:"message"`
	CVFile      string `json:"cv_file"`
	ProjectFile string `json:"project_file"`
}

// EvaluateRequest represents the request to start evaluation
type EvaluateRequest struct {
	CVFile      string `json:"cv_file" binding:"required"`
	ProjectFile string `json:"project_file" binding:"required"`
}

// EvaluateResponse represents the response after starting evaluation
type EvaluateResponse struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

// ResultResponse represents the response for getting evaluation result
type ResultResponse struct {
	ID     string            `json:"id"`
	Status string            `json:"status"`
	Result *EvaluationResult `json:"result,omitempty"`
	Error  string            `json:"error,omitempty"`
}
