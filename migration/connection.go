package migration

import (
	"context"

	"github.com/go-redis/redis/v8"
	"github.com/sirupsen/logrus"
)

func InitRedisClient(addr string, pass string) *redis.Client {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: pass, // no password set
		DB:       0,    // use default DB
	})
	_, err := client.Ping(context.Background()).Result()
	if err != nil {
		logrus.Error("REDIS CONNECTION: FAILED... | err", err)
		return nil
	}
	logrus.Info("REDIS CONNECTION: SUCCESS...")
	return client
}
