// event/dispatcher.go
package event

import (
	"github.com/ApocalypseJiaWei/go_esl/model"
	"github.com/panjf2000/ants/v2"
	"sync"
)

type EventListener func(event model.Event)

type EventDispatcher struct {
	pool      *ants.Pool
	listeners map[string][]EventListener
	mu        sync.RWMutex
}

func NewDispatcher(poolSize int) *EventDispatcher {
	pool, _ := ants.NewPool(poolSize)
	return &EventDispatcher{
		pool:      pool,
		listeners: make(map[string][]EventListener),
	}
}

func (ed *EventDispatcher) Register(eventName string, listener EventListener) {
	ed.mu.Lock()
	defer ed.mu.Unlock()
	ed.listeners[eventName] = append(ed.listeners[eventName], listener)
}

func (ed *EventDispatcher) Dispatch(event model.Event) {
	// 获取事件名称
	if name, exists := event.Headers["Event-Name"]; exists {
		event.Name = name
	}

	ed.pool.Submit(func() {
		ed.mu.RLock()
		defer ed.mu.RUnlock()

		// 处理全局监听器
		if listeners, ok := ed.listeners["*"]; ok {
			for _, listener := range listeners {
				listener(event)
			}
		}

		// 处理特定事件监听器
		if listeners, ok := ed.listeners[event.Name]; ok {
			for _, listener := range listeners {
				listener(event)
			}
		}
	})
}
