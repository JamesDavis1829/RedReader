package models

import (
	"strings"
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

const (
	maxDescriptionLength = 1000
)

func NewArticle(feedID string) *Article {
	return &Article{
		ID:        uuid.New().String(),
		FeedID:    feedID,
		CreatedAt: time.Now(),
	}
}

func (a *Article) ShouldShowDescription() bool {
	// Don't show if description is empty
	if a.Description == "" {
		return false
	}

	// Don't show if description is the same as content
	if a.Description == a.Content {
		return false
	}

	// Don't show if description is too long (more than 300 chars) and contains html
	if len(a.Description) > maxDescriptionLength && strings.Contains(a.Description, "<") && strings.Contains(a.Description, ">") {
		return false
	}

	return true
}

func (a *Article) TruncatedDescription() string {
	if len(a.Description) <= maxDescriptionLength {
		return a.Description
	}
	return strings.TrimSpace(a.Description[:297]) + "..."
}
