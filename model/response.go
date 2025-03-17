package model

// model/response.go

type Response struct {
	Success bool
	Body    string
	Headers map[string]string
}

type PoolStats struct {
	OpenConnections int
	IdleConnections int
	MaxOpen         int
	MaxIdle         int
}

type Event struct {
	Name    string
	Headers map[string]string
	Body    string
	Raw     string
}
