// Copyright 2019 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package telemetry

import (
	"bytes"
	"encoding/json"
	"math"
	"time"

	"github.com/newrelic/newrelic-telemetry-sdk-go/internal"
)

const metricTypeName string = "metrics"

// Count is the metric type that counts the number of times an event occurred.
// This counter should be reset every time the data is reported, meaning the
// value reported represents the difference in count over the reporting time
// window.
//
// Example possible uses:
//
//  * the number of messages put on a topic
//  * the number of HTTP requests
//  * the number of errors thrown
//  * the number of support tickets answered
//
type Count struct {
	// Name is the name of this metric.
	Name string
	// Attributes is a map of attributes for this metric.
	Attributes map[string]interface{}
	// AttributesJSON is a json.RawMessage of attributes for this metric. It
	// will only be sent if Attributes is nil.
	AttributesJSON json.RawMessage
	// Value is the value of this metric.
	Value float64
	// Timestamp is the start time of this metric's interval.  If Timestamp
	// is unset then the Harvester's period start will be used.
	Timestamp time.Time
	// Interval is the length of time for this metric.  If Interval is unset
	// then the time between Harvester harvests will be used.
	Interval time.Duration
	// Set to true to force the value of interval to be written to the payload
	ForceIntervalValid bool
}

func (m Count) validate() map[string]interface{} {
	if err := isFloatValid(m.Value); err != nil {
		return map[string]interface{}{
			"message": "invalid count value",
			"name":    m.Name,
			"err":     err.Error(),
		}
	}
	return nil
}

// Metric is implemented by Count, Gauge, and Summary.
type Metric interface {
	writeJSON(buf *bytes.Buffer)
	validate() map[string]interface{}
}

func writeTimestampInterval(w *internal.JSONFieldsWriter, timestamp time.Time, interval time.Duration, forceIntervalValid bool) {
	if !timestamp.IsZero() {
		w.IntField("timestamp", timestamp.UnixNano()/(1000*1000))
	}
	if interval != 0 || forceIntervalValid {
		w.IntField("interval.ms", interval.Milliseconds())
	}
}

func (m Count) writeJSON(buf *bytes.Buffer) {
	w := internal.JSONFieldsWriter{Buf: buf}
	w.Buf.WriteByte('{')
	w.StringField("name", m.Name)
	w.StringField("type", "count")
	w.FloatField("value", m.Value)
	writeTimestampInterval(&w, m.Timestamp, m.Interval, m.ForceIntervalValid)
	if nil != m.Attributes {
		w.WriterField("attributes", internal.Attributes(m.Attributes))
	} else if nil != m.AttributesJSON {
		w.RawField("attributes", m.AttributesJSON)
	}
	w.Buf.WriteByte('}')
}

// Summary is the metric type used for reporting aggregated information about
// discrete events.   It provides the count, average, sum, min and max values
// over time.  All fields should be reset to 0 every reporting interval.
//
// Example possible uses:
//
//  * the duration and count of spans
//  * the duration and count of transactions
//  * the time each message spent in a queue
//
type Summary struct {
	// Name is the name of this metric.
	Name string
	// Attributes is a map of attributes for this metric.
	Attributes map[string]interface{}
	// AttributesJSON is a json.RawMessage of attributes for this metric. It
	// will only be sent if Attributes is nil.
	AttributesJSON json.RawMessage
	// Count is the count of occurrences of this metric for this time period.
	Count float64
	// Sum is the sum of all occurrences of this metric for this time period.
	Sum float64
	// Min is the smallest value recorded of this metric for this time period.
	Min float64
	// Max is the largest value recorded of this metric for this time period.
	Max float64
	// Timestamp is the start time of this metric's interval.   If Timestamp
	// is unset then the Harvester's period start will be used.
	Timestamp time.Time
	// Interval is the length of time for this metric.  If Interval is unset
	// then the time between Harvester harvests will be used.
	Interval time.Duration
	// Set to true to force the value of interval to be written to the payload
	ForceIntervalValid bool
}

