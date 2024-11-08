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

func (r *ArticleRepository) GetPaginatedArticlesByFeed(feedId string, page, perPage int64) ([]*models.Article, int64, error) {
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

	var articles []*models.Article
	if err = cursor.All(ctx, &articles); err != nil {
		return nil, 0, err
	}

	return articles, total, nil
}

func (r *ArticleRepository) GetArticleContent(id string) (*models.Article, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var article models.Article
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&article)
	if err != nil {
		return nil, err
	}

	return &article, nil
}

func (r *ArticleRepository) GetPaginatedArticles(page, perPage int64) ([]*models.Article, int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	skip := (page - 1) * perPage

	total, err := r.collection.CountDocuments(ctx, bson.M{})
	if err != nil {
		return nil, 0, err
	}

	findOptions := options.Find().
		SetSort(bson.D{{Key: "publishedAt", Value: -1}}).
		SetSkip(skip).
		SetLimit(perPage)

	cursor, err := r.collection.Find(ctx, bson.M{}, findOptions)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var articles []*models.Article
	if err = cursor.All(ctx, &articles); err != nil {
		return nil, 0, err
	}

	return articles, total, nil
}
