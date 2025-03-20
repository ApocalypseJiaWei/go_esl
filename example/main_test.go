package example

import (
	"fmt"
	"github.com/ApocalypseJiaWei/go_esl/fs_esl"
	"github.com/ApocalypseJiaWei/go_esl/model"

	"testing"
	"time"
)

func TestEvent(t *testing.T) {
	// 初始化配置
	config := &fs_esl.PoolConfig{
		MaxOpen:     100,
		MaxIdle:     50,
		Host:        "127.0.0.1",
		Port:        8021,
		Password:    "ClueCon",
		IdleTimeout: 5 * time.Minute,
	}

	// 创建Client
	client := fs_esl.NewClient(config, fs_esl.WithCommandTimeout(15*time.Second))

	// 注册事件监听
	client.RegisterEventHandler("CHANNEL_CREATE", func(e model.Event) {
		fmt.Printf("New call created: %v\n", e.Headers["Caller-ID"])
	})

	// 执行API命令
	resp, _ := client.SendAPICommand("status")
	fmt.Println("Status:", resp)

	// 发起呼叫
	jobUUID, _ := client.OriginateCall("1001", "1002")
	fmt.Println("Job UUID:", jobUUID)
}
