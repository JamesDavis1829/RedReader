package repository

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"redapplications.com/redreader/models"
)

type FeedRepository struct {
	collection *mongo.Collection
}

func NewFeedRepository(client *mongo.Client) *FeedRepository {
	collection := client.Database("redreader").Collection("feeds")
	return &FeedRepository{collection: collection}
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

	_, err := r.collection.UpdateOne(
		ctx,
		bson.M{"id": id},
		bson.M{"$set": bson.M{"lastFetched": lastFetchedTime}},
	)
	return err
}
