package rag

import (
	"context"
	"fmt"
	"math"
	"strings"

	"ai-cv-summarize/internal/config"
	"ai-cv-summarize/internal/llm"
	"ai-cv-summarize/internal/models"
	"ai-cv-summarize/internal/repositories"
)

type VectorStore struct {
	llmClient  llm.LLMClient
	repository *repositories.MongoDBRepository
	config     *config.VectorDBConfig
}

func NewVectorStore(llmClient llm.LLMClient, repository *repositories.MongoDBRepository, config *config.VectorDBConfig) *VectorStore {
	return &VectorStore{
		llmClient:  llmClient,
		repository: repository,
		config:     config,
	}
}

func (vs *VectorStore) AddJobDescription(ctx context.Context, title, description, requirements string) error {
	fullText := fmt.Sprintf("Title: %s\nDescription: %s\nRequirements: %s", title, description, requirements)

	embedding, err := vs.llmClient.GenerateEmbedding(ctx, fullText)
	if err != nil {
		return fmt.Errorf("failed to generate embedding: %w", err)
	}

	jobDesc := &models.JobDescription{
		Title:        title,
		Description:  description,
		Requirements: requirements,
		Embedding:    embedding,
	}

	return vs.repository.CreateJobDescription(ctx, jobDesc)
}

func (vs *VectorStore) SearchSimilarJobDescriptions(ctx context.Context, query string, limit int) ([]*models.JobDescription, error) {
	if query == "" {
		return nil, fmt.Errorf("query cannot be empty")
	}

	query = strings.TrimSpace(query)
	if query == "" {
		return nil, fmt.Errorf("query is empty after trimming")
	}

	queryEmbedding, err := vs.llmClient.GenerateEmbedding(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to generate query embedding: %w", err)
	}

	jobDescs, err := vs.repository.GetAllJobDescriptions(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get job descriptions: %w", err)
	}

	type scoredJob struct {
		job   *models.JobDescription
		score float64
	}

	var scoredJobs []scoredJob
	for _, job := range jobDescs {
		similarity := vs.cosineSimilarity(queryEmbedding, job.Embedding)
		scoredJobs = append(scoredJobs, scoredJob{
			job:   job,
			score: similarity,
		})
	}

	// Sort by similarity
	for i := 0; i < len(scoredJobs); i++ {
		for j := i + 1; j < len(scoredJobs); j++ {
			if scoredJobs[i].score < scoredJobs[j].score {
				scoredJobs[i], scoredJobs[j] = scoredJobs[j], scoredJobs[i]
			}
		}
	}

	if limit > len(scoredJobs) {
		limit = len(scoredJobs)
	}

	var results []*models.JobDescription
	for i := 0; i < limit; i++ {
		results = append(results, scoredJobs[i].job)
	}

	return results, nil
}

func (vs *VectorStore) GetRelevantContext(ctx context.Context, cvContent, projectContent string) (string, error) {
	cvResults, err := vs.SearchSimilarJobDescriptions(ctx, cvContent, 2)
	if err != nil {
		return "", fmt.Errorf("failed to search CV context: %w", err)
	}

	projectResults, err := vs.SearchSimilarJobDescriptions(ctx, projectContent, 2)
	if err != nil {
		return "", fmt.Errorf("failed to search project context: %w", err)
	}

	contextMap := make(map[string]*models.JobDescription)

	for _, result := range cvResults {
		contextMap[result.ID.Hex()] = result
	}

	for _, result := range projectResults {
		contextMap[result.ID.Hex()] = result
	}

	var context strings.Builder
	context.WriteString("Relevant Job Descriptions:\n\n")

	for _, job := range contextMap {
		context.WriteString(fmt.Sprintf("Title: %s\n", job.Title))
		context.WriteString(fmt.Sprintf("Description: %s\n", job.Description))
		context.WriteString(fmt.Sprintf("Requirements: %s\n\n", job.Requirements))
	}

	return context.String(), nil
}

func (vs *VectorStore) cosineSimilarity(a, b []float64) float64 {
	if len(a) != len(b) {
		return 0.0
	}

	var dotProduct, normA, normB float64
	for i := 0; i < len(a); i++ {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 0.0
	}

	return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
}
