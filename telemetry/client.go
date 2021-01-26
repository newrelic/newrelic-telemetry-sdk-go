package telemetry

import "net/http"

type DataPoint struct {}

type Client interface {
	Send(DataPoint, ...ClientOption)
	SendBatch([]DataPoint, ...ClientOption)
}

type client struct {
	client http.Client
	insertKey string
	noDefaultKey bool
	host string
	port uint
}

func (c client) Send(point DataPoint, option ...ClientOption) {
	panic("implement me")
}

func (c client) SendBatch(points []DataPoint, option ...ClientOption) {
	panic("implement me")
}

type ClientOption func(o *client)

func NewClient(options ...ClientOption) Client {
	c := &client{
		client: http.Client{},
	}
	for _, opt := range options {
		opt(c)
	}

	if c.insertKey == "" && !c.noDefaultKey {
		panic("Insert key option must be specified! (one of WithInsertKey or WithNoDefaultKey)")
	}

	return c
}

func WithInsertKey(insertKey string) ClientOption {
	return func(o *client) {
		o.insertKey = insertKey
	}
}

func WithNoDefaultKey() ClientOption {
	return func(o *client) {
		o.noDefaultKey = true
	}
}

func WithHost(host string) ClientOption {
	return func(o *client) {
		o.host = host
	}
}

func WithPort(port uint) ClientOption {
	return func(o *client) {
		o.port = port
	}
}