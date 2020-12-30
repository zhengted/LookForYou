package redisConn

import (
	"github.com/garyburd/redigo/redis"
)

func Conn() *redis.Pool {
	pool := &redis.Pool{
		Dial: func() (redis.Conn, error) {
			return redis.Dial("tcp", "127.0.0.1:16379")
		},
		MaxActive: 100,
	}
	return pool
}
