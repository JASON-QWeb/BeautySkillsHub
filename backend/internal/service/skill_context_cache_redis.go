package service

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

const defaultSkillsContextTTL = 10 * time.Minute

type RedisSkillContextCache struct {
	client *redis.Client
}

func NewRedisSkillContextCache(client *redis.Client) *RedisSkillContextCache {
	return &RedisSkillContextCache{client: client}
}

func (c *RedisSkillContextCache) Get(ctx context.Context, key string) (string, error) {
	value, err := c.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return value, nil
}

func (c *RedisSkillContextCache) Set(ctx context.Context, key string, value string) error {
	return c.client.Set(ctx, key, value, defaultSkillsContextTTL).Err()
}

func (c *RedisSkillContextCache) PublishInvalidate(ctx context.Context, channel string, payload string) error {
	return c.client.Publish(ctx, channel, payload).Err()
}
