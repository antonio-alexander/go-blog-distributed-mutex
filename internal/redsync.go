package internal

import (
	"context"
	"fmt"
	"time"

	redsync "github.com/go-redsync/redsync/v4"
	redsyncgoredis "github.com/go-redsync/redsync/v4/redis/goredis/v9"
	redis "github.com/redis/go-redis/v9"
)

const hashKeyRedSyncMutex string = "redsync_mutex"

type RedisRedSyncMutex struct {
	config struct {
		retryInterval time.Duration
	}
	errorHandler func(error)
	redisClient  *redis.Client
	*redsync.Mutex
}

func NewRedSyncMutex(config *Configuration) (*RedisRedSyncMutex, error) {
	r := &RedisRedSyncMutex{
		errorHandler: func(err error) {
			fmt.Printf("redis error: %s\n", err.Error())
		},
	}
	address := config.RedisHost
	if config.RedisPort != "" {
		address = address + ":" + config.RedisPort
	}
	redisClient := redis.NewClient(&redis.Options{
		Addr:     address,
		Password: config.RedisPassword,
		DB:       config.RedisDatabase,
	})
	if err := redisClient.Ping(context.Background()).Err(); err != nil {
		return nil, err
	}
	redisPool := redsyncgoredis.NewPool(redisClient)
	redisMutex := redsync.New(redisPool).NewMutex(hashKeyRedSyncMutex,
		redsync.WithExpiry(config.MutexExpiration))
	r.redisClient, r.Mutex = redisClient, redisMutex
	r.config.retryInterval = config.RetryInterval
	return r, nil
}

func (r *RedisRedSyncMutex) Close() error {
	return r.redisClient.Close()
}

func (r *RedisRedSyncMutex) Lock() {
	if err := r.Mutex.Lock(); err != nil {
		r.errorHandler(err)
	} else {
		return
	}
	tRetry := time.NewTicker(r.config.retryInterval)
	defer tRetry.Stop()
	for {
		<-tRetry.C
		if err := r.Mutex.Lock(); err != nil {
			r.errorHandler(err)
			continue
		}
		return
	}
}

func (r *RedisRedSyncMutex) Unlock() {
	unlocked, err := r.Mutex.Unlock()
	if err != nil {
		r.errorHandler(err)
	} else {
		if !unlocked {
			panic("attempted to unlock an unlocked mutex")
		} else {
			return
		}
	}
	tRetry := time.NewTicker(r.config.retryInterval)
	defer tRetry.Stop()
	for {
		<-tRetry.C
		if unlocked, err = r.Mutex.Unlock(); err != nil {
			r.errorHandler(err)
			continue
		}
		if !unlocked {
			panic("attempted to unlock an unlocked mutex")
		} else {
			return
		}
	}
}
