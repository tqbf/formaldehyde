package main

import (
	"github.com/garyburd/redigo/redis"
)

type RedisRequest func(r *RedisLand)

type RedisLand struct {
	Conn	redis.Conn
	Comm	chan RedisRequest
}

func NewRedisLand(addr string) (*RedisLand, error) {
	conn, err := redis.Dial("tcp", addr)
	if err != nil {
		return nil, err
	}

	r := new(RedisLand)
	r.Comm = make(chan RedisRequest)
	r.Conn = conn
	
	return r, nil
}

func (r *RedisLand) Loop() {
	for {
		req := <- r.Comm
		req(r)
	}
}

