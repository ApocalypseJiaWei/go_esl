// pool/connection.go
package pool

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Connection 封装ESL连接
type Connection struct {
	conn     net.Conn
	lastUsed time.Time
	mu       sync.Mutex
	alive    bool
	config   *PoolConfig
}

//func newConnection(config *PoolConfig) (*Connection, error) {
//	address := fmt.Sprintf("%s:%d", config.Host, config.Port)
//	conn, err := net.DialTimeout("tcp", address, 5*time.Second)
//	if err != nil {
//		return nil, fmt.Errorf("connection failed: %w", err)
//	}
//
//	// 设置读写超时
//	deadline := time.Now().Add(5 * time.Second)
//	_ = conn.SetDeadline(deadline)
//
//	// 执行认证
//	_, err = conn.Write([]byte(fmt.Sprintf("auth %s\n\n", config.Password)))
//	if err != nil {
//		conn.Close()
//		return nil, fmt.Errorf("authentication failed: %w", err)
//	}
//
//	// 读取认证响应
//	buf := make([]byte, 1024)
//	n, err := conn.Read(buf)
//	if err != nil || !isValidAuthResponse(buf[:n]) {
//		conn.Close()
//		return nil, fmt.Errorf("authentication failed")
//	}
//
//	// 重置超时设置
//	_ = conn.SetDeadline(time.Time{})
//
//	return &Connection{
//		conn:     conn,
//		lastUsed: time.Now(),
//		alive:    true,
//		config:   config,
//	}, nil
//}

// IsValid 检查连接是否有效
func (c *Connection) IsValid() bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.alive {
		return false
	}

	// 心跳检测
	if err := c.conn.SetWriteDeadline(time.Now().Add(3 * time.Second)); err != nil {
		c.alive = false
		return false
	}

	if _, err := c.conn.Write([]byte("ping\n\n")); err != nil {
		c.alive = false
		return false
	}

	// 读取响应
	buf := make([]byte, 128)
	if err := c.conn.SetReadDeadline(time.Now().Add(3 * time.Second)); err != nil {
		c.alive = false
		return false
	}

	if _, err := c.conn.Read(buf); err != nil {
		c.alive = false
		return false
	}

	return true
}

// Close 关闭连接
func (c *Connection) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn != nil {
		_ = c.conn.Close()
	}
	c.alive = false
}

func isValidAuthResponse(data []byte) bool {
	return bytes.Contains(data, []byte("+OK accepted"))
}

// ReadMessage 完整实现ESL消息读取
func (c *Connection) ReadMessage() ([]byte, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// 设置读取超时
	_ = c.conn.SetReadDeadline(time.Now().Add(30 * time.Second))

	reader := bufio.NewReader(c.conn)
	var buffer bytes.Buffer
	var contentLength int

	// 读取头部
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			c.alive = false
			return nil, fmt.Errorf("read header failed: %w", err)
		}

		line = strings.TrimSpace(line)
		buffer.WriteString(line + "\n")

		// 检测头部结束
		if line == "" {
			break
		}

		// 解析Content-Length
		if strings.HasPrefix(line, "Content-Length:") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				contentLength, _ = strconv.Atoi(strings.TrimSpace(parts[1]))
			}
		}
	}

	// 读取消息体
	if contentLength > 0 {
		body := make([]byte, contentLength)
		_, err := io.ReadFull(reader, body)
		if err != nil {
			c.alive = false
			return nil, fmt.Errorf("read body failed: %w", err)
		}
		buffer.Write(body)
	}

	return buffer.Bytes(), nil
}

// WriteMessage 完整实现ESL命令发送
func (c *Connection) WriteMessage(cmd []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// 自动追加双换行符
	if !bytes.HasSuffix(cmd, []byte("\n\n")) {
		cmd = append(cmd, []byte("\n\n")...)
	}

	// 设置写超时
	_ = c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))

	totalSent := 0
	for totalSent < len(cmd) {
		n, err := c.conn.Write(cmd[totalSent:])
		if err != nil {
			c.alive = false
			return fmt.Errorf("write failed: %w", err)
		}
		totalSent += n
	}

	return nil
}

// 增强版连接工厂方法
func newConnection(config *PoolConfig) (*Connection, error) {
	conn, err := net.DialTimeout("tcp",
		fmt.Sprintf("%s:%d", config.Host, config.Port),
		5*time.Second,
	)
	if err != nil {
		return nil, fmt.Errorf("dial failed: %w", err)
	}

	eslConn := &Connection{
		conn:     conn,
		lastUsed: time.Now(),
		alive:    true,
		config:   config,
	}

	// 执行认证
	authCmd := fmt.Sprintf("auth %s\n\n", config.Password)
	if err := eslConn.WriteMessage([]byte(authCmd)); err != nil {
		conn.Close()
		return nil, fmt.Errorf("auth failed: %w", err)
	}

	// 验证认证响应
	resp, err := eslConn.ReadMessage()
	if err != nil || !bytes.Contains(resp, []byte("+OK accepted")) {
		conn.Close()
		return nil, errors.New("authentication failed")
	}

	return eslConn, nil
}
