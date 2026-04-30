package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrInvalidToken     = errors.New("invalid token")
	ErrInvalidSignature = errors.New("invalid token signature")
	ErrExpiredToken     = errors.New("token expired")
)

type JWTService struct {
	secret []byte
	issuer string
	ttl    time.Duration
	nowFn  func() time.Time
}

type TokenInput struct {
	Subject     string
	Role        string
	Email       string
	Provisional map[string]any
}

type TokenClaims struct {
	Role        string         `json:"role"`
	Email       string         `json:"email"`
	Provisional map[string]any `json:"provisional,omitempty"`
	jwt.RegisteredClaims
}

func NewJWTService(secret, issuer string, ttl time.Duration) (*JWTService, error) {
	if secret == "" {
		return nil, fmt.Errorf("secret is required")
	}
	if issuer == "" {
		return nil, fmt.Errorf("issuer is required")
	}
	if ttl <= 0 {
		return nil, fmt.Errorf("ttl must be greater than zero")
	}

	return &JWTService{
		secret: []byte(secret),
		issuer: issuer,
		ttl:    ttl,
		nowFn:  time.Now,
	}, nil
}

func (s *JWTService) GenerateToken(input TokenInput) (string, time.Time, error) {
	if input.Subject == "" {
		return "", time.Time{}, fmt.Errorf("subject is required")
	}

	now := s.nowFn().UTC()
	expiresAt := now.Add(s.ttl)

	claims := TokenClaims{
		Role:        input.Role,
		Email:       input.Email,
		Provisional: input.Provisional,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   input.Subject,
			Issuer:    s.issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	encoded, err := token.SignedString(s.secret)
	if err != nil {
		return "", time.Time{}, err
	}

	return encoded, expiresAt, nil
}

func (s *JWTService) ValidateToken(tokenString string) (*TokenClaims, error) {
	claims := &TokenClaims{}
	parsed, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Method.Alg())
		}
		return s.secret, nil
	}, jwt.WithIssuer(s.issuer), jwt.WithExpirationRequired(), jwt.WithTimeFunc(s.nowFn))
	if err != nil {
		switch {
		case errors.Is(err, jwt.ErrTokenExpired):
			return nil, ErrExpiredToken
		case errors.Is(err, jwt.ErrTokenSignatureInvalid):
			return nil, ErrInvalidSignature
		default:
			return nil, ErrInvalidToken
		}
	}

	if !parsed.Valid {
		return nil, ErrInvalidToken
	}

	return claims, nil
}
