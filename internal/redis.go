package internal

import (
	"context"
	"fmt"
	"time"

	redis "github.com/redis/go-redis/v9"
)

const hashKeyRedisMutex = "redis_mutex"

type RedisMutex struct {
	config struct {
		retryInterval   time.Duration
		mutexExpiration time.Duration
	}
	ctx          context.Context
	cancel       context.CancelFunc
	redisClient  *redis.Client
	errorHandler func(error)
}

func NewRedisMutex(config *Configuration) (*RedisMutex, error) {
	r := &RedisMutex{
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
		Username: config.RedisUsername,
		Password: config.RedisPassword,
		DB:       config.RedisDatabase,
	})
	ctx, cancel := context.WithCancel(context.Background())
	if err := redisClient.Ping(ctx).Err(); err != nil {
		cancel()
		return nil, err
	}
	r.ctx, r.cancel = ctx, cancel
	r.redisClient = redisClient
	r.config.mutexExpiration = config.MutexExpiration
	r.config.retryInterval = config.RetryInterval
	return r, nil
}

func (r *RedisMutex) Close() error {
	r.cancel()
	return r.redisClient.Close()
}

func (r *RedisMutex) Lock() {
	lockFx := func() (bool, error) {
		result, err := r.redisClient.SetNX(r.ctx,
			hashKeyRedisMutex, true, r.config.mutexExpiration).Result()
		if err != nil {
			return false, err
		}
		return result, nil
	}
	locked, err := lockFx()
	if err != nil {
		r.errorHandler(err)
	}
	if locked {
		return
	}
	tRetry := time.NewTicker(r.config.retryInterval)
	defer tRetry.Stop()
	for {
		<-tRetry.C
		if locked, err = lockFx(); err != nil {
			r.errorHandler(err)
			continue
		}
		if locked {
			return
		}
	}
}

func (r *RedisMutex) Reset() error {
	_, err := r.redisClient.Del(r.ctx,
		hashKeyRedisMutex).Result()
	if err != nil {
		return err
	}
	return nil
}

func (r *RedisMutex) Unlock() {
	unlockFx := func() (bool, error) {
		script := `
			local key = KEYS[1]
			local expected_value = ARGV[1]

			local current_value = redis.call('GET', key)

			if current_value == expected_value then
			    return redis.call('DEL', key)
			else
		    	return 0 -- Key not deleted (value did not match)
			end
		`
		item, err := r.redisClient.Eval(r.ctx, script,
			[]string{hashKeyRedisMutex}, true).Result()
		if err != nil {
			return false, err
		}
		i, ok := item.(int64)
		if !ok {
			return false, nil
		}
		return i == 1, nil
	}
	unlocked, err := unlockFx()
	if err != nil {
		r.errorHandler(err)
	} else {
		if !unlocked {
			panic("attempted to unlock an unlocked mutex")
		}
		return
	}
	tRetry := time.NewTicker(r.config.retryInterval)
	defer tRetry.Stop()
	for {
		<-tRetry.C
		if unlocked, err = unlockFx(); err != nil {
			r.errorHandler(err)
			continue
		}
		if !unlocked {
			panic("attempted to unlock an unlocked mutex")
		}
		return
	}
}
