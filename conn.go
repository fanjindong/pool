package pool

import (
	"net"
	"sync/atomic"
	"time"
)

var noDeadline = time.Time{}

type Conn struct {
	conn      interface{}
	Inited    bool
	pooled    bool
	createdAt time.Time
	usedAt    atomic.Value
}

func NewConn(conn interface{}) *Conn {
	cn := &Conn{
		conn:      conn,
		createdAt: time.Now(),
	}
	cn.SetUsedAt(time.Now())
	return cn
}

func (cn *Conn) UsedAt() time.Time {
	return cn.usedAt.Load().(time.Time)
}

func (cn *Conn) SetUsedAt(tm time.Time) {
	cn.usedAt.Store(tm)
}

func (cn *Conn) SetNetConn(netConn net.Conn) {
	cn.conn = conn
}
