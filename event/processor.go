package event

// Event 定义通用事件接口
type Event interface {
	Name() string
	Headers() map[string]string
	Body() string
	Raw() string
	Parse(raw string) error
}

// Processor 事件处理器接口
type Processor interface {
	RegisterHandler(eventType string, handler func(Event))
	Process(raw string) error
}

// 实现层 processor.go
type eventProcessor struct {
	handlers   map[string][]func(Event)
	decoder    Decoder    // 解码器接口
	dispatcher Dispatcher // 分发器接口
}

type Decoder interface {
	Decode(raw string) (Event, error)
}

type Dispatcher interface {
	Dispatch(e Event)
}

func NewProcessor(dec Decoder, disp Dispatcher) Processor {
	return &eventProcessor{
		handlers:   make(map[string][]func(Event)),
		decoder:    dec,
		dispatcher: disp,
	}
}

func (p *eventProcessor) RegisterHandler(eventType string, handler func(Event)) {
	p.handlers[eventType] = append(p.handlers[eventType], handler)
}

func (p *eventProcessor) Process(raw string) error {
	event, err := p.decoder.Decode(raw)
	if err != nil {
		return err
	}

	// 通用处理
	p.dispatcher.Dispatch(event)

	// 特定处理
	if handlers, ok := p.handlers[event.Name()]; ok {
		for _, h := range handlers {
			go h(event) // 使用协程处理
		}
	}
	return nil
}
