package services

import (
	"fmt"
	"math"

	"ai-cv-summarize/internal/models"
	"ai-cv-summarize/internal/repositories"
)

type ScoringService struct {
	repository *repositories.MongoDBRepository
}

func NewScoringService(repository *repositories.MongoDBRepository) *ScoringService {
	return &ScoringService{
		repository: repository,
	}
}

// CalculateCVScore calculates the overall CV score based on weighted criteria
func (ss *ScoringService) CalculateCVScore(scores models.CVScores) float64 {
	// Weighted average calculation
	weightedSum := (scores.TechnicalSkills * 0.4) +
		(scores.ExperienceLevel * 0.25) +
		(scores.Achievements * 0.2) +
		(scores.CulturalFit * 0.15)

	return math.Round(weightedSum*100) / 100 // Round to 2 decimal places
}

// CalculateProjectScore calculates the overall project score based on weighted criteria
func (ss *ScoringService) CalculateProjectScore(scores models.ProjectScores) float64 {
	// Weighted average calculation
	weightedSum := (scores.Correctness * 0.3) +
		(scores.CodeQuality * 0.25) +
		(scores.Resilience * 0.2) +
		(scores.Documentation * 0.15) +
		(scores.Creativity * 0.1)

	return math.Round(weightedSum*100) / 100 // Round to 2 decimal places
}

// NormalizeScore normalizes a score to a 0-1 range
func (ss *ScoringService) NormalizeScore(score, maxScore float64) float64 {
	if maxScore == 0 {
		return 0
	}
	return math.Min(score/maxScore, 1.0)
}

// CalculateOverallScore calculates the overall candidate score
func (ss *ScoringService) CalculateOverallScore(cvScore, projectScore float64) float64 {
	// 60% CV score, 40% project score
	overallScore := (cvScore * 0.6) + (projectScore * 0.4)
	return math.Round(overallScore*100) / 100
}

// GetScoreInterpretation returns a human-readable interpretation of the score
func (ss *ScoringService) GetScoreInterpretation(score float64) string {
	switch {
	case score >= 4.5:
		return "Excellent - Highly recommended"
	case score >= 4.0:
		return "Very Good - Strong candidate"
	case score >= 3.5:
		return "Good - Solid candidate"
	case score >= 3.0:
		return "Average - Consider with reservations"
	case score >= 2.5:
		return "Below Average - Not recommended"
	default:
		return "Poor - Not suitable"
	}
}

// ValidateScore validates if a score is within acceptable range
func (ss *ScoringService) ValidateScore(score float64) error {
	if score < 0 || score > 5 {
		return fmt.Errorf("score must be between 0 and 5, got %f", score)
	}
	return nil
}

// GetScoreBreakdown returns a detailed breakdown of scores
func (ss *ScoringService) GetScoreBreakdown(scores models.CVScores, projectScores models.ProjectScores) map[string]interface{} {
	return map[string]interface{}{
		"cv_scores": map[string]interface{}{
			"technical_skills": scores.TechnicalSkills,
			"experience_level": scores.ExperienceLevel,
			"achievements":     scores.Achievements,
			"cultural_fit":     scores.CulturalFit,
			"overall":          ss.CalculateCVScore(scores),
		},
		"project_scores": map[string]interface{}{
			"correctness":   projectScores.Correctness,
			"code_quality":  projectScores.CodeQuality,
			"resilience":    projectScores.Resilience,
			"documentation": projectScores.Documentation,
			"creativity":    projectScores.Creativity,
			"overall":       ss.CalculateProjectScore(projectScores),
		},
		"overall_score": ss.CalculateOverallScore(
			ss.CalculateCVScore(scores),
			ss.CalculateProjectScore(projectScores),
		),
	}
}

// GenerateScoreReport generates a comprehensive score report
func (ss *ScoringService) GenerateScoreReport(result *models.EvaluationResult) map[string]interface{} {
	overallScore := ss.CalculateOverallScore(
		ss.CalculateCVScore(result.CVScores),
		ss.CalculateProjectScore(result.ProjectScores),
	)

	return map[string]interface{}{
		"summary": map[string]interface{}{
			"overall_score":          overallScore,
			"overall_interpretation": ss.GetScoreInterpretation(overallScore),
			"cv_match_rate":          result.CVMatchRate,
			"project_score":          result.ProjectScore,
		},
		"cv_evaluation": map[string]interface{}{
			"match_rate": result.CVMatchRate,
			"feedback":   result.CVFeedback,
			"scores":     result.CVScores,
		},
		"project_evaluation": map[string]interface{}{
			"score":    result.ProjectScore,
			"feedback": result.ProjectFeedback,
			"scores":   result.ProjectScores,
		},
		"overall_summary": result.OverallSummary,
		"breakdown":       ss.GetScoreBreakdown(result.CVScores, result.ProjectScores),
	}
}

// CompareScores compares two sets of scores and returns the difference
func (ss *ScoringService) CompareScores(score1, score2 float64) map[string]interface{} {
	diff := score1 - score2
	percentageDiff := (diff / score2) * 100

	return map[string]interface{}{
		"score1":          score1,
		"score2":          score2,
		"difference":      diff,
		"percentage_diff": math.Round(percentageDiff*100) / 100,
		"higher":          score1 > score2,
	}
}

// GetScoreStatistics returns statistics for a set of scores
func (ss *ScoringService) GetScoreStatistics(scores []float64) map[string]interface{} {
	if len(scores) == 0 {
		return map[string]interface{}{
			"count": 0,
			"mean":  0,
			"min":   0,
			"max":   0,
		}
	}

	var sum float64
	min := scores[0]
	max := scores[0]

	for _, score := range scores {
		sum += score
		if score < min {
			min = score
		}
		if score > max {
			max = score
		}
	}

	mean := sum / float64(len(scores))

	return map[string]interface{}{
		"count": len(scores),
		"mean":  math.Round(mean*100) / 100,
		"min":   min,
		"max":   max,
	}
}
