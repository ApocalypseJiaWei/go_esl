// pool/config.go
package pool

import (
	"context"
	"time"
)

// PoolConfig 连接池配置结构体
type PoolConfig struct {
	MaxOpen     int           // 最大打开连接数
	MaxIdle     int           // 最大空闲连接数
	Timeout     time.Duration // 连接获取超时时间
	Host        string        // FreeSWITCH 地址
	Port        int           // FreeSWITCH 端口
	Password    string        // 认证密码
	IdleTimeout time.Duration // 连接最大空闲时间
}

// PoolStats 连接池统计信息
type PoolStats struct {
	OpenConnections int           // 当前打开连接数
	IdleConnections int           // 空闲连接数
	MaxOpen         int           // 最大打开连接数配置
	MaxIdle         int           // 最大空闲连接数配置
	WaitCount       int64         // 等待连接总数
	WaitDuration    time.Duration // 总等待时间
}

// ConnectionPool 连接池接口定义
type ConnectionPool interface {
	Get(ctx context.Context) (*Connection, error)
	Put(conn *Connection)
	Close()
	Stats() PoolStats
}
