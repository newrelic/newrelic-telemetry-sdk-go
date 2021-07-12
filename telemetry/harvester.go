// Copyright 2019 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package telemetry

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/newrelic/newrelic-telemetry-sdk-go/internal"
)

// Harvester aggregates and reports metrics and spans.
type Harvester struct {
	// These fields are not modified after Harvester creation.  They may be
	// safely accessed without locking.
	config           Config
	commonAttributes *commonAttributes

	// lock protects the mutable fields below.
	lock                 sync.Mutex
	lastHarvest          time.Time
	rawMetrics           []Metric
	aggregatedMetrics    map[metricIdentity]*metric
	spans                []Span
	events               []Event
	logs                 []Log
	spanRequestFactory   RequestFactory
	metricRequestFactory RequestFactory
	eventRequestFactory  RequestFactory
	logRequestFactory    RequestFactory
}

const (
	// NOTE:  These constant values are used in Config field doc comments.
	defaultHarvestPeriod  = 5 * time.Second
	defaultHarvestTimeout = 15 * time.Second

	// euKeyPrefix is used to sanitize the api-key for logging.
	euKeyPrefix = "eu01xx"
)

var (
	errAPIKeyUnset = errors.New("APIKey is required")
)

// NewHarvester creates a new harvester.
func NewHarvester(options ...func(*Config)) (*Harvester, error) {
	cfg := Config{
		Client:         &http.Client{},
		HarvestPeriod:  defaultHarvestPeriod,
		HarvestTimeout: defaultHarvestTimeout,
	}
	for _, opt := range options {
		opt(&cfg)
	}

	if cfg.APIKey == "" {
		return nil, errAPIKeyUnset
	}

	h := &Harvester{
		config:            cfg,
		lastHarvest:       time.Now(),
		aggregatedMetrics: make(map[metricIdentity]*metric),
	}

	// Marshal the common attributes to JSON here to avoid doing it on every
	// harvest.  This also has the benefit that it avoids race conditions if
	// the consumer modifies the CommonAttributes map after calling
	// NewHarvester.
	if nil != h.config.CommonAttributes {
		commonAttributes, err := newCommonAttributes(h.config.CommonAttributes)
		if err != nil {
			h.config.logError(map[string]interface{}{"err": err.Error()})
		}
		h.commonAttributes = commonAttributes
		h.config.CommonAttributes = nil
	}

	spanURL, err := url.Parse(h.config.spanURL())
	if nil != err {
		return nil, err
	}

	userAgent := "harvester " + h.config.userAgent()

	h.spanRequestFactory, err = NewSpanRequestFactory(
		WithInsertKey(h.config.APIKey),
		withScheme(spanURL.Scheme),
		WithEndpoint(spanURL.Host),
		WithUserAgent(userAgent),
	)
	if err != nil {
		return nil, err
	}

	metricURL, err := url.Parse(h.config.metricURL())
	if nil != err {
		return nil, err
	}

	h.metricRequestFactory, err = NewMetricRequestFactory(
		WithInsertKey(h.config.APIKey),
		withScheme(metricURL.Scheme),
		WithEndpoint(metricURL.Host),
		WithUserAgent(userAgent),
	)
	if err != nil {
		return nil, err
	}

	eventURL, err := url.Parse(h.config.eventURL())
	if nil != err {
		return nil, err
	}

	h.eventRequestFactory, err = NewEventRequestFactory(
		WithInsertKey(h.config.APIKey),
		withScheme(eventURL.Scheme),
		WithEndpoint(eventURL.Host),
		WithUserAgent(userAgent),
	)
	if err != nil {
		return nil, err
	}

	logURL, err := url.Parse(h.config.logURL())
	if err != nil {
		return nil, err
	}

	h.logRequestFactory, err = NewLogRequestFactory(
		WithInsertKey(h.config.APIKey),
		withScheme(logURL.Scheme),
		WithEndpoint(logURL.Host),
		WithUserAgent(userAgent),
	)
	if err != nil {
		return nil, err
	}

	h.config.logDebug(map[string]interface{}{
		"event":                  "harvester created",
		"api-key":                sanitizeAPIKeyForLogging(h.config.APIKey),
		"harvest-period-seconds": h.config.HarvestPeriod.Seconds(),
		"metrics-url-override":   h.config.MetricsURLOverride,
		"spans-url-override":     h.config.SpansURLOverride,
		"events-url-override":    h.config.EventsURLOverride,
		"logs-url-override":      h.config.LogsURLOverride,
		"version":                version,
	})

	if h.config.HarvestPeriod != 0 {
		go harvestRoutine(h)
	}

	return h, nil
}

