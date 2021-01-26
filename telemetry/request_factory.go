package telemetry

import (
	"errors"
	"net/http"
)

type DataPoint struct {}

type RequestFactory interface {
	BuildRequest([]DataPoint, ...ClientOption) http.Request
}

type requestFactory struct {
	insertKey string
	noDefaultKey bool
	host string
	port uint
}

func (f requestFactory) BuildRequest(points []DataPoint, option ...ClientOption) http.Request {
	panic("implement me")
}

type ClientOption func(o *requestFactory)

func NewRequestFactory(options ...ClientOption) (RequestFactory, error) {
	f := &requestFactory{}
	for _, opt := range options {
		opt(f)
	}

	if f.insertKey == "" && !f.noDefaultKey {
		return nil, errors.New("insert key option must be specified! (one of WithInsertKey or WithNoDefaultKey)")
	}

	return f, nil
}

func WithInsertKey(insertKey string) ClientOption {
	return func(o *requestFactory) {
		o.insertKey = insertKey
	}
}

func WithNoDefaultKey() ClientOption {
	return func(o *requestFactory) {
		o.noDefaultKey = true
	}
}

func WithHost(host string) ClientOption {
	return func(o *requestFactory) {
		o.host = host
	}
}

func WithPort(port uint) ClientOption {
	return func(o *requestFactory) {
		o.port = port
	}
}