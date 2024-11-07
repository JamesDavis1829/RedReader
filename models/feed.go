package models

import (
	"time"

	"github.com/google/uuid"
)

type Feed struct {
	ID          string    `json:"id" bson:"_id"`
	URL         string    `json:"url" bson:"url"`
	Title       string    `json:"title" bson:"title"`
	Description string    `json:"description" bson:"description"`
	LastFetched time.Time `json:"lastFetched" bson:"lastFetched"`
}

func NewFeed(url string) *Feed {
	return &Feed{
		ID:          uuid.New().String(),
		URL:         url,
		LastFetched: time.Time{},
	}
}
