package session

import (
	"app/internal/app/ds"
	"app/internal/app/role"
	"errors"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const issuer = "telescopes"

type Tokens struct {
	secret   []byte
	lifetime time.Duration
}

type LoginResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

func NewTokens(secret string, lifetime time.Duration) (*Tokens, error) {
	if len(secret) < 32 {
		return nil, errors.New("JWT_SECRET должен содержать не менее 32 байт")
	}
	if lifetime < time.Second {
		return nil, errors.New("срок действия JWT должен быть не менее секунды")
	}
	return &Tokens{secret: []byte(secret), lifetime: lifetime}, nil
}

func (t *Tokens) Issue(userID uint) (LoginResponse, error) {
	now := time.Now()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, &ds.JWTClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer: issuer, Subject: strconv.FormatUint(uint64(userID), 10),
			ID: uuid.NewString(), IssuedAt: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(t.lifetime)),
		},
		UserID: userID, Role: role.Creator,
	})
	signed, err := token.SignedString(t.secret)
	return LoginResponse{AccessToken: signed, TokenType: "Bearer", ExpiresIn: int(t.lifetime.Seconds())}, err
}

func (t *Tokens) Parse(raw string) (*ds.JWTClaims, error) {
	claims := &ds.JWTClaims{}
	_, err := jwt.ParseWithClaims(raw, claims, func(*jwt.Token) (interface{}, error) {
		return t.secret, nil
	}, jwt.WithValidMethods([]string{"HS256"}), jwt.WithIssuer(issuer), jwt.WithExpirationRequired())
	if err != nil {
		return nil, err
	}
	if claims.UserID == 0 || claims.Subject != strconv.FormatUint(uint64(claims.UserID), 10) || claims.ID == "" {
		return nil, errors.New("некорректные данные JWT")
	}
	return claims, nil
}
