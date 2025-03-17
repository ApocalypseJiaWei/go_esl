package command

import (
	"fmt"
	"strings"
	"time"
)

type CommandType int

const (
	API CommandType = iota
	BGAPI
	MSG
	EVENT
)

type Command struct {
	cmdType CommandType
	command string
	params  map[string]string
	headers map[string]string
	async   bool
	timeout time.Duration
	//	eventChan  chan<- Event // 事件通道
	resultChan chan<- Result
}

type Result struct {
	JobUUID string
	Body    string
	Error   error
}

func New(cmdType CommandType, command string) *Command {
	return &Command{
		cmdType: cmdType,
		command: command,
		params:  make(map[string]string),
		headers: make(map[string]string),
	}
}

func (c *Command) WithParam(key, value string) *Command {
	c.params[key] = value
	return c
}

func (c *Command) WithHeader(key, value string) *Command {
	c.headers[key] = value
	return c
}

func (c *Command) Async() *Command {
	c.async = true
	return c
}

func (c *Command) Timeout(d time.Duration) *Command {
	c.timeout = d
	return c
}

//func (c *Command) OnEvent(ch chan<- Event) *Command {
//	c.eventChan = ch
//	return c
//}

func (c *Command) OnResult(ch chan<- Result) *Command {
	c.resultChan = ch
	return c
}

func (c *Command) Build() string {
	var buf strings.Builder

	switch c.cmdType {
	case BGAPI:
		buf.WriteString("bgapi ")
	case MSG:
		buf.WriteString("sendmsg ")
	case EVENT:
		buf.WriteString("event ")
	}

	buf.WriteString(c.command + "\n")

	for k, v := range c.headers {
		buf.WriteString(fmt.Sprintf("%s: %s\n", k, v))
	}

	if len(c.params) > 0 {
		buf.WriteString("\n")
		for k, v := range c.params {
			buf.WriteString(fmt.Sprintf("%s=%s\n", k, v))
		}
	}

	buf.WriteString("\n")
	return buf.String()
}
