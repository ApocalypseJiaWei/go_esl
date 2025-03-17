// pool/pool.go
package pool

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

var (
	ErrPoolClosed = errors.New("connection pool is closed")
	ErrTimeout    = errors.New("get connection timeout")
	ErrInvalid    = errors.New("invalid connection")
)

type connectionPoolImpl struct {
	config    *PoolConfig
	factory   func() (*Connection, error)
	pool      chan *Connection
	openCount int32
	waitCount int64
	waitTime  time.Duration
	closed    bool
	mu        sync.Mutex
	cond      *sync.Cond
}

// NewConnectionPool 创建新的连接池
func NewConnectionPool(config *PoolConfig) ConnectionPool {
	p := &connectionPoolImpl{
		config: config,
		pool:   make(chan *Connection, config.MaxOpen),
		factory: func() (*Connection, error) {
			return newConnection(config)
		},
		closed: false,
	}
	p.cond = sync.NewCond(&p.mu)

	// 初始化空闲连接
	p.initIdleConnections()

	go p.healthCheck()
	go p.evictor()
	return p
}

func (p *connectionPoolImpl) initIdleConnections() {
	for i := 0; i < p.config.MaxIdle; i++ {
		conn, err := p.factory()
		if err != nil {
			continue
		}
		p.pool <- conn
		atomic.AddInt32(&p.openCount, 1)
	}
}

// Get 获取连接
func (p *connectionPoolImpl) Get(ctx context.Context) (*Connection, error) {
	if p.closed {
		return nil, ErrPoolClosed
	}

	start := time.Now()
	atomic.AddInt64(&p.waitCount, 1)
	defer func() {
		atomic.AddInt64(&p.waitCount, -1)
		p.waitTime += time.Since(start)
	}()

	select {
	case conn := <-p.pool:
		if conn.IsValid() {
			return conn, nil
		}
		conn.Close()
		atomic.AddInt32(&p.openCount, -1)
		return p.createConnection(ctx)
	default:
		if atomic.LoadInt32(&p.openCount) < int32(p.config.MaxOpen) {
			return p.createConnection(ctx)
		}
	}

	// 等待可用连接
	for {
		select {
		case <-ctx.Done():
			return nil, ErrTimeout
		default:
			p.mu.Lock()
			p.cond.Wait()
			p.mu.Unlock()

			select {
			case conn := <-p.pool:
				if conn.IsValid() {
					return conn, nil
				}
				conn.Close()
				atomic.AddInt32(&p.openCount, -1)
				return p.createConnection(ctx)
			default:
			}
		}
	}
}

// pool/pool.go
func (p *connectionPoolImpl) createConnection(ctx context.Context) (*Connection, error) {
	if atomic.LoadInt32(&p.openCount) >= int32(p.config.MaxOpen) {
		return nil, fmt.Errorf("reach max open connections: %d", p.config.MaxOpen)
	}

	conn, err := p.factory()
	if err != nil {
		return nil, fmt.Errorf("create connection failed: %w", err)
	}

	atomic.AddInt32(&p.openCount, 1)
	return conn, nil
}

// Put 归还连接
func (p *connectionPoolImpl) Put(conn *Connection) {
	if p.closed || !conn.IsValid() {
		conn.Close()
		atomic.AddInt32(&p.openCount, -1)
		return
	}

	conn.lastUsed = time.Now()

	select {
	case p.pool <- conn:
		p.cond.Signal()
	default:
		conn.Close()
		atomic.AddInt32(&p.openCount, -1)
	}
}

// Close 关闭连接池
func (p *connectionPoolImpl) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return
	}
	p.closed = true
	close(p.pool)

	for conn := range p.pool {
		conn.Close()
	}
}

// Stats 获取统计信息
func (p *connectionPoolImpl) Stats() PoolStats {
	return PoolStats{
		OpenConnections: int(atomic.LoadInt32(&p.openCount)),
		IdleConnections: len(p.pool),
		MaxOpen:         p.config.MaxOpen,
		MaxIdle:         p.config.MaxIdle,
		WaitCount:       atomic.LoadInt64(&p.waitCount),
		WaitDuration:    p.waitTime,
	}
}

// healthCheck 健康检查
func (p *connectionPoolImpl) healthCheck() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		if p.closed {
			return
		}

		p.mu.Lock()
		defer p.mu.Unlock()

		for i := 0; i < len(p.pool); i++ {
			select {
			case conn := <-p.pool:
				if !conn.IsValid() || time.Since(conn.lastUsed) > p.config.IdleTimeout {
					conn.Close()
					atomic.AddInt32(&p.openCount, -1)
				} else {
					p.pool <- conn
				}
			default:
				break
			}
		}
	}
}

// evictor 淘汰过期连接
func (p *connectionPoolImpl) evictor() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		if p.closed {
			return
		}

		p.mu.Lock()
		defer p.mu.Unlock()

		now := time.Now()
		n := len(p.pool)
		for i := 0; i < n; i++ {
			select {
			case conn := <-p.pool:
				if now.Sub(conn.lastUsed) > p.config.IdleTimeout {
					conn.Close()
					atomic.AddInt32(&p.openCount, -1)
				} else {
					p.pool <- conn
				}
			default:
				break
			}
		}
	}
}
