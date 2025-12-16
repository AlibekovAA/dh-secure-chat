package domain

import "time"

type IdentityKey struct {
	UserID    string
	PublicKey []byte
	CreatedAt time.Time
}
