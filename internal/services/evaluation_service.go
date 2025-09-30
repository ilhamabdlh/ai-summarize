package services

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"

	"ai-cv-summarize/internal/config"
	"ai-cv-summarize/internal/llm"
	"ai-cv-summarize/internal/models"
	"ai-cv-summarize/internal/rag"
	"ai-cv-summarize/internal/repositories"
)

type EvaluationService struct {
	llmClient   llm.LLMClient
	repository  *repositories.MongoDBRepository
	vectorStore *rag.VectorStore
	config      *config.Config
}

func NewEvaluationService(
	llmClient llm.LLMClient,
	repository *repositories.MongoDBRepository,
	vectorStore *rag.VectorStore,
	config *config.Config,
) *EvaluationService {
	return &EvaluationService{
		llmClient:   llmClient,
		repository:  repository,
		vectorStore: vectorStore,
		config:      config,
	}
}

// EvaluateCandidate runs the complete evaluation pipeline
func (es *EvaluationService) EvaluateCandidate(ctx context.Context, jobID string) error {
	// Get job from database
	job, err := es.repository.GetJobByID(ctx, jobID)
	if err != nil {
		return fmt.Errorf("failed to get job: %w", err)
	}

	// Update status to processing
	if err := es.repository.UpdateJobStatus(ctx, jobID, models.StatusProcessing); err != nil {
		return fmt.Errorf("failed to update job status: %w", err)
	}

	// Get relevant context from RAG
	context, err := es.vectorStore.GetRelevantContext(ctx, job.CVContent, job.ProjectContent)
	if err != nil {
		return fmt.Errorf("failed to get relevant context: %w", err)
	}

	// Step 1: Extract structured info from CV
	cvAnalysis, err := es.analyzeCV(ctx, job.CVContent, context)
	if err != nil {
		return fmt.Errorf("failed to analyze CV: %w", err)
	}

	// Step 2: Evaluate CV against job requirements
	cvEvaluation, err := es.evaluateCV(ctx, cvAnalysis, context)
	if err != nil {
		return fmt.Errorf("failed to evaluate CV: %w", err)
	}

	// Step 3: Evaluate project report
	projectEvaluation, err := es.evaluateProject(ctx, job.ProjectContent, context)
	if err != nil {
		return fmt.Errorf("failed to evaluate project: %w", err)
	}

	// Step 4: Generate overall summary
	overallSummary, err := es.generateOverallSummary(ctx, cvEvaluation, projectEvaluation)
	if err != nil {
		return fmt.Errorf("failed to generate overall summary: %w", err)
	}

	// Create final result
	result := &models.EvaluationResult{
		CVMatchRate:     cvEvaluation.MatchRate,
		CVFeedback:      cvEvaluation.Feedback,
		ProjectScore:    projectEvaluation.Score,
		ProjectFeedback: projectEvaluation.Feedback,
		OverallSummary:  overallSummary,
		CVScores:        cvEvaluation.Scores,
		ProjectScores:   projectEvaluation.Scores,
	}

	// Save result to database
	if err := es.repository.UpdateJobResult(ctx, jobID, result); err != nil {
		return fmt.Errorf("failed to update job result: %w", err)
	}

	return nil
}

// analyzeCV extracts structured information from CV
func (es *EvaluationService) analyzeCV(ctx context.Context, cvContent, context string) (*CVAnalysis, error) {
	prompt := fmt.Sprintf(`Analyze the following CV and extract structured information:

CV Content:
%s

Context:
%s

Please extract and return the following information in JSON format:
{
  "technical_skills": ["skill1", "skill2", ...],
  "experience_years": number,
  "projects": [
    {
      "name": "project_name",
      "description": "project_description",
      "technologies": ["tech1", "tech2", ...],
      "impact": "impact_description"
    }
  ],
  "achievements": ["achievement1", "achievement2", ...],
  "education": "education_background",
  "certifications": ["cert1", "cert2", ...]
}`, cvContent, context)

	response, err := es.llmClient.GenerateStructuredCompletionWithRetry(
		ctx, prompt, 0.3, es.config.JobQueue.MaxRetries,
	)
	if err != nil {
		return nil, err
	}

	var analysis CVAnalysis
	if err := json.Unmarshal([]byte(response), &analysis); err != nil {
		return nil, fmt.Errorf("failed to parse CV analysis: %w", err)
	}

	return &analysis, nil
}

