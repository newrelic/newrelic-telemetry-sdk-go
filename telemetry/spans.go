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

	if s.Name != "" {
		ww.StringField("name", s.Name)
	}
	if s.ParentID != "" {
		ww.StringField("parent.id", s.ParentID)
	}
	if s.Duration != 0 {
		ww.FloatField("duration.ms", s.Duration.Seconds()*1000.0)
	}
	if s.ServiceName != "" {
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

// spanCommonBlock represents the shared elements of a SpanGroup.
type spanCommonBlock struct {
	attributes *commonAttributes
}

// DataTypeKey returns the type of data contained in this MapEntry.
func (c *spanCommonBlock) DataTypeKey() string {
	return "common"
}

// WriteDataEntry writes the json serialized bytes of the MapEntry to the buffer.
func (c *spanCommonBlock) WriteDataEntry(buf *bytes.Buffer) *bytes.Buffer {
	buf.WriteByte('{')
	if c.attributes != nil {
		w := internal.JSONFieldsWriter{Buf: buf}
		w.AddKey(c.attributes.DataTypeKey())
		c.attributes.WriteDataEntry(buf)
	}
	buf.WriteByte('}')
	return buf
}

// SpanCommonBlockOption is a function that can be used to configure a span common block
type SpanCommonBlockOption func(scb *spanCommonBlock) error

// NewSpanCommonBlock creates a new MapEntry representing data common to all spans in a group.
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

// WithSpanAttributes creates a SpanCommonBlockOption to specify the common attributes of the common block.
// Invalid attributes will be detected and ignored
func WithSpanAttributes(commonAttributes map[string]interface{}) SpanCommonBlockOption {
	return func(scb *spanCommonBlock) error {
		validCommonAttr, err := newCommonAttributes(commonAttributes)
		if err != nil {
			// Ignore any error with invalid attributes
			if _, ok := err.(errInvalidAttributes); !ok {
				return err
			}
		}
		scb.attributes = validCommonAttr
		return nil
	}
}

// SpanGroup represents a grouping of spans in a payload to New Relic.
type spanGroup struct {
	Spans []Span
}

// DataTypeKey returns the type of data contained in this MapEntry.
func (group *spanGroup) DataTypeKey() string {
	return spanTypeName
}

// WriteDataEntry writes the json serialized bytes of the MapEntry to the buffer.
func (group *spanGroup) WriteDataEntry(buf *bytes.Buffer) *bytes.Buffer {
	buf.WriteByte('[')
	for idx, s := range group.Spans {
		if idx > 0 {
			buf.WriteByte(',')
		}
		s.writeJSON(buf)
	}
	buf.WriteByte(']')
	return buf
}

func (group *spanGroup) split() []splittablePayloadEntry {
	if len(group.Spans) < 2 {
		return nil
	}
	middle := len(group.Spans) / 2
	return []splittablePayloadEntry{&spanGroup{Spans: group.Spans[0:middle]}, &spanGroup{Spans: group.Spans[middle:]}}
}

// NewSpanGroup creates a new MapEntry representing a group of spans in a batch.
func NewSpanGroup(spans []Span) MapEntry {
	return &spanGroup{Spans: spans}
}
