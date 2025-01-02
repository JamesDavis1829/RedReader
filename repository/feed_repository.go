package repository

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"github.com/mmcdole/gofeed"
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

func (r *FeedRepository) GetPaginatedFeeds(user *models.User, page, perPage int64) ([]*models.Feed, int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var filter bson.M
	if user != nil {
		filter = bson.M{"$or": []bson.M{
			{"_id": bson.M{"$in": user.PersonalFeeds}},
			{"isDefault": true},
		}}
	} else {
		filter = bson.M{"isDefault": true}
	}

	// Calculate total count
	total, err := r.collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	// Set up pagination options
	skip := (page - 1) * perPage
	opts := options.Find().
		SetSort(bson.D{{Key: "title", Value: 1}}).
		SetSkip(skip).
		SetLimit(perPage)

	cursor, err := r.collection.Find(ctx, filter, opts)
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

func (r *FeedRepository) GetFeedsByIds(ids []string) ([]*models.Feed, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	objectIds := make([]primitive.ObjectID, 0)
	for _, id := range ids {
		objID, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			continue
		}
		objectIds = append(objectIds, objID)
	}

	cursor, err := r.collection.Find(ctx, bson.M{"_id": bson.M{"$in": objectIds}})
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

func (r *FeedRepository) GetFeedByTitle(ctx context.Context, name string) (*models.Feed, error) {
	filter := bson.M{"title": bson.M{"$regex": primitive.Regex{
		Pattern: "^" + regexp.QuoteMeta(name) + "$",
		Options: "i",
	}}}

	var feed models.Feed
	err := r.collection.FindOne(ctx, filter).Decode(&feed)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, fmt.Errorf("error finding feed by name: %v", err)
	}

	return &feed, nil
}

func (r *FeedRepository) AddSubscriptionStatus(feeds []*models.Feed, subscribedIds []string) {
	subscribedMap := make(map[string]bool)
	for _, id := range subscribedIds {
		subscribedMap[id] = true
	}

	for _, feed := range feeds {
		feed.IsSubscribed = subscribedMap[feed.ID.Hex()]
	}
}

func (r *FeedRepository) AddFeed(url string) (*models.Feed, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if url == "" {
		return nil, fmt.Errorf("invalid feed URL")
	}

	content, err := gofeed.NewParser().ParseURL(url)
	if err != nil {
		return nil, fmt.Errorf("error parsing feed: %w", err)
	}

	feed := models.NewFeed(url)
	feed.Title = content.Title
	feed.Description = content.Description
	feed.IsDefault = false
	feed.URL = url

	_, err = r.collection.InsertOne(ctx, feed)
	if err != nil {
		return nil, err
	}

	return feed, nil
}

func (r *FeedRepository) UserFeedExistsByURL(user *models.User, url string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	filter := bson.M{
		"$or": []bson.M{
			{"url": url, "_id": bson.M{"$in": user.PersonalFeeds}},
			{"url": url, "isDefault": true},
		},
	}

	count, err := r.collection.CountDocuments(ctx, filter)
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

func (r *FeedRepository) DeleteFeedByID(feedID primitive.ObjectID) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	filter := bson.M{"_id": feedID}

	result, err := r.collection.DeleteOne(ctx, filter)
	if err != nil {
		return err
	}

	if result.DeletedCount == 0 {
		return fmt.Errorf("feed with ID %s not found", feedID.Hex())
	}

	return nil
}
