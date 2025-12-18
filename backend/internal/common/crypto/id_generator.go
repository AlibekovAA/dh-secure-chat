package crypto

import "github.com/google/uuid"

type IDGenerator interface {
	NewID() (string, error)
}

type UUIDGenerator struct{}

func NewUUIDGenerator() *UUIDGenerator {
	return &UUIDGenerator{}
}

func (g *UUIDGenerator) NewID() (string, error) {
	return uuid.NewString(), nil
}
