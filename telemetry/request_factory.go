package telemetry

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
)

type Batch interface {
	Type() string
	Bytes() []byte
	writeJSON(*bytes.Buffer)
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

func configure(f *requestFactory, options []ClientOption) error {
	for _, option := range options {
		option(f)
	}

	if f.insertKey == "" && !f.noDefaultKey {
		return errors.New("insert key option must be specified! (one of WithInsertKey or WithNoDefaultKey)")
	}
	return nil

}

func (f *requestFactory) BuildRequest(batch Batch, options ...ClientOption) http.Request {
	configuredFactory := &requestFactory{
		insertKey:    f.insertKey,
		noDefaultKey: f.noDefaultKey,
		host:         f.host,
		port:         f.port,
	}

	err := configure(configuredFactory, options)

	// If unable to configure, just use the already configured request factory for the request
	if err != nil {
		configuredFactory = f
	}

	b := batch.Bytes()
	// TODO: compress batch bytes
	body := ioutil.NopCloser(bytes.NewReader(b))
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
		ContentLength: int64(len(b)),
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
	err := configure(f, options)
	if err != nil {
		return nil, err
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