// evaluateCV evaluates CV against job requirements
func (es *EvaluationService) evaluateCV(ctx context.Context, analysis *CVAnalysis, context string) (*CVEvaluation, error) {
	prompt := fmt.Sprintf(`Evaluate the following CV analysis against job requirements:

CV Analysis:
%s

Context:
%s

Evaluate based on these criteria (1-5 scale):
1. Technical Skills Match (40%% weight): backend, databases, APIs, cloud, AI/LLM exposure
2. Experience Level (25%% weight): years of experience and project complexity
3. Relevant Achievements (20%% weight): impact and scale of past work
4. Cultural/Collaboration Fit (15%% weight): communication, learning mindset, teamwork

Return JSON format:
{
  "technical_skills_score": number,
  "experience_level_score": number,
  "achievements_score": number,
  "cultural_fit_score": number,
  "match_rate": number,
  "feedback": "detailed_feedback_string"
}`, analysis.String(), context)

	response, err := es.llmClient.GenerateStructuredCompletionWithRetry(
		ctx, prompt, 0.3, es.config.JobQueue.MaxRetries,
	)
	if err != nil {
		return nil, err
	}

	var evaluation CVEvaluation
	if err := json.Unmarshal([]byte(response), &evaluation); err != nil {
		return nil, fmt.Errorf("failed to parse CV evaluation: %w", err)
	}

	// Calculate weighted match rate and round to 2 decimal places
	matchRate := (evaluation.TechnicalSkills*0.4 +
		evaluation.ExperienceLevel*0.25 +
		evaluation.Achievements*0.2 +
		evaluation.CulturalFit*0.15) / 5.0
	evaluation.MatchRate = math.Round(matchRate*100) / 100

	// Populate Scores struct
	evaluation.Scores = models.CVScores{
		TechnicalSkills: evaluation.TechnicalSkills,
		ExperienceLevel: evaluation.ExperienceLevel,
		Achievements:    evaluation.Achievements,
		CulturalFit:     evaluation.CulturalFit,
	}

	return &evaluation, nil
}

// evaluateProject evaluates project report
func (es *EvaluationService) evaluateProject(ctx context.Context, projectContent, context string) (*ProjectEvaluation, error) {
	prompt := fmt.Sprintf(`Evaluate the following project report:

Project Content:
%s

Context:
%s

Evaluate based on these criteria (1-5 scale):
1. Correctness (30%% weight): prompt design, LLM chaining, RAG, error handling
2. Code Quality (25%% weight): clean, modular, testable code
3. Resilience (20%% weight): handles failures, retries, error handling
4. Documentation (15%% weight): clear README, setup instructions, trade-offs
5. Creativity/Bonus (10%% weight): extra features beyond requirements

Return JSON format:
{
  "correctness_score": number,
  "code_quality_score": number,
  "resilience_score": number,
  "documentation_score": number,
  "creativity_score": number,
  "overall_score": number,
  "feedback": "detailed_feedback_string"
}`, projectContent, context)

	response, err := es.llmClient.GenerateStructuredCompletionWithRetry(
		ctx, prompt, 0.3, es.config.JobQueue.MaxRetries,
	)
	if err != nil {
		return nil, err
	}

	var evaluation ProjectEvaluation
	if err := json.Unmarshal([]byte(response), &evaluation); err != nil {
		return nil, fmt.Errorf("failed to parse project evaluation: %w", err)
	}

	// Calculate weighted overall score and round to 2 decimal places
	overallScore := (evaluation.Correctness*0.3 +
		evaluation.CodeQuality*0.25 +
		evaluation.Resilience*0.2 +
		evaluation.Documentation*0.15 +
		evaluation.Creativity*0.1)
	evaluation.Score = math.Round(overallScore*100) / 100

	// Populate Scores struct
	evaluation.Scores = models.ProjectScores{
		Correctness:   evaluation.Correctness,
		CodeQuality:   evaluation.CodeQuality,
		Resilience:    evaluation.Resilience,
		Documentation: evaluation.Documentation,
		Creativity:    evaluation.Creativity,
	}

	return &evaluation, nil
}

