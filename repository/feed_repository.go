package repository

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"redapplications.com/redreader/models"
)

type FeedRepository struct {
	collection *mongo.Collection
}

func NewFeedRepository(client *mongo.Client) *FeedRepository {
	collection := client.Database("redreader").Collection("feeds")
	return &FeedRepository{collection: collection}
}

func (r *FeedRepository) GetFeed(id string) (*models.Feed, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	objectId, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, fmt.Errorf("invalid id format: %v", err)
	}

	var feed models.Feed
	err = r.collection.FindOne(ctx, bson.M{"_id": objectId}).Decode(&feed)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("feed not found with id: %s", id)
		}
		return nil, err
	}

	return &feed, nil
}

func (r *FeedRepository) GetAllFeeds() ([]*models.Feed, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cursor, err := r.collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var feeds []*models.Feed
	if err = cursor.All(ctx, &feeds); err != nil {
		return nil, err
	}
	return feeds, nil
}

func (r *FeedRepository) UpdateLastFetched(id string, lastFetchedTime time.Time) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	result, err := r.collection.UpdateOne(
		ctx,
		bson.M{"_id": objectID},
		bson.M{"$set": bson.M{"lastFetched": lastFetchedTime}},
	)
	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return mongo.ErrNoDocuments
	}

	return nil
}

func (r *FeedRepository) GetPaginatedFeeds(page, perPage int64) ([]*models.Feed, int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Calculate total count
	total, err := r.collection.CountDocuments(ctx, bson.M{})
	if err != nil {
		return nil, 0, err
	}

	// Set up pagination options
	skip := (page - 1) * perPage
	opts := options.Find().
		SetSort(bson.D{{Key: "lastFetched", Value: -1}}).
		SetSkip(skip).
		SetLimit(perPage)

	cursor, err := r.collection.Find(ctx, bson.M{}, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var feeds []*models.Feed
	if err = cursor.All(ctx, &feeds); err != nil {
		return nil, 0, err
	}

	return feeds, total, nil
}
