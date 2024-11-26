package repository

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"redapplications.com/redreader/models"
)

type ArticleRepository struct {
	collection *mongo.Collection
}

type ArticleWithFeed struct {
	models.Article `bson:",inline"`
	FeedTitle      string `bson:"feedTitle"`
}

func NewArticleRepository(client *mongo.Client) *ArticleRepository {
	collection := client.Database("redreader").Collection("articles")
	return &ArticleRepository{collection: collection}
}

func (r *ArticleRepository) CreateArticle(article *models.Article) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := r.collection.InsertOne(ctx, article)
	return err
}

func (r *ArticleRepository) ArticleExists(url string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	count, err := r.collection.CountDocuments(ctx, bson.M{"url": url})
	return count > 0, err
}

func (r *ArticleRepository) GetPaginatedArticlesByFeed(feedId string, page, perPage int64) ([]*ArticleWithFeed, int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	skip := (page - 1) * perPage
	filter := bson.M{"feedId": feedId}

	total, err := r.collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	findOptions := options.Find().
		SetSort(bson.D{{Key: "publishedAt", Value: -1}}).
		SetSkip(skip).
		SetLimit(perPage)

	cursor, err := r.collection.Find(ctx, filter, findOptions)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var articles []*ArticleWithFeed
	if err = cursor.All(ctx, &articles); err != nil {
		return nil, 0, err
	}

	return articles, total, nil
}

func (r *ArticleRepository) GetPaginatedArticles(page, perPage int64) ([]*ArticleWithFeed, int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	skip := (page - 1) * perPage

	total, err := r.collection.CountDocuments(ctx, bson.M{})
	if err != nil {
		return nil, 0, err
	}

	pipeline := []bson.M{
		{
			"$lookup": bson.M{
				"from": "feeds",
				"let":  bson.M{"feedId": "$feedId"},
				"pipeline": []bson.M{
					{
						"$match": bson.M{
							"$expr": bson.M{
								"$and": []bson.M{
									{
										"$eq": []interface{}{
											"$_id",
											bson.M{"$toObjectId": "$$feedId"},
										},
									},
									{
										"$eq": []interface{}{
											"$isDefault",
											true,
										},
									},
								},
							},
						},
					},
				},
				"as": "feed",
			},
		},
		{
			"$unwind": "$feed",
		},
		{
			"$addFields": bson.M{
				"feedTitle": "$feed.title",
			},
		},
		{
			"$sort": bson.M{"publishedAt": -1},
		},
		{
			"$skip": skip,
		},
		{
			"$limit": perPage,
		},
	}

	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var articles []*ArticleWithFeed
	if err = cursor.All(ctx, &articles); err != nil {
		return nil, 0, err
	}

	return articles, total, nil
}

func (r *ArticleRepository) GetPaginatedArticlesForUser(user *models.User, page, perPage int64) ([]*ArticleWithFeed, int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// If user has no subscriptions, return empty result
	if len(user.SubscribedTo) == 0 {
		return []*ArticleWithFeed{}, 0, nil
	}

	skip := (page - 1) * perPage
	matchStage := bson.M{
		"$match": bson.M{
			"feedId": bson.M{
				"$in": user.SubscribedTo,
			},
		},
	}

	// Count total matching documents
	countPipeline := []bson.M{
		matchStage,
		{
			"$count": "total",
		},
	}

	countCursor, err := r.collection.Aggregate(ctx, countPipeline)
	if err != nil {
		return nil, 0, err
	}
	defer countCursor.Close(ctx)

	var countResult []struct{ Total int64 }
	if err = countCursor.All(ctx, &countResult); err != nil {
		return nil, 0, err
	}

	total := int64(0)
	if len(countResult) > 0 {
		total = countResult[0].Total
	}

	// Get paginated articles
	pipeline := []bson.M{
		matchStage,
		{
			"$lookup": bson.M{
				"from": "feeds",
				"let":  bson.M{"feedId": "$feedId"},
				"pipeline": []bson.M{
					{
						"$match": bson.M{
							"$expr": bson.M{
								"$eq": []interface{}{
									"$_id",
									bson.M{"$toObjectId": "$$feedId"},
								},
							},
						},
					},
				},
				"as": "feed",
			},
		},
		{
			"$unwind": "$feed",
		},
		{
			"$addFields": bson.M{
				"feedTitle": "$feed.title",
			},
		},
		{
			"$sort": bson.M{"publishedAt": -1},
		},
		{
			"$skip": skip,
		},
		{
			"$limit": perPage,
		},
	}

	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var articles []*ArticleWithFeed
	if err = cursor.All(ctx, &articles); err != nil {
		return nil, 0, err
	}

	return articles, total, nil
}

func (r *ArticleRepository) GetArticleContent(id string) (*ArticleWithFeed, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pipeline := []bson.M{
		{
			"$match": bson.M{"_id": id},
		},
		{
			"$lookup": bson.M{
				"from": "feeds",
				"let":  bson.M{"feedId": "$feedId"},
				"pipeline": []bson.M{
					{
						"$match": bson.M{
							"$expr": bson.M{
								"$eq": []interface{}{
									"$_id",
									bson.M{"$toObjectId": "$$feedId"},
								},
							},
						},
					},
				},
				"as": "feed",
			},
		},
		{
			"$unwind": "$feed",
		},
		{
			"$addFields": bson.M{
				"feedTitle": "$feed.title",
			},
		},
	}

	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var articles []*ArticleWithFeed
	if err = cursor.All(ctx, &articles); err != nil {
		return nil, err
	}

	if len(articles) == 0 {
		return nil, mongo.ErrNoDocuments
	}

	return articles[0], nil
}
