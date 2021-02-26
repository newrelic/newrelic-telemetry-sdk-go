package telemetry

import (
	"bytes"
	"compress/gzip"
	"errors"
	"github.com/newrelic/newrelic-telemetry-sdk-go/internal/uuid"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"sync"

	"github.com/newrelic/newrelic-telemetry-sdk-go/internal"
)

const defaultUserAgent = "NewRelic-Go-TelemetrySDK/" + version
const defaultScheme = "https"

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
	insertKey    string
	noDefaultKey bool
	scheme       string
	endpoint     string
	path         string
	userAgent    string
	zippers      *sync.Pool
}

type hashRequestFactory struct {
	*requestFactory
}

type eventRequestFactory struct {
	*requestFactory
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
	if len(options) > 0 {
		configuredFactory = &requestFactory{
			insertKey:    f.insertKey,
			noDefaultKey: f.noDefaultKey,
			scheme:       f.scheme,
			endpoint:     f.endpoint,
			path:         f.path,
			userAgent:    f.userAgent,
			zippers:      f.zippers,
		}

		err := configure(configuredFactory, options)

		if err != nil {
			return &http.Request{}, errors.New("unable to configure this request based on options passed in")
		}
	}

	buf := &bytes.Buffer{}
	getPayloadBytes(buf, entries)

	var compressedBuffer bytes.Buffer
	zipper := configuredFactory.zippers.Get().(*gzip.Writer)
	defer configuredFactory.zippers.Put(zipper)
	zipper.Reset(&compressedBuffer)
	err := internal.CompressWithWriter(buf.Bytes(), zipper)
	if err != nil {
		return &http.Request{}, err
	}
	buf = &compressedBuffer

	getBody := func() (io.ReadCloser, error) {
		return ioutil.NopCloser(bytes.NewBuffer(buf.Bytes())), nil
	}

	var contentLength = int64(buf.Len())
	body, _ := getBody()
	endpoint := configuredFactory.endpoint
	headers := configuredFactory.getHeaders()

	return &http.Request{
		Method: "POST",
		URL: &url.URL{
			Scheme: configuredFactory.scheme,
			Host:   configuredFactory.endpoint,
			Path:   configuredFactory.path,
		},
		Header:        headers,
		Body:          body,
		GetBody:       getBody,
		ContentLength: contentLength,
		Close:         false,
		Host:          endpoint,
	}, nil
}

func (f *requestFactory) getHeaders() http.Header {
	return http.Header{
		"Content-Type":     []string{"application/json"},
		"Content-Encoding": []string{"gzip"},
		"Api-Key":          []string{f.insertKey},
		"User-Agent":       []string{f.userAgent},
		"x-request-id":     []string{uuid.NewString()},
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
		if idx > 0 {
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
	f := &requestFactory{
		endpoint:  "trace-api.newrelic.com",
		path:      "/trace/v1",
		userAgent: defaultUserAgent,
		scheme:    defaultScheme,
		zippers:   newGzipPool(),
	}
	err := configure(f, options)
	if err != nil {
		return nil, err
	}

	return &hashRequestFactory{requestFactory: f}, nil
}

// NewMetricRequestFactory creates a new instance of a RequestFactory that can be used to send Metric data to New Relic.
func NewMetricRequestFactory(options ...ClientOption) (RequestFactory, error) {
	f := &requestFactory{
		endpoint:  "metric-api.newrelic.com",
		path:      "/metric/v1",
		userAgent: defaultUserAgent,
		scheme:    defaultScheme,
		zippers:   newGzipPool(),
	}
	err := configure(f, options)
	if err != nil {
		return nil, err
	}

	return &hashRequestFactory{requestFactory: f}, nil
}

// NewEventRequestFactory creates a new instance of a RequestFactory that can be used to send Event data to New Relic.
func NewEventRequestFactory(options ...ClientOption) (RequestFactory, error) {
	f := &requestFactory{
		endpoint:  "insights-collector.newrelic.com",
		path:      "/v1/accounts/events",
		userAgent: defaultUserAgent,
		scheme:    defaultScheme,
		zippers:   newGzipPool(),
	}
	err := configure(f, options)
	if err != nil {
		return nil, err
	}

	return &eventRequestFactory{requestFactory: f}, nil
}

func newGzipPool() *sync.Pool {
	pool := sync.Pool{New: func() interface{} {
		return gzip.NewWriter(nil)
	}}
	return &pool
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

// WithEndpoint creates a ClientOption to specify the hostname and port to use for the generated requests.
func WithEndpoint(endpoint string) ClientOption {
	return func(o *requestFactory) {
		o.endpoint = endpoint
	}
}

// WithUserAgent creates a ClientOption to specify additional user agent information for the generated requests.
func WithUserAgent(userAgent string) ClientOption {
	return func(o *requestFactory) {
		o.userAgent = defaultUserAgent + " " + userAgent
	}
}

// WithInsecure creates a ClientOption to speficy that requests should be sent over http instead of https.
func WithInsecure() ClientOption {
	return func(o *requestFactory) {
		o.scheme = "http"
	}
}

// withScheme is meant to be used with the harvester because the harvester requires specifying
// an absolute uri which includes the scheme.
func withScheme(scheme string) ClientOption {
	return func(o *requestFactory) {
		o.scheme = scheme
	}
}