func (m Summary) validate() map[string]interface{} {
	for _, v := range []float64{
		m.Count,
		m.Sum,
	} {
		if err := isFloatValid(v); err != nil {
			return map[string]interface{}{
				"message": "invalid summary field",
				"name":    m.Name,
				"err":     err.Error(),
			}
		}
	}

	for _, v := range []float64{
		m.Min,
		m.Max,
	} {
		if math.IsInf(v, 0) {
			return map[string]interface{}{
				"message": "invalid summary field",
				"name":    m.Name,
				"err":     errFloatInfinity.Error(),
			}
		}
	}

	return nil
}

func (m Summary) writeJSON(buf *bytes.Buffer) {
	w := internal.JSONFieldsWriter{Buf: buf}
	buf.WriteByte('{')

	w.StringField("name", m.Name)
	w.StringField("type", "summary")

	w.AddKey("value")
	buf.WriteByte('{')
	vw := internal.JSONFieldsWriter{Buf: buf}
	vw.FloatField("sum", m.Sum)
	vw.FloatField("count", m.Count)
	if math.IsNaN(m.Min) {
		w.RawField("min", json.RawMessage(`null`))
	} else {
		vw.FloatField("min", m.Min)
	}
	if math.IsNaN(m.Max) {
		vw.RawField("max", json.RawMessage(`null`))
	} else {
		vw.FloatField("max", m.Max)
	}
	buf.WriteByte('}')

	writeTimestampInterval(&w, m.Timestamp, m.Interval, m.ForceIntervalValid)
	if nil != m.Attributes {
		w.WriterField("attributes", internal.Attributes(m.Attributes))
	} else if nil != m.AttributesJSON {
		w.RawField("attributes", m.AttributesJSON)
	}
	buf.WriteByte('}')
}

// Gauge is the metric type that records a value that can increase or decrease.
// It generally represents the value for something at a particular moment in
// time.  One typically records a Gauge value on a set interval.
//
// Example possible uses:
//
//  * the temperature in a room
//  * the amount of memory currently in use for a process
//  * the bytes per second flowing into Kafka at this exact moment in time
//  * the current speed of your car
//
type Gauge struct {
	// Name is the name of this metric.
	Name string
	// Attributes is a map of attributes for this metric.
	Attributes map[string]interface{}
	// AttributesJSON is a json.RawMessage of attributes for this metric. It
	// will only be sent if Attributes is nil.
	AttributesJSON json.RawMessage
	// Value is the value of this metric.
	Value float64
	// Timestamp is the time at which this metric was gathered.  If
	// Timestamp is unset then the Harvester's period start will be used.
	Timestamp time.Time
}

func (m Gauge) validate() map[string]interface{} {
	if err := isFloatValid(m.Value); err != nil {
		return map[string]interface{}{
			"message": "invalid gauge field",
			"name":    m.Name,
			"err":     err.Error(),
		}
	}
	return nil
}

func (m Gauge) writeJSON(buf *bytes.Buffer) {
	w := internal.JSONFieldsWriter{Buf: buf}
	buf.WriteByte('{')
	w.StringField("name", m.Name)
	w.StringField("type", "gauge")
	w.FloatField("value", m.Value)
	writeTimestampInterval(&w, m.Timestamp, 0, false)
	if nil != m.Attributes {
		w.WriterField("attributes", internal.Attributes(m.Attributes))
	} else if nil != m.AttributesJSON {
		w.RawField("attributes", m.AttributesJSON)
	}
	buf.WriteByte('}')
}

type metricCommonBlock struct {
	// timestamp is the start time of all metrics in the MetricGroup.  This value
	// can be overridden by setting timestamp on any particular metric.
	// timestamp must be set here or on all metrics.
	timestamp time.Time
	// interval is the length of time for all metrics in the MetricGroup.  This
	// value can be overridden by setting interval on any particular Count or
	// Summary metric.  interval must be set to a non-zero value here or on
	// all Count and Summary metrics.
	interval time.Duration
	// Set to true to force the value of interval to be written to the payload
	forceIntervalValid bool
	// attributes is the reference to the common attributes that apply to
	// all metrics in the batch.
	attributes *commonAttributes
}

// Type returns the type of data contained in this MapEntry.
func (mcb *metricCommonBlock) DataTypeKey() string {
	return "common"
}

