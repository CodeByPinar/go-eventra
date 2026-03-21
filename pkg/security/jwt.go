package security

import (
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type JWTManager struct {
	secretKey []byte
	expiresIn time.Duration
}

type Claims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	jwt.RegisteredClaims
}

func NewJWTManager(secret string, expiresIn time.Duration) *JWTManager {
	return &JWTManager{
		secretKey: []byte(secret),
		expiresIn: expiresIn,
	}
}

func (m *JWTManager) GenerateToken(userID uuid.UUID, email string) (string, error) {
	now := time.Now().UTC()
	jti := uuid.NewString()
	claims := Claims{
		UserID: userID.String(),
		Email:  email,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(m.expiresIn)),
			Subject:   userID.String(),
			ID:        jti,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(m.secretKey)
}

func (m *JWTManager) GenerateTokenWithClaims(userID uuid.UUID, email string) (string, Claims, error) {
	token, err := m.GenerateToken(userID, email)
	if err != nil {
		return "", Claims{}, err
	}

	claims, err := m.ValidateToken(token)
	if err != nil {
		return "", Claims{}, err
	}

	return token, claims, nil
}

func (m *JWTManager) ValidateToken(rawToken string) (Claims, error) {
	rawToken = strings.TrimSpace(rawToken)
	if rawToken == "" {
		return Claims{}, fmt.Errorf("token is required")
	}

	parsed, err := jwt.ParseWithClaims(rawToken, &Claims{}, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %s", token.Method.Alg())
		}
		return m.secretKey, nil
	})
	if err != nil {
		return Claims{}, err
	}

	claims, ok := parsed.Claims.(*Claims)
	if !ok || !parsed.Valid {
		return Claims{}, fmt.Errorf("invalid token")
	}

	return *claims, nil
}
