# pool

Golang universal connection pool generator

### Reference

- [go-redis](https://github.com/go-redis/redis)
- [pool](https://github.com/fatih/pool)

### Feature

1. 模板化连接池的创建
2. 允许设置最大连接数量
3. 允许设置最小空闲连接数
4. 允许设置空闲连接的超时过期时间
5. 允许设置时间间隔，清理过期的空闲连接

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