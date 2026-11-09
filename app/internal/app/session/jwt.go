package session

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const issuer = "telescopes"
const tokenLifetime = time.Hour
const blacklistPrefix = "telescopes.jwt.blacklist:"

type Tokens struct {
	secret []byte
	redis  *redis.Client
}

type LoginResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

func NewTokens(secret string, options *redis.Options) (*Tokens, error) {
	if len(secret) < 32 {
		return nil, errors.New("JWT_SECRET должен содержать не менее 32 байт")
	}
	return &Tokens{secret: []byte(secret), redis: redis.NewClient(options)}, nil
}

func (t *Tokens) Close() error {
	return t.redis.Close()
}

func (t *Tokens) Issue(userID uint) (LoginResponse, error) {
	now := time.Now()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Issuer: issuer, Subject: strconv.FormatUint(uint64(userID), 10),
		ID: uuid.NewString(), IssuedAt: jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(tokenLifetime)),
	})
	signed, err := token.SignedString(t.secret)
	return LoginResponse{AccessToken: signed, TokenType: "Bearer", ExpiresIn: int(tokenLifetime.Seconds())}, err
}

func (t *Tokens) Parse(raw string) (uint, *jwt.RegisteredClaims, error) {
	claims := &jwt.RegisteredClaims{}
	_, err := jwt.ParseWithClaims(raw, claims, func(*jwt.Token) (interface{}, error) {
		return t.secret, nil
	}, jwt.WithValidMethods([]string{"HS256"}), jwt.WithIssuer(issuer), jwt.WithExpirationRequired())
	if err != nil {
		return 0, nil, err
	}
	id, err := strconv.ParseUint(claims.Subject, 10, strconv.IntSize)
	if err != nil || id == 0 || claims.ID == "" {
		return 0, nil, errors.New("некорректные данные JWT")
	}
	return uint(id), claims, nil
}

func (t *Tokens) Revoked(ctx context.Context, id string) (bool, error) {
	count, err := t.redis.Exists(ctx, blacklistPrefix+id).Result()
	return count > 0, err
}

func (t *Tokens) Revoke(ctx context.Context, claims *jwt.RegisteredClaims) error {
	ttl := time.Until(claims.ExpiresAt.Time)
	if ttl <= 0 {
		return nil
	}
	return t.redis.Set(ctx, blacklistPrefix+claims.ID, "1", ttl).Err()
}
