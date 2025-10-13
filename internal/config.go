package internal

import (
	"strconv"
	"time"
)

// Configuration provides the different items we can use to
// configure how we connect to the database
type Configuration struct {
	MysqlHost       string        `json:"mysql_host"`
	MysqlPort       string        `json:"mysql_port"`
	MysqlUsername   string        `json:"mysql_username"`
	MysqlPassword   string        `json:"mysql_password"`
	MysqlDatabase   string        `json:"mysql_database"`
	MysqlParseTime  bool          `json:"mysql_parse_time"`
	RedisHost       string        `json:"redis_host"`
	RedisPort       string        `json:"redis_port"`
	RedisUsername   string        `json:"redis_username"`
	RedisPassword   string        `json:"redis_password"`
	RedisDatabase   int           `json:"redis_database"`
	RedisTimeout    time.Duration `json:"redis_timeout"`
	MutexType       string        `json:"mutex_type"`
	GoRoutines      int           `json:"go_routines"`
	DemoDuration    time.Duration `json:"demo_duration"`
	MutateInterval  time.Duration `json:"mutate_interval"`
	RetryInterval   time.Duration `json:"retry_interval"`
	MutexExpiration time.Duration `json:"mutex_expiration"`
}

// ConfigFromEnv can be used to generate a configuration pointer
// from a list of environments, it'll set the default configuraton
// as well
func ConfigFromEnv(envs map[string]string) *Configuration {
	c := &Configuration{
		MysqlHost:       "localhost",
		MysqlPort:       "3306",
		MysqlUsername:   "root",
		MysqlPassword:   "mysql",
		MysqlDatabase:   "go_blog_distributed_mutex",
		MysqlParseTime:  false,
		RedisHost:       "localhost",
		RedisPort:       "6379",
		RedisUsername:   "go_blog_distributed_mutex",
		RedisPassword:   "go_blog_distributed_mutex",
		MutexType:       "redis",
		GoRoutines:      2,
		DemoDuration:    10 * time.Second,
		MutateInterval:  1000 * time.Millisecond,
		RetryInterval:   time.Millisecond,
		MutexExpiration: 10 * time.Second,
	}
	if host, ok := envs["MYSQL_HOST"]; ok {
		c.MysqlHost = host
	}
	if port, ok := envs["MYSQL_PORT"]; ok {
		c.MysqlPort = port
	}
	if username, ok := envs["MYSQL_USERNAME"]; ok {
		c.MysqlUsername = username
	}
	if password, ok := envs["MYSQL_PASSWORD"]; ok {
		c.MysqlPassword = password
	}
	if database, ok := envs["MYSQL_DATABASE"]; ok {
		c.MysqlDatabase = database
	}
	if redisAddress, ok := envs["REDIS_ADDRESS"]; ok {
		c.RedisHost = redisAddress
	}
	if redisPort, ok := envs["REDIS_PORT"]; ok {
		c.RedisPort = redisPort
	}
	if redisUsername, ok := envs["REDIS_USERNAME"]; ok {
		c.RedisUsername = redisUsername
	}
	if redisPassword, ok := envs["REDIS_PASSWORD"]; ok {
		c.RedisPassword = redisPassword
	}
	if redisDatabase, ok := envs["REDIS_DATABASE"]; ok {
		i, _ := strconv.ParseInt(redisDatabase, 10, 64)
		c.RedisDatabase = int(i)
	}
	if redisTimeout, ok := envs["REDIS_TIMEOUT"]; ok {
		i, _ := strconv.ParseInt(redisTimeout, 10, 64)
		c.RedisTimeout = time.Duration(i) * time.Second
	}
	if retryInterval, ok := envs["RETRY_INTERVAL"]; ok {
		i, _ := strconv.ParseInt(retryInterval, 10, 64)
		c.RetryInterval = time.Duration(i) * time.Millisecond
	}
	if mutexType, ok := envs["MUTEX_TYPE"]; ok {
		c.MutexType = mutexType
	}
	if mutexExpiration, ok := envs["MUTEX_EXPIRATION"]; ok {
		i, _ := strconv.ParseInt(mutexExpiration, 10, 64)
		c.MutexExpiration = time.Duration(i) * time.Second
	}
	return c
}
