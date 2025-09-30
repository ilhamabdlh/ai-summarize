package repositories

import (
	"context"
	"fmt"
	"time"

	"ai-cv-summarize/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoDBRepository struct {
	db *mongo.Database
}

func NewMongoDBRepository(db *mongo.Database) *MongoDBRepository {
	return &MongoDBRepository{db: db}
}

// Job Repository Methods
func (r *MongoDBRepository) CreateJob(ctx context.Context, job *models.EvaluationJob) (interface{}, error) {
	collection := r.db.Collection("evaluation_jobs")
	id, err := collection.InsertOne(ctx, job)
	fmt.Println("Job created: ", id.InsertedID)
	return id.InsertedID, err
}

func (r *MongoDBRepository) GetJobByID(ctx context.Context, id string) (*models.EvaluationJob, error) {
	collection := r.db.Collection("evaluation_jobs")
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}

	var job models.EvaluationJob
	err = collection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&job)
	if err != nil {
		return nil, err
	}

	return &job, nil
}

func (r *MongoDBRepository) UpdateJobStatus(ctx context.Context, id string, status models.JobStatus) error {
	collection := r.db.Collection("evaluation_jobs")
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	update := bson.M{
		"$set": bson.M{
			"status":     status,
			"updated_at": time.Now(),
		},
	}

	if status == models.StatusProcessing {
		now := time.Now()
		update["$set"].(bson.M)["started_at"] = now
	} else if status == models.StatusCompleted || status == models.StatusFailed {
		now := time.Now()
		update["$set"].(bson.M)["completed_at"] = now
	}

	_, err = collection.UpdateOne(ctx, bson.M{"_id": objectID}, update)
	return err
}

func (r *MongoDBRepository) UpdateJobResult(ctx context.Context, id string, result *models.EvaluationResult) error {
	collection := r.db.Collection("evaluation_jobs")
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	update := bson.M{
		"$set": bson.M{
			"result":       result,
			"status":       models.StatusCompleted,
			"updated_at":   time.Now(),
			"completed_at": time.Now(),
		},
	}

	_, err = collection.UpdateOne(ctx, bson.M{"_id": objectID}, update)
	return err
}

func (r *MongoDBRepository) UpdateJobError(ctx context.Context, id string, errorMessage string) error {
	collection := r.db.Collection("evaluation_jobs")
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	update := bson.M{
		"$set": bson.M{
			"error_message": errorMessage,
			"status":        models.StatusFailed,
			"updated_at":    time.Now(),
			"completed_at":  time.Now(),
		},
	}

	_, err = collection.UpdateOne(ctx, bson.M{"_id": objectID}, update)
	return err
}

func (r *MongoDBRepository) IncrementRetryCount(ctx context.Context, id string) error {
	collection := r.db.Collection("evaluation_jobs")
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	update := bson.M{
		"$inc": bson.M{"retry_count": 1},
		"$set": bson.M{"updated_at": time.Now()},
	}

	_, err = collection.UpdateOne(ctx, bson.M{"_id": objectID}, update)
	return err
}

func (r *MongoDBRepository) GetPendingJobs(ctx context.Context) ([]*models.EvaluationJob, error) {
	collection := r.db.Collection("evaluation_jobs")

	cursor, err := collection.Find(ctx, bson.M{
		"status": bson.M{"$in": []models.JobStatus{models.StatusQueued, models.StatusProcessing}},
	})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var jobs []*models.EvaluationJob
	if err = cursor.All(ctx, &jobs); err != nil {
		return nil, err
	}

	return jobs, nil
}

func (r *MongoDBRepository) GetJobsWithFilters(ctx context.Context, status string, limit, offset int) ([]*models.EvaluationJob, error) {
	collection := r.db.Collection("evaluation_jobs")

	filter := bson.M{}
	if status != "" {
		filter["status"] = status
	}

	opts := options.Find().
		SetLimit(int64(limit)).
		SetSkip(int64(offset)).
		SetSort(bson.D{{"created_at", -1}})

	cursor, err := collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var jobs []*models.EvaluationJob
	if err = cursor.All(ctx, &jobs); err != nil {
		return nil, err
	}

	return jobs, nil
}

// Job Description Repository Methods
func (r *MongoDBRepository) CreateJobDescription(ctx context.Context, jobDesc *models.JobDescription) error {
	collection := r.db.Collection("job_descriptions")
	_, err := collection.InsertOne(ctx, jobDesc)
	return err
}

func (r *MongoDBRepository) GetJobDescription(ctx context.Context, id string) (*models.JobDescription, error) {
	collection := r.db.Collection("job_descriptions")
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}

	var jobDesc models.JobDescription
	err = collection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&jobDesc)
	if err != nil {
		return nil, err
	}

	return &jobDesc, nil
}

func (r *MongoDBRepository) GetAllJobDescriptions(ctx context.Context) ([]*models.JobDescription, error) {
	collection := r.db.Collection("job_descriptions")

	cursor, err := collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var jobDescs []*models.JobDescription
	if err = cursor.All(ctx, &jobDescs); err != nil {
		return nil, err
	}

	return jobDescs, nil
}

// Scoring Rubric Repository Methods
func (r *MongoDBRepository) CreateScoringRubric(ctx context.Context, rubric *models.ScoringRubric) error {
	collection := r.db.Collection("scoring_rubrics")
	_, err := collection.InsertOne(ctx, rubric)
	return err
}

func (r *MongoDBRepository) GetScoringRubric(ctx context.Context, id string) (*models.ScoringRubric, error) {
	collection := r.db.Collection("scoring_rubrics")
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}

	var rubric models.ScoringRubric
	err = collection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&rubric)
	if err != nil {
		return nil, err
	}

	return &rubric, nil
}

func (r *MongoDBRepository) GetDefaultScoringRubric(ctx context.Context) (*models.ScoringRubric, error) {
	collection := r.db.Collection("scoring_rubrics")

	var rubric models.ScoringRubric
	err := collection.FindOne(ctx, bson.M{"name": "default"}).Decode(&rubric)
	if err != nil {
		return nil, err
	}

	return &rubric, nil
}
