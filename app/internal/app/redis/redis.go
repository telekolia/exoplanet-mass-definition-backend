package redis

import (
	"app/internal/app/config"
	"context"
	"fmt"
	"net"

	redisv8 "github.com/go-redis/redis/v8"
)

type Client struct {
	client *redisv8.Client
}

func New(ctx context.Context, cfg config.RedisConfig) (*Client, error) {
	client := redisv8.NewClient(&redisv8.Options{
		Addr:     net.JoinHostPort(cfg.Host, cfg.Port),
		Username: cfg.User, Password: cfg.Password,
		DialTimeout: cfg.DialTimeout, ReadTimeout: cfg.ReadTimeout, WriteTimeout: cfg.WriteTimeout,
	})
	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("подключение к Redis: %w", err)
	}
	return &Client{client: client}, nil
}

func (c *Client) Close() error {
	return c.client.Close()
}
