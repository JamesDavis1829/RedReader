package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Feed struct {
	ID           primitive.ObjectID `json:"id" bson:"_id"`
	URL          string             `json:"url" bson:"url"`
	Title        string             `json:"title" bson:"title"`
	Description  string             `json:"description" bson:"description"`
	LastFetched  time.Time          `json:"lastFetched" bson:"lastFetched"`
	IsSubscribed bool               `json:"isSubscribed" bson:"-"`
	IsDefault    bool               `json:"isDefault" bson:"isDefault"`
}

func NewFeed(url string) *Feed {
	return &Feed{
		ID:          primitive.NewObjectID(),
		URL:         url,
		LastFetched: time.Time{},
		IsDefault:   false,
	}
}
