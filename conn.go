package pool

import (
	"net"
	"sync/atomic"
	"time"
)

var noDeadline = time.Time{}

type Conn struct {
	conn   interface{}
	usedAt atomic.Value
}

func NewConn(conn interface{}) *Conn {
	cn := &Conn{
		conn: conn,
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

func (cn *Conn) SetNetConn(conn net.Conn) {
	cn.conn = conn
}