// generateOverallSummary generates overall summary
func (es *EvaluationService) generateOverallSummary(ctx context.Context, cvEval *CVEvaluation, projectEval *ProjectEvaluation) (string, error) {
	prompt := fmt.Sprintf(`Generate an overall summary based on the following evaluations:

CV Evaluation:
- Match Rate: %.2f
- Technical Skills: %.2f/5
- Experience Level: %.2f/5
- Achievements: %.2f/5
- Cultural Fit: %.2f/5
- Feedback: %s

Project Evaluation:
- Overall Score: %.2f/5
- Correctness: %.2f/5
- Code Quality: %.2f/5
- Resilience: %.2f/5
- Documentation: %.2f/5
- Creativity: %.2f/5
- Feedback: %s

Generate a 3-5 sentence summary that includes:
1. Overall assessment of the candidate
2. Key strengths
3. Areas for improvement
4. Recommendation`,
		cvEval.MatchRate, cvEval.TechnicalSkills, cvEval.ExperienceLevel,
		cvEval.Achievements, cvEval.CulturalFit, cvEval.Feedback,
		projectEval.Score, projectEval.Correctness, projectEval.CodeQuality,
		projectEval.Resilience, projectEval.Documentation, projectEval.Creativity, projectEval.Feedback)

	summary, err := es.llmClient.GenerateCompletionWithRetry(
		ctx, prompt, 0.3, es.config.JobQueue.MaxRetries,
	)
	if err != nil {
		return "", err
	}

	return summary, nil
}

// Helper structs for evaluation
type CVAnalysis struct {
	TechnicalSkills []string  `json:"technical_skills"`
	ExperienceYears int       `json:"experience_years"`
	Projects        []Project `json:"projects"`
	Achievements    []string  `json:"achievements"`
	Education       string    `json:"education"`
	Certifications  []string  `json:"certifications"`
}

type Project struct {
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	Technologies []string `json:"technologies"`
	Impact       string   `json:"impact"`
}

type CVEvaluation struct {
	TechnicalSkills float64 `json:"technical_skills_score"`
	ExperienceLevel float64 `json:"experience_level_score"`
	Achievements    float64 `json:"achievements_score"`
	CulturalFit     float64 `json:"cultural_fit_score"`
	MatchRate       float64 `json:"match_rate"`
	Feedback        string  `json:"feedback"`
	Scores          models.CVScores
}

type ProjectEvaluation struct {
	Correctness   float64 `json:"correctness_score"`
	CodeQuality   float64 `json:"code_quality_score"`
	Resilience    float64 `json:"resilience_score"`
	Documentation float64 `json:"documentation_score"`
	Creativity    float64 `json:"creativity_score"`
	Score         float64 `json:"overall_score"`
	Feedback      string  `json:"feedback"`
	Scores        models.ProjectScores
}

func (cv *CVAnalysis) String() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Technical Skills: %s\n", strings.Join(cv.TechnicalSkills, ", ")))
	sb.WriteString(fmt.Sprintf("Experience Years: %d\n", cv.ExperienceYears))
	sb.WriteString(fmt.Sprintf("Projects: %d\n", len(cv.Projects)))
	sb.WriteString(fmt.Sprintf("Achievements: %s\n", strings.Join(cv.Achievements, ", ")))
	sb.WriteString(fmt.Sprintf("Education: %s\n", cv.Education))
	sb.WriteString(fmt.Sprintf("Certifications: %s\n", strings.Join(cv.Certifications, ", ")))
	return sb.String()
}
