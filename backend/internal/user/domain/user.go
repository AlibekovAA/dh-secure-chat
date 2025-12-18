package domain

import "time"

type ID string

type User struct {
	ID           ID
	Username     string
	PasswordHash string
	CreatedAt    time.Time
	LastSeenAt   *time.Time
}

type Summary struct {
	ID        ID
	Username  string
	CreatedAt time.Time
}
