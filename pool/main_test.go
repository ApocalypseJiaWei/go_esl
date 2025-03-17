package pool

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestEvent(t *testing.T) {
	// 初始化连接池
	pool := NewConnectionPool(&PoolConfig{
		MaxOpen:     100,
		MaxIdle:     50,
		Host:        "127.0.0.1",
		Port:        8021,
		Password:    "ClueCon",
		IdleTimeout: 5 * time.Minute,
	})

	// 获取连接
	conn, _ := pool.Get(context.Background())
	defer pool.Put(conn)

	// 执行API命令
	_ = conn.WriteMessage([]byte("api status\n\n"))
	resp, _ := conn.ReadMessage()
	fmt.Printf("Status: %s\n", resp)

	// 执行BGAPI命令
	bgapiCmd := `bgapi originate {origination_call_id_name=test}user/1000 &echo`
	_ = conn.WriteMessage([]byte(bgapiCmd))
	jobResp, _ := conn.ReadMessage()
	fmt.Printf("Job UUID: %s\n", parseJobUUID(jobResp))
}

func parseJobUUID(resp []byte) string {
	// 实现响应解析逻辑
	return ""
}
