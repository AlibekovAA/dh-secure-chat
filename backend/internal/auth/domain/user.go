package domain

import "time"

type UserID string

type User struct {
	ID           UserID
	Username     string
	PasswordHash string
	CreatedAt    time.Time
}
