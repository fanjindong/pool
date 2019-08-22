package example

import (
	"fmt"
	"time"

	"github.com/fanjindong/pool"

	"github.com/streadway/amqp"
)

//Conn is ...
var Conn *amqp.Connection

// ChannelPool is ...
var ChannelPool pool.Pool

var err error

// Init is ...
func Init() {

	Conn, err = amqp.Dial("amqp://root:password@127.0.0.1:5672/")
	if err != nil {
		panic("Failed to connect to RabbitMQ")
	}

	//factory 创建连接的方法
	factory := func() (interface{}, error) { return DramatiqConn.Channel() }
	//close 关闭连接的方法
	close := func(v interface{}) error { return v.(*amqp.Channel).Close() }

	//创建一个连接池： 初始化2，最大连接30
	poolConfig := &pool.Config{
		InitialCap: 2,
		MaxCap:     20,
		Factory:    factory,
		Close:      close,
		//Ping:       ping,
		//连接最大空闲时间，超过该时间的连接 将会关闭，可避免空闲时连接EOF，自动失效的问题
		IdleTimeout: 10 * time.Second,
	}
	ChannelPool, err = pool.NewChannelPool(poolConfig)
	if err != nil {
		fmt.Println("err=", err)
	}
	//从连接池中取得一个连接
	v, err := ChannelPool.Get()

	//将连接放回连接池中
	ChannelPool.Put(v)

	//释放连接池中的所有连接
	ChannelPool.Release()

	//查看当前连接中的数量
	current := ChannelPool.Len()
}

// Close is ...
func Close() {
	Conn.Close()
}
