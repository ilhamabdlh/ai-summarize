package services

import (
	"context"
	"log"
	"time"

	"ai-cv-summarize/internal/models"
	"ai-cv-summarize/internal/repositories"
)

type DatabaseInitService struct {
	repository *repositories.MongoDBRepository
}

func NewDatabaseInitService(repository *repositories.MongoDBRepository) *DatabaseInitService {
	return &DatabaseInitService{
		repository: repository,
	}
}

// InitializeDatabase initializes the database with default data
func (dis *DatabaseInitService) InitializeDatabase(ctx context.Context) error {
	log.Println("Initializing database...")

	// Initialize default job description
	if err := dis.initializeDefaultJobDescription(ctx); err != nil {
		return err
	}

	// Initialize default scoring rubric
	if err := dis.initializeDefaultScoringRubric(ctx); err != nil {
		return err
	}

	log.Println("Database initialization completed")
	return nil
}

// initializeDefaultJobDescription creates a default job description
func (dis *DatabaseInitService) initializeDefaultJobDescription(ctx context.Context) error {
	// Check if job descriptions already exist
	existing, err := dis.repository.GetAllJobDescriptions(ctx)
	if err != nil {
		return err
	}

	if len(existing) > 0 {
		log.Println("Job descriptions already exist, skipping initialization")
		return nil
	}

	// Create default job description
	jobDesc := &models.JobDescription{
		Title: "Product Engineer (Backend) - Rakamin",
		Description: `Rakamin is hiring a Product Engineer (Backend) to work on Rakamin. We're looking for dedicated engineers who write code they're proud of and who are eager to keep scaling and improving complex systems, including those powered by AI.

You'll be building new product features alongside a frontend engineer and product manager using our Agile methodology, as well as addressing issues to ensure our apps are robust and our codebase is clean. As a Product Engineer, you'll write clean, efficient code to enhance our product's codebase in meaningful ways.

In addition to classic backend work, this role also touches on building AI-powered systems, where you'll design and orchestrate how large language models (LLMs) integrate into Rakamin's product ecosystem.`,
		Requirements: `We're looking for candidates with a strong track record of working on backend technologies of web apps, ideally with exposure to AI/LLM development or a strong desire to learn.

You should have experience with backend languages and frameworks (Node.js, Django, Rails), as well as modern backend tooling and technologies such as:

- Database management (MySQL, PostgreSQL, MongoDB)
- RESTful APIs
- Security compliance
- Cloud technologies (AWS, Google Cloud, Azure)
- Server-side languages (Java, Python, Ruby, or JavaScript)
- Understanding of frontend technologies
- User authentication and authorization between multiple systems, servers, and environments
- Scalable application design principles
- Creating database schemas that represent and support business processes
- Implementing automated testing platforms and unit tests
- Familiarity with LLM APIs, embeddings, vector databases and prompt design best practices`,
		CreatedAt: time.Now(),
	}

	// Save to database
	if err := dis.repository.CreateJobDescription(ctx, jobDesc); err != nil {
		return err
	}

	log.Println("Default job description created")
	return nil
}

// initializeDefaultScoringRubric creates a default scoring rubric
func (dis *DatabaseInitService) initializeDefaultScoringRubric(ctx context.Context) error {
	// Check if scoring rubrics already exist
	existing, err := dis.repository.GetDefaultScoringRubric(ctx)
	if err == nil && existing != nil {
		log.Println("Scoring rubric already exists, skipping initialization")
		return nil
	}

	// Create default scoring rubric
	rubric := &models.ScoringRubric{
		Name:        "default",
		Description: "Default scoring rubric for candidate evaluation",
		Criteria: []models.RubricCriteria{
			{
				Name:        "Technical Skills Match",
				Description: "Alignment with job requirements (backend, databases, APIs, cloud, AI/LLM)",
				Weight:      0.4,
				MaxScore:    5.0,
			},
			{
				Name:        "Experience Level",
				Description: "Years of experience and project complexity",
				Weight:      0.25,
				MaxScore:    5.0,
			},
			{
				Name:        "Relevant Achievements",
				Description: "Impact of past work (scaling, performance, adoption)",
				Weight:      0.2,
				MaxScore:    5.0,
			},
			{
				Name:        "Cultural/Collaboration Fit",
				Description: "Communication, learning mindset, teamwork/leadership",
				Weight:      0.15,
				MaxScore:    5.0,
			},
		},
		CreatedAt: time.Now(),
	}

	// Save to database
	if err := dis.repository.CreateScoringRubric(ctx, rubric); err != nil {
		return err
	}

	log.Println("Default scoring rubric created")
	return nil
}

// CreateSampleJobDescriptions creates sample job descriptions for testing
func (dis *DatabaseInitService) CreateSampleJobDescriptions(ctx context.Context) error {
	sampleJobs := []*models.JobDescription{
		{
			Title:        "Senior Backend Developer - Tech Company",
			Description:  "We are looking for a senior backend developer to join our team. You will be responsible for designing and implementing scalable backend systems.",
			Requirements: "5+ years of experience with Node.js, Python, or Java. Experience with microservices, Docker, and cloud platforms.",
			CreatedAt:    time.Now(),
		},
		{
			Title:        "AI Engineer - Startup",
			Description:  "Join our AI team to build cutting-edge AI solutions. You will work on machine learning models and AI-powered features.",
			Requirements: "Experience with Python, TensorFlow/PyTorch, and cloud AI services. Knowledge of NLP and computer vision.",
			CreatedAt:    time.Now(),
		},
		{
			Title:        "Full Stack Developer - E-commerce",
			Description:  "We need a full stack developer to work on our e-commerce platform. You will handle both frontend and backend development.",
			Requirements: "Experience with React, Node.js, and databases. Knowledge of payment systems and e-commerce best practices.",
			CreatedAt:    time.Now(),
		},
	}

	for _, job := range sampleJobs {
		if err := dis.repository.CreateJobDescription(ctx, job); err != nil {
			return err
		}
	}

	log.Println("Sample job descriptions created")
	return nil
}
