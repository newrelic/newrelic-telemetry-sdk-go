package telemetry

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"sync"

	"github.com/newrelic/newrelic-telemetry-sdk-go/internal"
)

const defaultUserAgent = "NewRelic-Go-TelemetrySDK/" + version
const defaultScheme = "https"
const apiKeyHeader = "Api-Key"
const licenseKeyHeader = "X-License-Key"

// MapEntry represents a piece of the telemetry data that is included in a single
// request that should be sent to New Relic. Example MapEntry types include SpanGroup
// and the internal spanCommonBlock.
type MapEntry interface {
	// Type returns the type of data contained in this MapEntry.
	DataTypeKey() string

	// WriteDataEntry writes the json serialized bytes of the MapEntry to the buffer.
	// It returns the input buffer for chaining.
	WriteDataEntry(*bytes.Buffer) *bytes.Buffer
}

// A Batch is an array of MapEntry. A single HTTP request body is composed of
// an array of Batch.
type Batch = []MapEntry

// RequestFactory is used for sending telemetry data to New Relic when you want to have
// direct access to the http.Request and you want to manually send the request using a
// http.Client. Consider using the Harvester if you do not want to manage the requests
// and corresponding responses manually.
type RequestFactory interface {
	// BuildRequest converts the telemetry payload slice into an http.Request.
	// Do not mix telemetry data types in a single call to build request. Each
	// telemetry data type has its own RequestFactory.
	BuildRequest(context.Context, []Batch, ...ClientOption) (*http.Request, error)
}

type requestFactory struct {
	apiKeyHeader        string
	apiKey              string
	noDefaultKey        bool
	scheme              string
	endpoint            string
	path                string
	userAgent           string
	zippers             *sync.Pool
	uncompressedBuffers *sync.Pool
}

type gzipPoolEntry struct {
	compressedBuffer *bytes.Buffer
	zipper           *gzip.Writer
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

	if f.apiKey == "" && !f.noDefaultKey {
		return errors.New("api key option must be specified! (one of WithLicenseKey, WithInsertKey, or WithNoDefaultKey)")
	}
	return nil

}

func (f *hashRequestFactory) BuildRequest(ctx context.Context, batches []Batch, options ...ClientOption) (*http.Request, error) {
	return f.buildRequest(ctx, batches, bufferRequestBytes, options)
}

func (f *eventRequestFactory) BuildRequest(ctx context.Context, batches []Batch, options ...ClientOption) (*http.Request, error) {
	return f.buildRequest(ctx, batches, bufferEventRequestBytes, options)
}

type writer func(buf *bytes.Buffer, batches []Batch)

func (f *requestFactory) buildRequest(ctx context.Context, batches []Batch, bufferRequestBytes writer, options []ClientOption) (*http.Request, error) {
	configuredFactory := f
	if len(options) > 0 {
		configuredFactory = &requestFactory{
			apiKeyHeader:        f.apiKeyHeader,
			apiKey:              f.apiKey,
			noDefaultKey:        f.noDefaultKey,
			scheme:              f.scheme,
			endpoint:            f.endpoint,
			path:                f.path,
			userAgent:           f.userAgent,
			zippers:             f.zippers,
			uncompressedBuffers: f.uncompressedBuffers,
		}

		err := configure(configuredFactory, options)

		if err != nil {
			return &http.Request{}, errors.New("unable to configure this request based on options passed in")
		}
	}

	// Grab a buffer from the cached buffers and reset it
	decompressedBuffer := configuredFactory.uncompressedBuffers.Get().(*bytes.Buffer)
	defer configuredFactory.uncompressedBuffers.Put(decompressedBuffer)
	decompressedBuffer.Reset()

	// Grab a gzip structure (and buffer) from the cache and reset it
	poolEntry := configuredFactory.zippers.Get().(*gzipPoolEntry)
	defer configuredFactory.zippers.Put(poolEntry)
	poolEntry.compressedBuffer.Reset()
	poolEntry.zipper.Reset(poolEntry.compressedBuffer)

	// Generate the payload
	bufferRequestBytes(decompressedBuffer, batches)

	// Compress the payload
	err := internal.CompressWithWriter(decompressedBuffer.Bytes(), poolEntry.zipper)
	if err != nil {
		return &http.Request{}, err
	}

	// The following buffers are no longer used after this point:
	// * decompressedBuffer
	// * poolEntry.compressedBuffer
	requestBytes := make([]byte, len(poolEntry.compressedBuffer.Bytes()))
	copy(requestBytes, poolEntry.compressedBuffer.Bytes())

	getBody := func() (io.ReadCloser, error) {
		return ioutil.NopCloser(bytes.NewBuffer(requestBytes)), nil
	}

	var contentLength = int64(len(requestBytes))
	body, _ := getBody()
	endpoint := configuredFactory.endpoint
	headers := configuredFactory.getHeaders()

	request := &http.Request{
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
	}
	return request.WithContext(ctx), nil
}

func (f *requestFactory) getHeaders() http.Header {
	return http.Header{
		"Content-Type":     []string{"application/json"},
		"Content-Encoding": []string{"gzip"},
		f.apiKeyHeader:     []string{f.apiKey},
		"User-Agent":       []string{f.userAgent},
	}
}

