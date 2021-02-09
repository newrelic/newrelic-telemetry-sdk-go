package telemetry

import (
	"bytes"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/newrelic/newrelic-telemetry-sdk-go/internal"
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
	insertKey     string
	noDefaultKey  bool
	host          string
	path          string
	userAgent     string
	useRawPayload bool
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
		insertKey:     f.insertKey,
		noDefaultKey:  f.noDefaultKey,
		host:          f.host,
		path:          f.path,
		userAgent:     f.userAgent,
		useRawPayload: f.useRawPayload,
	}

	err := configure(configuredFactory, options)

	if err != nil {
		return http.Request{}, errors.New("unable to configure this request based on options passed in")
	}

	buf := &bytes.Buffer{}
	buf.WriteByte('[')
	if (!f.useRawPayload) {
		buf.WriteByte('{')
	}
	w := internal.JSONFieldsWriter{Buf: buf}

	for idx, entry := range entries {
		if (!f.useRawPayload) {
			w.RawField(entry.Type(), entry.Bytes())
		} else {
			if (idx > 0) {
				buf.WriteByte(',')
			}
			buf.Write(entry.Bytes())
		}
	}

	if (!f.useRawPayload) {
		buf.WriteByte('}')
	}
	buf.WriteByte(']')

	buf, err = internal.Compress(buf.Bytes())
	if err != nil {
		return http.Request{}, err
	}

	getBody := func() (io.ReadCloser, error) {
		return ioutil.NopCloser(bytes.NewBuffer(buf.Bytes())), nil
	}

	var contentLength = int64(buf.Len())
	body, _ := getBody()
	host := configuredFactory.host
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
		GetBody:       getBody,
		ContentLength: contentLength,
		Close:         false,
		Host:          host,
	}, nil
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
	f := &requestFactory{host: "trace-api.newrelic.com", path: "/trace/v1", userAgent: defaultUserAgent, useRawPayload: false}
	err := configure(f, options)
	if err != nil {
		return nil, err
	}

	return f, nil
}

func NewMetricRequestFactory(options ...ClientOption) (RequestFactory, error) {
	f := &requestFactory{host: "metric-api.newrelic.com", path: "/metric/v1", userAgent: defaultUserAgent, useRawPayload: false}
	err := configure(f, options)
	if err != nil {
		return nil, err
	}

	return f, nil
}

func NewEventRequestFactory(options ...ClientOption) (RequestFactory, error) {
	f := &requestFactory{host: "insights-collector.newrelic.com", path: "/v1/accounts/events", userAgent: defaultUserAgent, useRawPayload: true}
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

func WithUserAgent(userAgent string) ClientOption {
	return func(o *requestFactory) {
		o.userAgent = defaultUserAgent + " " + userAgent
	}
}
