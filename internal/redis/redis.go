package redis

import (
	"context"
	"fmt"
	"github.com/redis/go-redis/v9"
)

type RedisConn struct {
	addr     string
	password string
	db       int
}

func NewRedisConn(addr string, password string, db int) *RedisConn {
	return &RedisConn{
		addr:     addr,
		password: password,
		db:       db,
	}
}

func (r *RedisConn) Connect() (*redis.Client, error) {

	rdb := redis.NewClient(&redis.Options{
		Addr:     r.addr,
		Password: r.password,
		DB:       r.db,
	})

	result, err := rdb.Ping(context.Background()).Result()
	if err != nil {
		return nil, err
	}

	if result != "PONG" {
		return nil, fmt.Errorf("redis connection error: %s\n", result)
	}

	return rdb, nil
}
