package telemetry

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/newrelic/newrelic-telemetry-sdk-go/internal"
	"io/ioutil"
	"net/http"
	"net/url"
)
const defaultUserAgent = "NewRelic-Go-TelemetrySDK/" + version

type PayloadEntry interface {
	Type() string
	Bytes() []byte
}

type RequestFactory interface {
	BuildRequest([]PayloadEntry, ...ClientOption) (http.Request, error)
}

type requestFactory struct {
	insertKey    string
	noDefaultKey bool
	host         string
	port         uint
	path         string
	userAgent    string
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

func (f *requestFactory) BuildRequest(entries []PayloadEntry, options ...ClientOption) (http.Request, error) {
	configuredFactory := &requestFactory{
		insertKey:    f.insertKey,
		noDefaultKey: f.noDefaultKey,
		host:         f.host,
		port:         f.port,
		path:         f.path,
		userAgent:    f.userAgent,
	}

	err := configure(configuredFactory, options)

	if err != nil {
		return http.Request{}, errors.New("unable to configure this request based on options passed in")
	}

	buf := &bytes.Buffer{}
	buf.WriteByte('[')
	buf.WriteByte('{')
	w := internal.JSONFieldsWriter{Buf: buf}

	for _, entry := range entries {
		w.RawField(entry.Type(), entry.Bytes())
	}

	buf.WriteByte('}')
	buf.WriteByte(']')

	buf, err = internal.Compress(buf.Bytes())
	if err != nil {
		return http.Request{}, err
	}
	var contentLength = int64(buf.Len())
	body := ioutil.NopCloser(buf)
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
		ContentLength: contentLength,
		Close:         false,
		Host:          host,
	}, nil
}

func (f *requestFactory) getHost() string {
	s := f.host
	if f.port > 0 {
		s = s + fmt.Sprintf(":%d", f.port)
	}
	return s
}

func (f *requestFactory) getHeaders() http.Header {
	return http.Header{
		"Content-Type":     []string{"application/json"},
		"Content-Encoding": []string{"gzip"},
		"Api-Key":          []string{f.insertKey},
		"User-Agent":       []string{f.userAgent},
	}
}

type ClientOption func(o *requestFactory)

func NewSpanRequestFactory(options ...ClientOption) (RequestFactory, error) {
	f := &requestFactory{host: "trace-api.newrelic.com", path: "/trace/v1", userAgent: defaultUserAgent}
	err := configure(f, options)
	if err != nil {
		return nil, err
	}

	return f, nil
}

func NewMetricRequestFactory(options ...ClientOption) (RequestFactory, error) {
	f := &requestFactory{host: "metric-api.newrelic.com", path: "/metric/v1", userAgent: defaultUserAgent}
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

func WithUserAgent(userAgent string) ClientOption {
	return func(o *requestFactory) {
		o.userAgent = defaultUserAgent + " " + userAgent
	}
}