func sanitizeAPIKeyForLogging(apiKey string) string {
	if len(apiKey) <= 8 {
		return apiKey
	}
	end := 8
	if strings.HasPrefix(apiKey, euKeyPrefix) {
		end += len(euKeyPrefix)
	}
	return apiKey[:end]
}

var (
	errSpanIDUnset     = errors.New("span id must be set")
	errTraceIDUnset    = errors.New("trace id must be set")
	errEventTypeUnset  = errors.New("eventType must be set")
	errLogMessageUnset = errors.New("log message must be set")
)

// RecordSpan records the given span.
func (h *Harvester) RecordSpan(s Span) error {
	if nil == h {
		return nil
	}
	if s.TraceID == "" {
		return errTraceIDUnset
	}
	if s.ID == "" {
		return errSpanIDUnset
	}
	if s.Timestamp.IsZero() {
		s.Timestamp = time.Now()
	}

	h.lock.Lock()
	defer h.lock.Unlock()

	h.spans = append(h.spans, s)
	return nil
}

// RecordMetric adds a fully formed metric.  This metric is not aggregated with
// any other metrics and is never dropped.  The timestamp field must be
// specified on Gauge metrics.  The timestamp/interval fields on Count and
// Summary are optional and will be assumed to be the harvester batch times if
// unset.  Use MetricAggregator() instead to aggregate metrics.
func (h *Harvester) RecordMetric(m Metric) {
	if nil == h {
		return
	}
	h.lock.Lock()
	defer h.lock.Unlock()

	if fields := m.validate(); nil != fields {
		h.config.logError(fields)
		return
	}

	h.rawMetrics = append(h.rawMetrics, m)
}

// RecordEvent records the given event.
func (h *Harvester) RecordEvent(e Event) error {
	if nil == h {
		return nil
	}
	if e.EventType == "" {
		return errEventTypeUnset
	}
	if e.Timestamp.IsZero() {
		e.Timestamp = time.Now()
	}

	h.lock.Lock()
	defer h.lock.Unlock()

	h.events = append(h.events, e)
	return nil
}

// RecordLog records the given log message.
func (h *Harvester) RecordLog(l Log) error {
	if nil == h {
		return nil
	}
	if l.Message == "" {
		return errLogMessageUnset
	}
	if l.Timestamp.IsZero() {
		l.Timestamp = time.Now()
	}

	h.lock.Lock()
	defer h.lock.Unlock()

	h.logs = append(h.logs, l)
	return nil
}

type response struct {
	statusCode int
	body       []byte
	err        error
	retryAfter string
}

var (
	backoffSequenceSeconds = []int{0, 1, 2, 4, 8, 16}
)

func (r response) needsRetry(cfg *Config, attempts int) (bool, time.Duration) {
	if attempts >= len(backoffSequenceSeconds) {
		attempts = len(backoffSequenceSeconds) - 1
	}
	backoff := time.Duration(backoffSequenceSeconds[attempts]) * time.Second

	switch r.statusCode {
	case 202, 200:
		// success
		return false, 0
	case 400, 403, 404, 405, 411, 413:
		// errors that should not retry
		return false, 0
	case 429:
		// special retry backoff time
		if r.retryAfter != "" {
			// Honor Retry-After header value in seconds
			if d, err := time.ParseDuration(r.retryAfter + "s"); nil == err {
				if d > backoff {
					return true, d
				}
			}
		}
		return true, backoff
	default:
		// all other errors should retry
		return true, backoff
	}
}

func postData(req *http.Request, client *http.Client) response {
	resp, err := client.Do(req)
	if nil != err {
		return response{err: fmt.Errorf("error posting data: %v", err)}
	}
	defer resp.Body.Close()

	r := response{
		statusCode: resp.StatusCode,
		retryAfter: resp.Header.Get("Retry-After"),
	}

	// On success, metrics ingest returns 202, span ingest returns 200.
	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusAccepted {
		r.body, _ = ioutil.ReadAll(resp.Body)
	} else {
		r.err = fmt.Errorf("unexpected post response code: %d: %s",
			resp.StatusCode, http.StatusText(resp.StatusCode))
	}

	return r
}

