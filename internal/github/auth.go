package github

import (
	"crypto/rsa"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Authenticator struct {
	appID      int64
	privateKey *rsa.PrivateKey
}

func NewAuthenticator(appID int64, privateKeyPath string) (*Authenticator, error) {
	keyBytes, err := os.ReadFile(privateKeyPath)
	if err != nil {
		slog.Error("failed to read private key: %v", err)
		return nil, fmt.Errorf("failed to read private key file: %w", err)
	}

	privateKey, err := jwt.ParseRSAPrivateKeyFromPEM(keyBytes)

	if err != nil {
		slog.Error("failed to parse private key: %v", err)
		return nil, fmt.Errorf("failed to parse RSA private key: %w", err)
	}

	return &Authenticator{
		appID:      appID,
		privateKey: privateKey,
	}, nil
}

func (a *Authenticator) GenerateJWT() (string, error) {
	now := time.Now()
	claims := jwt.MapClaims{
		"iat": now.Add(-60 * time.Second).Unix(), // Account for clock skew
		"exp": now.Add(10 * time.Minute).Unix(),  // GitHub max expiry is 10m
		"iss": a.appID,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	signedToken, err := token.SignedString(a.privateKey)
	if err != nil {
		slog.Error("failed to sign token: %v", err)
		return "", fmt.Errorf("failed to sign JWT: %w", err)
	}

	return signedToken, nil
}
