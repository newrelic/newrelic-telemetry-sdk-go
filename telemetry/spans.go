// Copyright 2019 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package telemetry

import (
	"bytes"
	"time"

	"github.com/newrelic/newrelic-telemetry-sdk-go/internal"
)

const spanTypeName string = "spans"

// Span is a distributed tracing span.
type Span struct {
	// Required Fields:
	//
	// ID is a unique identifier for this span.
	ID string
	// TraceID is a unique identifier shared by all spans within a single
	// trace.
	TraceID string
	// Timestamp is when this span started.  If Timestamp is not set, it
	// will be assigned to time.Now() in Harvester.RecordSpan.
	Timestamp time.Time

	// Recommended Fields:
	//
	// Name is the name of this span.
	Name string
	// ParentID is the span id of the previous caller of this span.  This
	// can be empty if this is the first span.
	ParentID string
	// Duration is the duration of this span.  This field will be reported
	// in milliseconds.
	Duration time.Duration
	// ServiceName is the name of the service that created this span.
	ServiceName string

	// Additional Fields:
	//
	// Attributes is a map of user specified tags on this span.  The map
	// values can be any of bool, number, or string.
	Attributes map[string]interface{}
	// Events is a slice of events that occurred during the execution of a span.
	// This feature is a work in progress.
	Events []Event
}

func (s *Span) writeJSON(buf *bytes.Buffer) {
	w := internal.JSONFieldsWriter{Buf: buf}
	buf.WriteByte('{')

	w.StringField("id", s.ID)
	w.StringField("trace.id", s.TraceID)
	w.IntField("timestamp", s.Timestamp.UnixNano()/(1000*1000))

	w.AddKey("attributes")
	buf.WriteByte('{')
	ww := internal.JSONFieldsWriter{Buf: buf}

	if "" != s.Name {
		ww.StringField("name", s.Name)
	}
	if "" != s.ParentID {
		ww.StringField("parent.id", s.ParentID)
	}
	if 0 != s.Duration {
		ww.FloatField("duration.ms", s.Duration.Seconds()*1000.0)
	}
	if "" != s.ServiceName {
		ww.StringField("service.name", s.ServiceName)
	}

	internal.AddAttributes(&ww, s.Attributes)
	buf.WriteByte('}')

	if len(s.Events) > 0 {
		w.AddKey("events")
		buf.WriteByte('[')
		for i, e := range s.Events {
			if i > 0 {
				buf.WriteByte(',')
			}
			buf.WriteByte('{')
			aw := internal.JSONFieldsWriter{Buf: buf}
			aw.StringField("name", e.EventType)
			aw.IntField("timestamp", e.Timestamp.UnixNano()/(1000*1000))
			aw.AddKey("attributes")
			buf.WriteByte('{')
			aw.NoComma()
			internal.AddAttributes(&aw, e.Attributes)
			buf.WriteByte('}')
			buf.WriteByte('}')
		}
		buf.WriteByte(']')
	}

	buf.WriteByte('}')
}

// spanCommonBlock represents the shared elements of a SpanBatch.
type spanCommonBlock struct {
	Attributes *commonAttributes
}

// Type returns the type of data contained in this MapEntry.
func (c *spanCommonBlock) Type() string {
	return "common"
}

// Bytes returns the json serialized bytes of the MapEntry.
func (c *spanCommonBlock) Bytes() []byte {
	buf := &bytes.Buffer{}
	buf.WriteByte('{')
	w := internal.JSONFieldsWriter{Buf: buf}
	w.RawField(c.Attributes.Type(), c.Attributes.Bytes())
	buf.WriteByte('}')
	return buf.Bytes()
}

// SpanCommonBlockOption is a function that can be used to configure a spanCommonBlock
type SpanCommonBlockOption func(scb *spanCommonBlock) error

// NewSpanCommonBlock creates a new MapEntry representing data common to all spans in a batch.
func NewSpanCommonBlock(options ...SpanCommonBlockOption) (MapEntry, error) {
	scb := &spanCommonBlock{}
	for _, option := range options {
		err := option(scb)
		if err != nil {
			return scb, err
		}
	}
	return scb, nil
}

// SpanCommonBlockBuilder is a builder for the span common block MapEntry.
type SpanCommonBlockBuilder struct {
	commonAttributes map[string]interface{}
}

// WithSpanAttributes creates a SpanCommonBlockOption to specify the common attributes of the common block.
// If invalid attributes are detected an error will be returned describing which keys were invalid, but
// the valid attributes will still be added to the span common block.
func WithSpanAttributes(commonAttributes map[string]interface{}) SpanCommonBlockOption {
	return func(scb *spanCommonBlock) error {
		validCommonAttr, err := newCommonAttributes(commonAttributes)
		scb.Attributes = validCommonAttr
		return err
	}
}

// SpanBatch represents a single batch of spans to report to New Relic.
type SpanBatch struct {
	Spans []Span
}

// Type returns the type of data contained in this MapEntry.
func (batch *SpanBatch) Type() string {
	return spanTypeName
}

// Bytes returns the json serialized bytes of the MapEntry.
func (batch *SpanBatch) Bytes() []byte {
	buf := &bytes.Buffer{}
	buf.WriteByte('[')
	for idx, s := range batch.Spans {
		if idx > 0 {
			buf.WriteByte(',')
		}
		s.writeJSON(buf)
	}
	buf.WriteByte(']')
	return buf.Bytes()
}

func (batch *SpanBatch) split() []*SpanBatch {
	if len(batch.Spans) < 2 {
		return nil
	}
	middle := len(batch.Spans) / 2
	return []*SpanBatch{{Spans: batch.Spans[0:middle]}, {Spans: batch.Spans[middle:]}}
}
