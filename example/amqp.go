package main

import (
	"fmt"
	"time"

	"github.com/fanjindong/pool"

	"github.com/streadway/amqp"
)

func main() {
	// Conn, err := amqp.Dial("amqp://uki:Neoclub2018@192.168.3.3:5672/")
	Conn, err := amqp.Dial("amqp://root:password@127.0.0.1:5672/")
	if err != nil {
		panic("Failed to connect to RabbitMQ")
	}
	dialerf := func() (interface{}, error) { return Conn.Channel() }
	closef := func(v interface{}) error { return v.(*amqp.Channel).Close() }

	opt := &pool.Options{
		PoolSize: 10,
		Dialer:   dialerf,
		OnClose:  closef,
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
