package telemetry

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
)

type DataPoint interface{}

type Batch interface {
	Type() string
	Bytes() *bytes.Buffer
	DataPoints() []DataPoint
}

type RequestFactory interface {
	BuildRequest(Batch, ...ClientOption) http.Request
}

type requestFactory struct {
	insertKey    string
	noDefaultKey bool
	host         string
	port         uint
}

func configure(factory *requestFactory, options []ClientOption) {
	for _, option := range options {
		option(factory)
	}
}

func (f *requestFactory) BuildRequest(batch Batch, options ...ClientOption) http.Request {
	configuredFactory := requestFactory{
		insertKey:    f.insertKey,
		noDefaultKey: f.noDefaultKey,
		host:         f.host,
		port:         f.port,
	}

	configure(&configuredFactory, options)

	bytes := batch.Bytes()
	body := ioutil.NopCloser(bytes)
	host := configuredFactory.getHost()
	headers := configuredFactory.getHeaders()
	path := configuredFactory.getPath(batch.Type())

	return http.Request{
		Method: "POST",
		URL: &url.URL{
			Scheme: "https",
			Host:   configuredFactory.host,
			Path:   path,
		},
		Header:        headers,
		Body:          body,
		ContentLength: int64(bytes.Len()),
		Close:         false,
		Host:          host,
	}
}

func (f *requestFactory) getPath(t string) string {
	switch t {
	case "spans":
		return "/trace/v1"
	case "metrics":
		return "/metric/v1"
	case "logs":
		return "/log/v1"
	default:
		return ""
	}
}

func (f *requestFactory) getHost() string {
	s := f.host
	if f.port > 0 {
		s = s + fmt.Sprintf(":%d", f.port)
	}
	return s
}

func (f *requestFactory) getHeaders() http.Header {
	return http.Header{}
}

type ClientOption func(o *requestFactory)

func NewRequestFactory(options ...ClientOption) (RequestFactory, error) {
	f := &requestFactory{}
	configure(f, options)

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
