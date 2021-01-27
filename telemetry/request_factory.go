package telemetry

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
)

type PayloadEntry interface {
	Type() string
	ToPayload() interface{}
}

type RequestFactory interface {
	BuildRequest([]PayloadEntry, ...ClientOption) http.Request
}

type requestFactory struct {
	insertKey    string
	noDefaultKey bool
	host         string
	port         uint
	path         string
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

func (f *requestFactory) BuildRequest(entries []PayloadEntry, options ...ClientOption) http.Request {
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

	var mappedEntries map[string]interface{}

	for _, entry := range entries {
		mappedEntries[entry.Type()] = entry.ToPayload()
	}

	var payload []interface{}
	payload = append(payload, mappedEntries)
	b, _ := json.Marshal(payload)

	// TODO: compress batch bytes
	body := ioutil.NopCloser(bytes.NewReader(b))
	host := configuredFactory.getHost()
	headers := configuredFactory.getHeaders()

	return http.Request{
		Method: "POST",
		URL: &url.URL{
			Scheme: "https",
			Host:   configuredFactory.host,
			Path:   configuredFactory.path,
		},
		Header:        headers,
		Body:          body,
		ContentLength: int64(len(b)),
		Close:         false,
		Host:          host,
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

func NewSpanRequestFactory(options ...ClientOption) (RequestFactory, error) {
	f := &requestFactory{host: "trace-api.newrelic.com", path: "/trace/v1"}
	err := configure(f, options)
	if err != nil {
		return nil, err
	}

	return f, nil
}

func NewMetricRequestFactory(options ...ClientOption) (RequestFactory, error) {
	f := &requestFactory{host: "metric-api.newrelic.com", path: "/metric/v1"}
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