// WriteBytes writes the json serialized bytes of the MapEntry to the buffer.
func (mcb *metricCommonBlock) WriteDataEntry(buf *bytes.Buffer) *bytes.Buffer {
	buf.WriteByte('{')
	w := internal.JSONFieldsWriter{Buf: buf}
	writeTimestampInterval(&w, mcb.timestamp, mcb.interval, mcb.forceIntervalValid)
	if nil != mcb.attributes && nil != mcb.attributes.RawJSON {
		w.AddKey(mcb.attributes.DataTypeKey())
		mcb.attributes.WriteDataEntry(buf)
	}
	buf.WriteByte('}')
	return buf
}

// MetricCommonBlockOption is a function that can be used to configure a metric common block
type MetricCommonBlockOption func(block *metricCommonBlock) error

// NewMetricCommonBlock creates a new MapEntry representing data common to all metrics in a group.
func NewMetricCommonBlock(options ...MetricCommonBlockOption) (MapEntry, error) {
	common := &metricCommonBlock{}
	for _, option := range options {
		err := option(common)
		if err != nil {
			return nil, err
		}
	}
	return common, nil
}

// WithMetricAttributes creates a MetricCommonBlockOption to specify the common attributes of the common block.
// Invalid attributes will be detected and ignored
func WithMetricAttributes(commonAttributes map[string]interface{}) MetricCommonBlockOption {
	return func(b *metricCommonBlock) error {
		validCommonAttr, err := newCommonAttributes(commonAttributes)
		if err != nil {
			// Ignore any error with invalid attributes
			if _, ok := err.(errInvalidAttributes); !ok {
				return err
			}
		}
		b.attributes = validCommonAttr
		return nil
	}
}

// WithMetricInterval sets the interval in a common block for metrics.
func WithMetricInterval(interval time.Duration) MetricCommonBlockOption {
	return func(b *metricCommonBlock) error {
		b.interval = interval
		b.forceIntervalValid = true
		return nil
	}
}

// WithMetricTimestamp sets the timestamp (start time) in a common block for metrics.
func WithMetricTimestamp(startTime time.Time) MetricCommonBlockOption {
	return func(b *metricCommonBlock) error {
		b.timestamp = startTime
		return nil
	}
}

// MetricGroup represents a single grouping of metrics in a payload sent to New Relic.
//
// timestamp/interval are optional and can be used to represent the start and
// duration of the batch as a whole. Individual Count and Summary metrics may
// provide timestamp/interval fields which will take priority over the batch
// timestamp/interval. This is not the case for Gauge metrics which each require
// a timestamp.
//
// attributes are any attributes that should be applied to all metrics in this
// batch. Each metric type also accepts an attributes field.
type metricGroup struct {
	// Metrics is the slice of metrics to send with this MetricGroup.
	Metrics []Metric
}

// split will split the MetricGroup into 2 equal parts, returning a slice of MetricBatches.
// If the number of metrics in the original is 0 or 1 then nil is returned.
func (group *metricGroup) split() []splittablePayloadEntry {
	if len(group.Metrics) < 2 {
		return nil
	}

	half := len(group.Metrics) / 2
	mb1 := *group
	mb1.Metrics = group.Metrics[:half]
	mb2 := *group
	mb2.Metrics = group.Metrics[half:]

	return []splittablePayloadEntry{&mb1, &mb2}
}

// DataTypeKey returns the type of data contained in this MapEntry.
func (group *metricGroup) DataTypeKey() string {
	return metricTypeName
}

// WriteDataEntry writes the json serialized bytes of the MapEntry to the buffer.
func (group *metricGroup) WriteDataEntry(buf *bytes.Buffer) *bytes.Buffer {
	buf.WriteByte('[')
	for idx, m := range group.Metrics {
		if idx > 0 {
			buf.WriteByte(',')
		}
		m.writeJSON(buf)
	}
	buf.WriteByte(']')
	return buf
}

// NewMetricGroup creates a new MapEntry representing a group of metrics in a batch.
func NewMetricGroup(metrics []Metric) MapEntry {
	return &metricGroup{Metrics: metrics}
}
