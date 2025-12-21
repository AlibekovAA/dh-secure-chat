package crypto

import "golang.org/x/crypto/bcrypt"

type PasswordHasher interface {
	Hash(password string) (string, error)
	Compare(hash string, password string) error
}

type BcryptHasher struct{}

func (h *BcryptHasher) Hash(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

func (h *BcryptHasher) Compare(hash string, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}
