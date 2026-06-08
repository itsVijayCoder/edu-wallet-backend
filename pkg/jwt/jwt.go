package jwt

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrExpiredToken = errors.New("token has expired")
)

// Claims holds the custom claims for access tokens.
type Claims struct {
	UserID      uuid.UUID  `json:"user_id"`
	Email       string     `json:"email"`
	Roles       []string   `json:"roles"`
	TenantID    *uuid.UUID `json:"tenant_id,omitempty"`
	Permissions []string   `json:"permissions,omitempty"`
	jwt.RegisteredClaims
}

// TokenManager handles JWT generation and validation.
type TokenManager interface {
	GenerateAccess(userID uuid.UUID, email string, roles []string) (string, error)
	GenerateTenantAccess(userID uuid.UUID, email string, roles []string, tenantID uuid.UUID, permissions []string) (string, error)
	GenerateRefresh(userID uuid.UUID) (string, error)
	ValidateAccess(tokenStr string) (*Claims, error)
	ValidateRefresh(tokenStr string) (*Claims, error)
}

type tokenManager struct {
	accessSecret  string
	refreshSecret string
	accessExpiry  time.Duration
	refreshExpiry time.Duration
	issuer        string
}

func NewTokenManager(accessSecret, refreshSecret string, accessExpiry, refreshExpiry time.Duration, issuer string) TokenManager {
	return &tokenManager{
		accessSecret:  accessSecret,
		refreshSecret: refreshSecret,
		accessExpiry:  accessExpiry,
		refreshExpiry: refreshExpiry,
		issuer:        issuer,
	}
}

func (m *tokenManager) GenerateAccess(userID uuid.UUID, email string, roles []string) (string, error) {
	return m.generateAccess(userID, email, roles, nil, nil)
}

func (m *tokenManager) GenerateTenantAccess(
	userID uuid.UUID,
	email string,
	roles []string,
	tenantID uuid.UUID,
	permissions []string,
) (string, error) {
	return m.generateAccess(userID, email, roles, &tenantID, permissions)
}

func (m *tokenManager) generateAccess(
	userID uuid.UUID,
	email string,
	roles []string,
	tenantID *uuid.UUID,
	permissions []string,
) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID:      userID,
		Email:       email,
		Roles:       roles,
		TenantID:    tenantID,
		Permissions: permissions,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(m.accessExpiry)),
			IssuedAt:  jwt.NewNumericDate(now),
			ID:        uuid.New().String(),
			Issuer:    m.issuer,
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(m.accessSecret))
}

func (m *tokenManager) GenerateRefresh(userID uuid.UUID) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(m.refreshExpiry)),
			IssuedAt:  jwt.NewNumericDate(now),
			ID:        uuid.New().String(),
			Issuer:    m.issuer,
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(m.refreshSecret))
}

func (m *tokenManager) ValidateAccess(tokenStr string) (*Claims, error) {
	return m.validate(tokenStr, m.accessSecret)
}

func (m *tokenManager) ValidateRefresh(tokenStr string) (*Claims, error) {
	return m.validate(tokenStr, m.refreshSecret)
}

func (m *tokenManager) validate(tokenStr, secret string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return []byte(secret), nil
	})
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	// Validate issuer
	if m.issuer != "" && claims.Issuer != m.issuer {
		return nil, ErrInvalidToken
	}

	return claims, nil
}
