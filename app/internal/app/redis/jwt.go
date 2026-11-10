package redis

import (
	"context"
	"time"
)

const blacklistPrefix = "telescopes.jwt.blacklist:"

func (c *Client) CheckJWTInBlacklist(ctx context.Context, id string) (bool, error) {
	count, err := c.client.Exists(ctx, blacklistPrefix+id).Result()
	return count > 0, err
}

func (c *Client) WriteJWTToBlacklist(ctx context.Context, id string, expiresAt time.Time) error {
	ttl := time.Until(expiresAt)
	if ttl <= 0 {
		return nil
	}
	return c.client.Set(ctx, blacklistPrefix+id, "1", ttl).Err()
}
