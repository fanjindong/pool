package main

import (
	"fmt"
	"pool"
	"time"

	"github.com/streadway/amqp"
)

func main() {
	// Conn, err := amqp.Dial("amqp://uki:Neoclub2018@192.168.3.3:5672/")
	Conn, err := amqp.Dial("amqp://root:password@127.0.0.1:5672/")
	if err != nil {
		panic("Failed to connect to RabbitMQ")
	}
	//factory 创建连接的方法
	factory := func() (interface{}, error) { return Conn.Channel() }
	//close 关闭连接的方法
	close := func(v interface{}) error { return v.(*amqp.Channel).Close() }

	//创建一个连接池： 最小空闲连接数2，最大连接10
	opt := &pool.Options{
		MinIdleConns:       2,
		PoolSize:           10,
		Dialer:             factory,
		OnClose:            close,
		PoolTimeout:        10 * time.Second,
		IdleTimeout:        10 * time.Second,
		IdleCheckFrequency: 10 * time.Second,
	}

	p, err := pool.NewConnPool(opt)
	if err != nil {
		fmt.Println("err=", err)
	}
	//从连接池中取得一个连接
	cn, err := p.Get()
	if err != nil {
		fmt.Println("err=", err)
	}
	fmt.Println("connPool.idleLength:", p.IdleLen())
	p.Put(cn)
	fmt.Println("connPool.idleLength:", p.IdleLen())
	time.Sleep(20 * time.Second)
	fmt.Println("connPool.idleLength:", p.IdleLen())
	opt.MinIdleConns = 1
	time.Sleep(20 * time.Second)
	fmt.Println("connPool.idleLength:", p.IdleLen())
	p.Close()
}
