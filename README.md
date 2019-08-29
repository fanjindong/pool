# pool

Golang universal connection pool generator

### Reference

- [go-redis](https://github.com/go-redis/redis)
- [pool](https://github.com/fatih/pool)

### Feature

1. Quickly and easily generate connection pool.
2. Customize the maximum number of connections for this connection pool.
3. Customize the minimum number of idle connections which is useful when establishing.
4. The number of connection pools is automatically scaled between the maximum and minimum values.

### Install

`go get -u -v github.com/fanjindong/pool`

### Demo

```go
package main

import (
	"fmt"
	"github.com/fanjindong/pool"
	"time"

	"github.com/streadway/amqp"
)

func main() {
	Conn, err := amqp.Dial("amqp://root:password@127.0.0.1:5672/")
	if err != nil {
		panic("Failed to connect to RabbitMQ")
	}

	dialerf := func() (interface{}, error) { return Conn.Channel() }
	closef := func(v interface{}) error { return v.(*amqp.Channel).Close() }

	opt := &pool.Options{
		PoolSize:           10,
		Dialer:             dialerf,
		OnClose:            closef,
	}

	p, err := pool.NewConnPool(opt)
	if err != nil {
		fmt.Println("err=", err)
	}

	cn, err := p.Get()
	if err != nil {
		fmt.Println("err=", err)
	}
	p.Put(cn)
	p.Close()
}

```