package db

import (
	"fmt"
	"time"

	"github.com/fanjindong/pool"

	"github.com/streadway/amqp"
)

//DramatiqConn is ...
var DramatiqConn *amqp.Connection

// NodeConn is ...
var NodeConn *amqp.Connection

// DramatiqChannelPool is ...
var DramatiqChannelPool pool.Pool

// NodeChannelPool is ...
var NodeChannelPool pool.Pool
var err error

// Init is ...
func Init() {
	config := conf.GetConfig()
	DramatiqConn, err = amqp.Dial(config["amqp_url"].(string))
	if err != nil {
		panic("Failed to connect to RabbitMQ")
	}
	NodeConn, err := amqp.Dial(config["amqp_node_url"].(string))
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
	DramatiqChannelPool, err = pool.NewChannelPool(poolConfig)
	if err != nil {
		fmt.Println("err=", err)
	}

	//factory 创建连接的方法
	factoryNode := func() (interface{}, error) { return NodeConn.Channel() }
	//创建一个连接池： 初始化2，最大连接30
	poolConfigNode := &pool.Config{
		InitialCap: 2,
		MaxCap:     20,
		Factory:    factoryNode,
		Close:      close,
		//Ping:       ping,
		//连接最大空闲时间，超过该时间的连接 将会关闭，可避免空闲时连接EOF，自动失效的问题
		IdleTimeout: 10 * time.Second,
	}
	NodeChannelPool, err = pool.NewChannelPool(poolConfigNode)
	if err != nil {
		fmt.Println("err=", err)
	}
}

// Close is ...
func Close() {
	DramatiqConn.Close()
	NodeConn.Close()
}
