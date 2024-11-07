package models

import "github.com/google/uuid"

type User struct {
	ID     string   `json:"id"`
	Email  string   `json:"email"`
	Name   string   `json:"name"`
	Tokens []string `json:"tokens"`
}

func NewUser(email, name string) *User {
	return &User{
		ID:     uuid.New().String(),
		Email:  email,
		Name:   name,
		Tokens: make([]string, 0),
	}
}
