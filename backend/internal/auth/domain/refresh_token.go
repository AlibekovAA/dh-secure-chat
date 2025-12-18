package domain

import "time"

type RefreshToken struct {
	ID        string
	TokenHash string
	UserID    string
	ExpiresAt time.Time
	CreatedAt time.Time
	RawToken  string
}