func (h *Harvester) swapOutMetrics(now time.Time) []*http.Request {
	h.lock.Lock()
	lastHarvest := h.lastHarvest
	h.lastHarvest = now
	rawMetrics := h.rawMetrics
	h.rawMetrics = nil
	aggregatedMetrics := h.aggregatedMetrics
	h.aggregatedMetrics = make(map[metricIdentity]*metric, len(aggregatedMetrics))
	h.lock.Unlock()

	for _, m := range aggregatedMetrics {
		if nil != m.c {
			rawMetrics = append(rawMetrics, m.c)
		}
		if nil != m.s {
			rawMetrics = append(rawMetrics, m.s)
		}
		if nil != m.g {
			rawMetrics = append(rawMetrics, m.g)
		}
	}

	if len(rawMetrics) == 0 {
		return nil
	}

	commonBlock := &metricCommonBlock{
		timestamp:  lastHarvest,
		interval:   now.Sub(lastHarvest),
		attributes: h.commonAttributes,
	}
	group := &metricGroup{Metrics: rawMetrics}
	entries := []MapEntry{commonBlock, group}
	reqs, err := buildSplitRequests([]Batch{entries}, h.metricRequestFactory)
	if nil != err {
		h.config.logError(map[string]interface{}{
			"err":     err.Error(),
			"message": "error creating requests for metrics",
		})
		return nil
	}
	return reqs
}

func (h *Harvester) swapOutSpans() []*http.Request {
	h.lock.Lock()
	sps := h.spans
	h.spans = nil
	h.lock.Unlock()

	if nil == sps {
		return nil
	}

	var entries []MapEntry
	if nil != h.commonAttributes {
		entries = append(entries, &spanCommonBlock{attributes: h.commonAttributes})
	}
	entries = append(entries, &spanGroup{Spans: sps})
	reqs, err := buildSplitRequests([]Batch{entries}, h.spanRequestFactory)
	if nil != err {
		h.config.logError(map[string]interface{}{
			"err":     err.Error(),
			"message": "error creating requests for spans",
		})
		return nil
	}
	return reqs
}

func (h *Harvester) swapOutEvents() []*http.Request {
	h.lock.Lock()
	events := h.events
	h.events = nil
	h.lock.Unlock()

	if nil == events {
		return nil
	}
	group := &eventGroup{
		Events: events,
	}
	reqs, err := buildSplitRequests([]Batch{{group}}, h.eventRequestFactory)
	if nil != err {
		h.config.logError(map[string]interface{}{
			"err":     err.Error(),
			"message": "error creating requests for events",
		})
		return nil
	}
	return reqs
}

func (h *Harvester) swapOutLogs() []*http.Request {
	h.lock.Lock()
	logs := h.logs
	h.logs = nil
	h.lock.Unlock()

	if nil == logs {
		return nil
	}

	var entries []MapEntry
	if nil != h.commonAttributes {
		entries = append(entries, &logCommonBlock{attributes: h.commonAttributes})
	}
	entries = append(entries, &logGroup{Logs: logs})
	reqs, err := buildSplitRequests([]Batch{entries}, h.logRequestFactory)
	if nil != err {
		h.config.logError(map[string]interface{}{
			"err":     err.Error(),
			"message": "error creating requests for logs",
		})
		return nil
	}
	return reqs
}

func harvestRequest(req *http.Request, cfg *Config, wg *sync.WaitGroup) {
	var attempts int
	defer wg.Done()
	for {
		cfg.logDebug(map[string]interface{}{
			"event":       "data post",
			"url":         req.URL.String(),
			"body-length": req.ContentLength,
		})
		// Check if the audit log is enabled to prevent unnecessarily
		// copying UncompressedBody.
		if cfg.auditLogEnabled() {
			bodyReader, _ := req.GetBody()
			compressedBody, _ := ioutil.ReadAll(bodyReader)
			uncompressedBody, _ := internal.Uncompress(compressedBody)
			cfg.logAudit(map[string]interface{}{
				"event": "uncompressed request body",
				"url":   req.URL.String(),
				"data":  jsonString(uncompressedBody),
			})
		}

		resp := postData(req, cfg.Client)

		if nil != resp.err {
			cfg.logError(map[string]interface{}{
				"err": resp.err.Error(),
			})
		} else {
			cfg.logDebug(map[string]interface{}{
				"event":  "data post response",
				"status": resp.statusCode,
				"body":   jsonOrString(resp.body),
			})
		}
		retry, backoff := resp.needsRetry(cfg, attempts)
		if !retry {
			return
		}

		tmr := time.NewTimer(backoff)
		select {
		case <-tmr.C:
		case <-req.Context().Done():
			tmr.Stop()
			if err := req.Context().Err(); err != nil {
				// NOTE: It is possible that the context was
				// cancelled/timedout right after the request
				// successfully finished.  In that case, we will
				// erroneously log a message.  I (will) don't think
				// that's worth trying to engineer around.
				cfg.logError(map[string]interface{}{
					"event":         "harvest cancelled or timed out",
					"message":       "dropping data",
					"context-error": err.Error(),
				})
			}
			return
		}
		attempts++

		// Reattach request body because the original one has already been read
		// and closed.
		originalBody, _ := req.GetBody()
		req.Body = originalBody
	}
}

