package models

import (
	"time"

	"github.com/google/uuid"
)

type Article struct {
	ID          string    `json:"id" bson:"_id"`
	FeedID      string    `json:"feedId" bson:"feedId"`
	Title       string    `json:"title" bson:"title"`
	Description string    `json:"description" bson:"description"`
	Content     string    `json:"content" bson:"content"`
	URL         string    `json:"url" bson:"url"`
	Author      string    `json:"author" bson:"author"`
	PublishedAt time.Time `json:"publishedAt" bson:"publishedAt"`
	CreatedAt   time.Time `json:"createdAt" bson:"createdAt"`
}

func NewArticle(feedID string) *Article {
	return &Article{
		ID:        uuid.New().String(),
		FeedID:    feedID,
		CreatedAt: time.Now(),
	}
}
