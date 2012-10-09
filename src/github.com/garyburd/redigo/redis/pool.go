// Copyright 2012 Gary Burd
//
// Licensed under the Apache License, Version 2.0 (the "License"): you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
// WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
// License for the specific language governing permissions and limitations
// under the License.

package redis

import (
	"errors"
)

// Pool maintains a pool of connections.
//
// Pooled connections do not support concurrent access or pub/sub.
//
// The following example shows how to use a pool in a web application. The
// application creates a pool at application startup and makes it available to
// request handlers, possibly using a global variable:
//
//      var server string           // host:port of server
//      var password string
//      ...
//
//      pool = redis.NewPool(func () (c redis.Conn, err error) {
//              c, err = redis.Dial(server)
//              if err != nil {
//                  err = c.Do("AUTH", password)
//              }
//              return
//          }, 3)
//
// This pool has a maximum of three connections to the server specified by the
// variable "server". Each connection is authenticated using a password.
//
// A request handler gets a connection from the pool and closes the connection
// when the handler is done:
//
//  conn, err := pool.Get()
//  if err != nil {
//      // handle the error
//  }
//  defer conn.Close()
//  // do something with the connection
//
// Close() returns the connection to the pool if there's room in the pool and
// the connection does not have a permanent error. Otherwise, Close() releases
// the resources used by the connection.
type Pool struct {
	newFn func() (Conn, error)
	conns chan Conn
}

type pooledConnection struct {
	c    Conn
	err  error
	pool *Pool
}

// NewPool returns a new connection pool. The pool uses newFn to create
// connections as needed and maintains a maximum of maxIdle idle connections.
func NewPool(newFn func() (Conn, error), maxIdle int) *Pool {
	return &Pool{newFn: newFn, conns: make(chan Conn, maxIdle)}
}

// Get returns an idle connection from the pool if available or creates a new
// connection. The caller should Close() the connection to return the
// connection to the pool.
func (p *Pool) Get() (Conn, error) {
	var c Conn
	select {
	case c = <-p.conns:
	default:
		var err error
		c, err = p.newFn()
		if err != nil {
			return nil, err
		}
	}
	return &pooledConnection{c: c, pool: p}, nil
}

func (c *pooledConnection) Err() error {
	if c.err != nil {
		return c.err
	}
	return c.c.Err()
}

func (c *pooledConnection) Do(commandName string, args ...interface{}) (reply interface{}, err error) {
	if c.err != nil {
		return nil, c.err
	}
	return c.c.Do(commandName, args...)
}

func (c *pooledConnection) Send(commandName string, args ...interface{}) error {
	if c.err != nil {
		return c.err
	}
	return c.c.Send(commandName, args...)
}

func (c *pooledConnection) Flush() error {
	if c.err != nil {
		return c.err
	}
	return c.c.Flush()
}

func (c *pooledConnection) Receive() (reply interface{}, err error) {
	if c.err != nil {
		return nil, c.err
	}
	return c.c.Receive()
}

var errPoolClosed = errors.New("redigo: pooled connection closed")

func (c *pooledConnection) Close() (err error) {
	if c.err != nil {
		return c.err
	}
	c.err = errPoolClosed
	if c.c.Err() != nil {
		return c.c.Close()
	}
	select {
	case c.pool.conns <- c.c:
	default:
		return c.c.Close()
	}
	return nil
}