// HarvestNow sends metric and span data to New Relic.  This method blocks until
// all data has been sent successfully or the Config.HarvestTimeout timeout has
// elapsed. This method can be used with a zero Config.HarvestPeriod value to
// control exactly when data is sent to New Relic servers.
func (h *Harvester) HarvestNow(ct context.Context) {
	if nil == h {
		return
	}

	ctx, cancel := context.WithTimeout(ct, h.config.HarvestTimeout)
	defer cancel()

	var reqs []*http.Request
	reqs = append(reqs, h.swapOutMetrics(time.Now())...)
	reqs = append(reqs, h.swapOutSpans()...)
	reqs = append(reqs, h.swapOutEvents()...)
	reqs = append(reqs, h.swapOutLogs()...)
	wg := sync.WaitGroup{}

	for _, req := range reqs {
		wg.Add(1)
		httpRequest := req.WithContext(ctx)
		go harvestRequest(httpRequest, &h.config, &wg)
	}
	wg.Wait()
}

func harvestRoutine(h *Harvester) {
	// Introduce a small jitter to ensure the backend isn't hammered if many
	// harvesters start at once.
	d := minDuration(h.config.HarvestPeriod, 3*time.Second)
	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))
	jitter := time.Nanosecond * time.Duration(rnd.Int63n(d.Nanoseconds()))
	time.Sleep(jitter)

	ticker := time.NewTicker(h.config.HarvestPeriod)
	for range ticker.C {
		go h.HarvestNow(context.Background())
	}
}

type metricIdentity struct {
	// Note that the type is not a field here since a single 'metric' type
	// may contain a count, gauge, and summary.
	Name           string
	attributesJSON string
}

type metric struct {
	s *Summary
	c *Count
	g *Gauge
}

type metricHandle struct {
	metricIdentity
	harvester *Harvester
}

func newMetricHandle(h *Harvester, name string, attributes map[string]interface{}) metricHandle {
	return metricHandle{
		harvester: h,
		metricIdentity: metricIdentity{
			attributesJSON: string(internal.MarshalOrderedAttributes(attributes)),
			Name:           name,
		},
	}
}

// findOrCreateMetric finds or creates the metric associated with the given
// identity.  This function assumes the Harvester is locked.
func (h *Harvester) findOrCreateMetric(identity metricIdentity) *metric {
	m := h.aggregatedMetrics[identity]
	if nil == m {
		// this happens the first time we update the value,
		// or after a harvest when the metric is removed.
		m = &metric{}
		h.aggregatedMetrics[identity] = m
	}
	return m
}

// MetricAggregator is used to aggregate individual data points into metrics.
type MetricAggregator struct {
	harvester *Harvester
}

// MetricAggregator returns a metric aggregator.  Use this instead of
// RecordMetric if you have individual data points that you would like to
// combine into metrics.
func (h *Harvester) MetricAggregator() *MetricAggregator {
	if nil == h {
		return nil
	}
	return &MetricAggregator{harvester: h}
}

// Count creates a new AggregatedCount metric.
func (ag *MetricAggregator) Count(name string, attributes map[string]interface{}) *AggregatedCount {
	if nil == ag {
		return nil
	}
	return &AggregatedCount{metricHandle: newMetricHandle(ag.harvester, name, attributes)}
}

// Gauge creates a new AggregatedGauge metric.
func (ag *MetricAggregator) Gauge(name string, attributes map[string]interface{}) *AggregatedGauge {
	if nil == ag {
		return nil
	}
	return &AggregatedGauge{metricHandle: newMetricHandle(ag.harvester, name, attributes)}
}

// Summary creates a new AggregatedSummary metric.
func (ag *MetricAggregator) Summary(name string, attributes map[string]interface{}) *AggregatedSummary {
	if nil == ag {
		return nil
	}
	return &AggregatedSummary{metricHandle: newMetricHandle(ag.harvester, name, attributes)}
}