func bufferRequestBytes(buf *bytes.Buffer, batches []Batch) {
	buf.WriteByte('[')
	for i, batch := range batches {
		if i > 0 {
			buf.WriteByte(',')
		}
		buf.WriteByte('{')
		w := internal.JSONFieldsWriter{Buf: buf}
		for _, mapEntry := range batch {
			w.AddKey(mapEntry.DataTypeKey())
			mapEntry.WriteDataEntry(buf)
		}
		buf.WriteByte('}')
	}
	buf.WriteByte(']')
}

func bufferEventRequestBytes(buf *bytes.Buffer, batches []Batch) {
	buf.WriteByte('[')
	count := 0
	for _, batch := range batches {
		for _, mapEntry := range batch {
			if count > 0 {
				buf.WriteByte(',')
			}
			mapEntry.WriteDataEntry(buf)
			count++
		}
	}
	buf.WriteByte(']')
}

// ClientOption is a function that can be used to configure the RequestFactory
// or a generated request.
type ClientOption func(o *requestFactory)

// NewSpanRequestFactory creates a new instance of a RequestFactory that can be used to send Span data to New Relic,
func NewSpanRequestFactory(options ...ClientOption) (RequestFactory, error) {
	f := &requestFactory{
		apiKeyHeader:        apiKeyHeader,
		endpoint:            "trace-api.newrelic.com",
		path:                "/trace/v1",
		userAgent:           defaultUserAgent,
		scheme:              defaultScheme,
		zippers:             newGzipPool(gzip.DefaultCompression),
		uncompressedBuffers: newUncompressedBufferPool(),
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
		apiKeyHeader:        apiKeyHeader,
		endpoint:            "metric-api.newrelic.com",
		path:                "/metric/v1",
		userAgent:           defaultUserAgent,
		scheme:              defaultScheme,
		zippers:             newGzipPool(gzip.DefaultCompression),
		uncompressedBuffers: newUncompressedBufferPool(),
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
		apiKeyHeader:        apiKeyHeader,
		endpoint:            "insights-collector.newrelic.com",
		path:                "/v1/accounts/events",
		userAgent:           defaultUserAgent,
		scheme:              defaultScheme,
		zippers:             newGzipPool(gzip.DefaultCompression),
		uncompressedBuffers: newUncompressedBufferPool(),
	}
	err := configure(f, options)
	if err != nil {
		return nil, err
	}

	return &eventRequestFactory{requestFactory: f}, nil
}

// NewLogRequestFactory creates a new instance of a RequestFactory that can be used to send Log data to New Relic.
func NewLogRequestFactory(options ...ClientOption) (RequestFactory, error) {
	f := &requestFactory{
		apiKeyHeader:        apiKeyHeader,
		endpoint:            "log-api.newrelic.com",
		path:                "/log/v1",
		userAgent:           defaultUserAgent,
		scheme:              defaultScheme,
		zippers:             newGzipPool(gzip.DefaultCompression),
		uncompressedBuffers: newUncompressedBufferPool(),
	}
	err := configure(f, options)
	if err != nil {
		return nil, err
	}

	return &hashRequestFactory{requestFactory: f}, nil
}

func newGzipPool(gzipLevel int) *sync.Pool {
	return &sync.Pool{New: func() interface{} {
		var buffer bytes.Buffer
		z, _ := gzip.NewWriterLevel(&buffer, gzipLevel)
		return &gzipPoolEntry{compressedBuffer: &buffer, zipper: z}
	}}
}

func newUncompressedBufferPool() *sync.Pool {
	return &sync.Pool{New: func() interface{} {
		var buffer bytes.Buffer
		return &buffer
	}}
}

// WithInsertKey creates a ClientOption to specify the insert key to use when generating requests.
func WithInsertKey(insertKey string) ClientOption {
	return func(o *requestFactory) {
		o.apiKeyHeader = apiKeyHeader
		o.apiKey = insertKey
	}
}

// WithLicenseKey creates a ClientOption to specify the license key to use when generating requests.
func WithLicenseKey(licenseKey string) ClientOption {
	return func(o *requestFactory) {
		o.apiKeyHeader = licenseKeyHeader
		o.apiKey = licenseKey
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

// WithInsecure creates a ClientOption to specify that requests should be sent over http instead of https.
func WithInsecure() ClientOption {
	return func(o *requestFactory) {
		o.scheme = "http"
	}
}

// WithGzipCompressionLevel creates a ClientOption to specify the level of gzip compression that should be used for the request.
func WithGzipCompressionLevel(level int) ClientOption {
	return func(o *requestFactory) {
		// If the gzip compression level is invalid, the gzip pool is not overridden
		if _, err := gzip.NewWriterLevel(nil, level); err != nil {
			o.zippers = newGzipPool(level)
		}
	}
}

// withScheme is meant to be used with the harvester because the harvester requires specifying
// an absolute uri which includes the scheme.
func withScheme(scheme string) ClientOption {
	return func(o *requestFactory) {
		o.scheme = scheme
	}
}
