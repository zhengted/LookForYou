package redisConn

import (
	"github.com/garyburd/redigo/redis"
)

func Conn() *redis.Pool {
	pool := &redis.Pool{
		Dial: func() (redis.Conn, error) {
			return redis.Dial("tcp", "0.0.0.0:6379")
		},
		MaxActive: 100,
	}
	return pool
}
