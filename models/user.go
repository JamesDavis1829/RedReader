package models

import (
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type User struct {
	ID            string               `json:"id" bson:"id"`
	Email         string               `json:"email" bson:"email"`
	Name          string               `json:"name" bson:"name"`
	Tokens        []string             `json:"tokens" bson:"tokens"`
	SubscribedTo  []string             `json:"subscribedTo" bson:"subscribedTo"`   // Array of Feed IDs
	PersonalFeeds []primitive.ObjectID `json:"personalFeeds" bson:"personalFeeds"` // Array of Feed IDs
}

func NewUser(email, name string) *User {
	return &User{
		ID:            uuid.New().String(),
		Email:         email,
		Name:          name,
		Tokens:        make([]string, 0),
		SubscribedTo:  make([]string, 0),
		PersonalFeeds: make([]primitive.ObjectID, 0),
	}
}
