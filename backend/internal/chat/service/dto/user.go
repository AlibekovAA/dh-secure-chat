package dto

import "time"

type User struct {
	ID         string
	Username   string
	CreatedAt  time.Time
	LastSeenAt *time.Time
}

type UserSummary struct {
	ID        string
	Username  string
	CreatedAt time.Time
}
