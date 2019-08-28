package pool

import (
	"errors"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

var (
	//ErrClosed 连接池已经关闭Error
	ErrClosed = errors.New("pool is closed")
	//ErrPoolTimeout 连接池超时Error
	ErrPoolTimeout = errors.New("connection pool timeout")
	// ErrOptionsPoolSize is invalid options's MinIdleConns or PoolSize
	ErrOptionsPoolSize = errors.New("invalid options settings: MinIdleConns, PoolSize")
	// ErrOptionsDialer is invalid Dialer func settings
	ErrOptionsDialer = errors.New("invalid Dialer func settings")
	// ErrOptionsOnClose is invalid OnClose func settings
	ErrOptionsOnClose = errors.New("invalid OnClose func settings")
	// timers is ...
	timers = sync.Pool{
		New: func() interface{} {
			t := time.NewTimer(time.Hour)
			t.Stop()
			return t
		},
	}
)

// Stats contains pool state information and accumulated stats.
type Stats struct {
	Hits     uint32 // number of times free connection was found in the pool
	Misses   uint32 // number of times free connection was NOT found in the pool
	Timeouts uint32 // number of times a wait timeout occurred

	TotalConns uint32 // number of total connections in the pool
	IdleConns  uint32 // number of idle connections in the pool
	StaleConns uint32 // number of stale connections removed from the pool
}

// Pooler is ...
type Pooler interface {
	NewConn() (interface{}, error)
	CloseConn(interface{}) error

	Get() (interface{}, error)
	Put(interface{})
	Remove(interface{}, error)

	Len() int
	IdleLen() int
	Stats() *Stats

	Close() error
}

// Options 连接池相关配置
type Options struct {
	// Dialer creates new network connection and has priority over
	Dialer func() (interface{}, error)
	// Close closes the connection, releasing any open resources.
	OnClose func(interface{}) error

	// Maximum number of socket connections.
	// Default is 10 connections per every CPU as reported by runtime.NumCPU.
	PoolSize int
	// Minimum number of idle connections which is useful when establishing
	MinIdleConns int
	// Amount of time client waits for connection if all connections
	// are busy before returning an error.
	// Default is 5 second.
	PoolTimeout time.Duration
	// Amount of time after which client closes idle connections.
	// Should be less than server's timeout.
	// Default is 5 minutes. -1 disables idle timeout check.
	IdleTimeout time.Duration
	// Frequency of idle checks made by idle connections reaper.
	// Default is 1 minute. -1 disables idle connections reaper,
	// but idle connections are still discarded by the client
	// if IdleTimeout is set.
	IdleCheckFrequency time.Duration
}

// ConnPool is conn pool
type ConnPool struct {
	opt *Options

	dialErrorsNum uint32 // atomic

	lastDialErrorMu sync.RWMutex
	lastDialError   error

	queue chan struct{}

	connsMu      sync.Mutex
	conns        []*Conn
	idleConns    []*Conn
	poolSize     int
	idleConnsLen int

	stats Stats

	_closed uint32 // atomic
}

// NewConnPool is new a ConnPool
func NewConnPool(opt *Options) (*ConnPool, error) {
	if opt.MinIdleConns < 0 || opt.PoolSize < 0 || opt.MinIdleConns > opt.PoolSize {
		return nil, ErrOptionsPoolSize
	}
	if opt.Dialer == nil {
		return nil, ErrOptionsDialer
	}
	if opt.OnClose == nil {
		return nil, ErrOptionsOnClose
	}
	if opt.PoolSize == 0 {
		opt.PoolSize = 10 * runtime.NumCPU()
	}
	if opt.PoolTimeout == 0 {
		opt.PoolTimeout = 5 * time.Second
	}
	if opt.IdleTimeout == 0 {
		opt.IdleTimeout = 5 * time.Minute
	}
	if opt.IdleCheckFrequency == 0 {
		opt.IdleCheckFrequency = time.Minute
	}

	p := &ConnPool{
		opt: opt,

		queue:     make(chan struct{}, opt.PoolSize),
		conns:     make([]*Conn, 0, opt.PoolSize),
		idleConns: make([]*Conn, 0, opt.PoolSize),
	}

	for i := 0; i < opt.MinIdleConns; i++ {
		p.checkMinIdleConns()
	}

	if opt.IdleTimeout > 0 && opt.IdleCheckFrequency > 0 {
		go p.reaper(opt.IdleCheckFrequency)
	}

	return p, nil
}

func (p *ConnPool) checkMinIdleConns() {
	if p.opt.MinIdleConns == 0 {
		return
	}
	if p.poolSize < p.opt.PoolSize && p.idleConnsLen < p.opt.MinIdleConns {
		p.poolSize++
		p.idleConnsLen++
		go p.addIdleConn()
	}
}

func (p *ConnPool) addIdleConn() {
	conn, err := p.newConn()
	if err != nil {
		return
	}

	p.connsMu.Lock()
	p.conns = append(p.conns, conn)
	p.idleConns = append(p.idleConns, conn)
	p.connsMu.Unlock()
}

// _NewConn is new a conn, append to p.conns, but not append to p.idleConns
func (p *ConnPool) _NewConn() (*Conn, error) {
	conn, err := p.newConn()
	if err != nil {
		return nil, err
	}

	p.connsMu.Lock()
	p.conns = append(p.conns, conn)

	if p.poolSize < p.opt.PoolSize {
		p.poolSize++
	}

	p.connsMu.Unlock()
	return conn, nil
}

func (p *ConnPool) newConn() (*Conn, error) {
	if p.closed() {
		return nil, ErrClosed
	}

	if atomic.LoadUint32(&p.dialErrorsNum) >= uint32(p.opt.PoolSize) {
		return nil, p.getLastDialError()
	}

	cn, err := p.opt.Dialer()
	if err != nil {
		p.setLastDialError(err)
		if atomic.AddUint32(&p.dialErrorsNum, 1) == uint32(p.opt.PoolSize) {
			go p.tryDial()
		}
		return nil, err
	}

	conn := NewConn(cn)
	return conn, nil
}

func (p *ConnPool) tryDial() {
	for {
		if p.closed() {
			return
		}

		conn, err := p.opt.Dialer()
		if err != nil {
			p.setLastDialError(err)
			time.Sleep(time.Second)
			continue
		}

		atomic.StoreUint32(&p.dialErrorsNum, 0)
		p.opt.OnClose(conn)
		return
	}
}
func (p *ConnPool) setLastDialError(err error) {
	p.lastDialErrorMu.Lock()
	p.lastDialError = err
	p.lastDialErrorMu.Unlock()
}

func (p *ConnPool) getLastDialError() error {
	p.lastDialErrorMu.RLock()
	err := p.lastDialError
	p.lastDialErrorMu.RUnlock()
	return err
}

// Get returns existed connection from the pool or creates a new one.
func (p *ConnPool) Get() (interface{}, error) {
	if p.closed() {
		return nil, ErrClosed
	}

	err := p.waitTurn()
	if err != nil {
		return nil, err
	}

	for {
		p.connsMu.Lock()
		conn := p.popIdle()
		p.connsMu.Unlock()

		if conn == nil {
			break
		}

		if p.isStaleConn(conn) {
			_ = p.CloseConn(conn)
			continue
		}

		atomic.AddUint32(&p.stats.Hits, 1)
		return conn.cn, nil
	}

	atomic.AddUint32(&p.stats.Misses, 1)

	newconn, err := p._NewConn()
	if err != nil {
		p.freeTurn()
		return nil, err
	}

	return newconn.cn, nil
}

func (p *ConnPool) getTurn() {
	p.queue <- struct{}{}
}

func (p *ConnPool) waitTurn() error {
	select {
	case p.queue <- struct{}{}:
		return nil
	default:
		timer := timers.Get().(*time.Timer)
		timer.Reset(p.opt.PoolTimeout)

		select {
		case p.queue <- struct{}{}:
			if !timer.Stop() {
				<-timer.C
			}
			timers.Put(timer)
			return nil
		case <-timer.C:
			timers.Put(timer)
			atomic.AddUint32(&p.stats.Timeouts, 1)
			return ErrPoolTimeout
		}
	}
}

func (p *ConnPool) freeTurn() {
	<-p.queue
}

func (p *ConnPool) popIdle() *Conn {
	if len(p.idleConns) == 0 {
		return nil
	}

	idx := len(p.idleConns) - 1
	conn := p.idleConns[idx]
	p.idleConns = p.idleConns[:idx]
	p.idleConnsLen--
	p.checkMinIdleConns()
	return conn
}

// Put a cn to ConnPool
func (p *ConnPool) Put(cn interface{}) {
	conn := NewConn(cn)
	p.connsMu.Lock()
	p.idleConns = append(p.idleConns, conn)
	p.idleConnsLen++
	p.connsMu.Unlock()
	p.freeTurn()
}

// Remove a conn
func (p *ConnPool) Remove(conn *Conn, reason error) {
	p.removeConn(conn)
	p.freeTurn()
	_ = p.closeConn(conn)
}

// CloseConn is close a conn
func (p *ConnPool) CloseConn(conn *Conn) error {
	p.removeConn(conn)
	return p.closeConn(conn)
}

func (p *ConnPool) removeConn(conn *Conn) {
	p.connsMu.Lock()
	for i, c := range p.conns {
		if c.cn == conn.cn {
			p.conns = append(p.conns[:i], p.conns[i+1:]...)
			p.poolSize--
			p.checkMinIdleConns()
			break
		}
	}
	p.connsMu.Unlock()
}

func (p *ConnPool) closeConn(conn *Conn) error {
	return p.opt.OnClose(conn.cn)
}

// Len returns total number of connections.
func (p *ConnPool) Len() int {
	p.connsMu.Lock()
	n := len(p.conns)
	p.connsMu.Unlock()
	return n
}

// IdleLen returns number of idle connections.
func (p *ConnPool) IdleLen() int {
	p.connsMu.Lock()
	n := p.idleConnsLen
	p.connsMu.Unlock()
	return n
}

// Stats return stats info of ConnPool
func (p *ConnPool) Stats() *Stats {
	idleLen := p.IdleLen()
	return &Stats{
		Hits:     atomic.LoadUint32(&p.stats.Hits),
		Misses:   atomic.LoadUint32(&p.stats.Misses),
		Timeouts: atomic.LoadUint32(&p.stats.Timeouts),

		TotalConns: uint32(p.Len()),
		IdleConns:  uint32(idleLen),
		StaleConns: atomic.LoadUint32(&p.stats.StaleConns),
	}
}

func (p *ConnPool) closed() bool {
	return atomic.LoadUint32(&p._closed) == 1
}

// Close is close ConnPool
func (p *ConnPool) Close() error {
	if !atomic.CompareAndSwapUint32(&p._closed, 0, 1) {
		return ErrClosed
	}

	var firstErr error
	p.connsMu.Lock()
	for _, conn := range p.conns {
		if err := p.closeConn(conn); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	p.conns = nil
	p.poolSize = 0
	p.idleConns = nil
	p.idleConnsLen = 0
	p.connsMu.Unlock()

	return firstErr
}

func (p *ConnPool) reapStaleConn() *Conn {
	if len(p.idleConns) == 0 {
		return nil
	}

	conn := p.idleConns[0]
	if !p.isStaleConn(conn) {
		return nil
	}

	p.idleConns = append(p.idleConns[:0], p.idleConns[1:]...)
	p.idleConnsLen--

	return conn
}

// ReapStaleConns returns number of clear conn
func (p *ConnPool) ReapStaleConns() (int, error) {
	var n int
	for {
		p.getTurn()

		p.connsMu.Lock()
		conn := p.reapStaleConn()
		p.connsMu.Unlock()

		if conn != nil {
			p.removeConn(conn)
		}

		p.freeTurn()

		if conn != nil {
			p.closeConn(conn)
			n++
		} else {
			break
		}
	}
	return n, nil
}

func (p *ConnPool) reaper(frequency time.Duration) {
	ticker := time.NewTicker(frequency)
	defer ticker.Stop()

	for range ticker.C {
		if p.closed() {
			break
		}
		n, err := p.ReapStaleConns()
		if err != nil {
			continue
		}
		atomic.AddUint32(&p.stats.StaleConns, uint32(n))
	}
}

func (p *ConnPool) isStaleConn(conn *Conn) bool {
	if p.opt.IdleTimeout == 0 {
		return false
	}

	now := time.Now()
	if p.opt.IdleTimeout > 0 && now.Sub(conn.UsedAt()) >= p.opt.IdleTimeout {
		return true
	}

	return false
}
