package pool

import (
	"sync/atomic"
	"time"
)

var noDeadline = time.Time{}

// Conn is struct encapsulating cn
type Conn struct {
	// cn is user's connection object
	cn     interface{}
	usedAt atomic.Value
}

// NewConn is a new Conn
func NewConn(cn interface{}) *Conn {
	conn := &Conn{
		cn: cn,
	}
	conn.SetUsedAt(time.Now())
	return conn
}

// UsedAt is return the conn last used time
func (conn *Conn) UsedAt() time.Time {
	return conn.usedAt.Load().(time.Time)
}

// SetUsedAt is set the conn last used time
func (conn *Conn) SetUsedAt(tm time.Time) {
	conn.usedAt.Store(tm)
}
