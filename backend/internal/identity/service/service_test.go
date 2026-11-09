package service

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"testing"

	commonerrors "github.com/AlibekovAA/dh-secure-chat/backend/internal/common/errors"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/logger"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/identity/domain"
)

type identityRepoStub struct{}

func (repo *identityRepoStub) Create(context.Context, domain.IdentityKey) error { return nil }
func (repo *identityRepoStub) FindByUserID(context.Context, string) (domain.IdentityKey, error) {
	return domain.IdentityKey{}, commonerrors.ErrIdentityKeyNotFound
}
func (repo *identityRepoStub) Update(context.Context, string, []byte) error { return nil }

func marshalPublicKey(t *testing.T, key any) []byte {
	t.Helper()
	encoded, err := x509.MarshalPKIXPublicKey(key)
	if err != nil {
		t.Fatal(err)
	}
	return encoded
}

func TestIdentityServiceAcceptsOnlyP256SPKI(t *testing.T) {
	log, err := logger.New(t.TempDir(), "test", "ERROR")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = log.Close() })
	repo := &identityRepoStub{}
	service := NewIdentityService(IdentityServiceDeps{Repo: repo, Log: log})
	p256, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	p384, _ := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	rsaKey, _ := rsa.GenerateKey(rand.Reader, 1024)

	if err := service.CreateIdentityKey(context.Background(), "user", marshalPublicKey(t, &p256.PublicKey)); err != nil {
		t.Fatalf("valid P-256 key rejected: %v", err)
	}
	for name, encoded := range map[string][]byte{
		"random": []byte("this is not a public key but has some bytes"),
		"P-384":  marshalPublicKey(t, &p384.PublicKey),
		"RSA":    marshalPublicKey(t, &rsaKey.PublicKey),
	} {
		t.Run(name, func(t *testing.T) {
			if err := service.UpdatePublicKey(context.Background(), "user", encoded); err == nil {
				t.Fatal("expected invalid public key error")
			}
		})
	}
}
