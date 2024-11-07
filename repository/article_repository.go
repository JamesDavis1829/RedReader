package repository

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
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
