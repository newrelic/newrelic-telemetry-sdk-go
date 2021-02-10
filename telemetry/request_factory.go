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

// PayloadEntry represents a piece of the telemetry data that is included in a single
// request that should be sent to New Relic. Example PayloadEntry types include SpanBatch
// and the internal spanCommonBlock.
type PayloadEntry interface {
	// Type returns the type of data contained in this PayloadEntry.
	Type() string
	// Bytes returns the json serialized bytes of the PayloadEntry.
	Bytes() []byte
}

// RequestFactory is used for sending telemetry data to New Relic when you want to have
// direct access to the http.Request and you want to manually send the request using a
// http.Client. Consider using the Harvester if you do not want to manage the requests
// and corresponding responses manually.
type RequestFactory interface {
	// BuildRequest converts the telemetry payload entries into an http.Request.
	// Do not mix telemetry data types in a single call to build request. Each 
	// telemetry data type has its own RequestFactory.
	BuildRequest([]PayloadEntry, ...ClientOption) (*http.Request, error)
}

type requestFactory struct {
	insertKey     string
	noDefaultKey  bool
	host          string
	path          string
	userAgent     string
	useRawPayload bool
}

type hashRequestFactory struct {
	* requestFactory
}

type eventRequestFactory struct {
	* requestFactory
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

func (f *hashRequestFactory) BuildRequest(entries []PayloadEntry, options ...ClientOption) (*http.Request, error) {
	return f.buildRequest(entries, getHashedPayloadBytes, options)
}

func (f *eventRequestFactory) BuildRequest(entries []PayloadEntry, options ...ClientOption) (*http.Request, error) {
	return f.buildRequest(entries, getEventPayloadBytes, options)
}

type payloadWriter func(buf *bytes.Buffer, entries []PayloadEntry)

func (f *requestFactory) buildRequest(entries []PayloadEntry, getPayloadBytes payloadWriter, options []ClientOption) (*http.Request, error) {
	configuredFactory := f
	if (len(options) > 0) {
		configuredFactory = &requestFactory{
			insertKey:     f.insertKey,
			noDefaultKey:  f.noDefaultKey,
			host:          f.host,
			path:          f.path,
			userAgent:     f.userAgent,
		}

		err := configure(configuredFactory, options)

		if err != nil {
			return &http.Request{}, errors.New("unable to configure this request based on options passed in")
		}
	}

	buf := &bytes.Buffer{}
	getPayloadBytes(buf, entries)

	buf, err := internal.Compress(buf.Bytes())
	if err != nil {
		return &http.Request{}, err
	}

	getBody := func() (io.ReadCloser, error) {
		return ioutil.NopCloser(bytes.NewBuffer(buf.Bytes())), nil
	}

	var contentLength = int64(buf.Len())
	body, _ := getBody()
	host := configuredFactory.host
	headers := configuredFactory.getHeaders()

	return &http.Request{
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

func getHashedPayloadBytes(buf *bytes.Buffer, entries []PayloadEntry) {
	buf.Write([]byte{'[', '{'})
	w := internal.JSONFieldsWriter{Buf: buf}

	for _, entry := range entries {
		w.RawField(entry.Type(), entry.Bytes())
	}

	buf.Write([]byte{'}', ']'})
}

func getEventPayloadBytes(buf *bytes.Buffer, entries []PayloadEntry) {
	buf.WriteByte('[')

	for idx, entry := range entries {
		if (idx > 0) {
			buf.WriteByte(',')
		}
		buf.Write(entry.Bytes())
	}

	buf.WriteByte(']')
}

// ClientOption is a function that can be used to configure the RequestFactory
// or a generated request.
type ClientOption func(o *requestFactory)

// NewSpanRequestFactory creates a new instance of a RequestFactory that can be used to send Span data to New Relic,
func NewSpanRequestFactory(options ...ClientOption) (RequestFactory, error) {
	f := &requestFactory{host: "trace-api.newrelic.com", path: "/trace/v1", userAgent: defaultUserAgent}
	err := configure(f, options)
	if err != nil {
		return nil, err
	}

	return &hashRequestFactory{requestFactory: f}, nil
}

// NewMetricRequestFactory creates a new instance of a RequestFactory that can be used to send Metric data to New Relic.
func NewMetricRequestFactory(options ...ClientOption) (RequestFactory, error) {
	f := &requestFactory{host: "metric-api.newrelic.com", path: "/metric/v1", userAgent: defaultUserAgent}
	err := configure(f, options)
	if err != nil {
		return nil, err
	}

	return &hashRequestFactory{requestFactory: f}, nil
}

// NewEventRequestFactory creates a new instance of a RequestFactory that can be used to send Event data to New Relic.
func NewEventRequestFactory(options ...ClientOption) (RequestFactory, error) {
	f := &requestFactory{host: "insights-collector.newrelic.com", path: "/v1/accounts/events", userAgent: defaultUserAgent}
	err := configure(f, options)
	if err != nil {
		return nil, err
	}

	return &eventRequestFactory{requestFactory: f}, nil
}

// WithInsertKey creates a ClientOption to specify the api key to use when generating requests.
func WithInsertKey(insertKey string) ClientOption {
	return func(o *requestFactory) {
		o.insertKey = insertKey
	}
}

// WithNoDefaultKey creates a ClientOption to specify that each time a request is generated the api key will
// need to be provided as a ClientOption to BuildRequest.
func WithNoDefaultKey() ClientOption {
	return func(o *requestFactory) {
		o.noDefaultKey = true
	}
}

// WithHost creates a ClientOption to specify the host to use for the generated requests.
func WithHost(host string) ClientOption {
	return func(o *requestFactory) {
		o.host = host
	}
}

// WithUserAgent creates a ClientOption to specify additional user agent information for the generated requests.
func WithUserAgent(userAgent string) ClientOption {
	return func(o *requestFactory) {
		o.userAgent = defaultUserAgent + " " + userAgent
	}
}
