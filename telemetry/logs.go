// Copyright 2019 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package telemetry

import (
	"bytes"
	"time"

	"github.com/newrelic/newrelic-telemetry-sdk-go/internal"
)

const logTypeName string = "logs"

// Log is a log.
type Log struct {
	// Required Fields:
	//
	// Message is the log message.
	Message string

	// Recommended Fields:
	//
	// Timestamp of the log message.  If Timestamp is not set, it
	// will be assigned to time.Now() in Harvester.RecordLog.
	Timestamp time.Time

	// Additional Fields:
	//
	// Attributes is a map of user specified tags on this log message.  The map
	// values can be any of bool, number, or string.
	Attributes map[string]interface{}
}

func (l *Log) writeJSON(buf *bytes.Buffer) {
	w := internal.JSONFieldsWriter{Buf: buf}
	buf.WriteByte('{')

	w.StringField("message", l.Message)
	w.IntField("timestamp", l.Timestamp.UnixNano()/(1000*1000))

	w.AddKey("attributes")
	buf.WriteByte('{')
	ww := internal.JSONFieldsWriter{Buf: buf}

	internal.AddAttributes(&ww, l.Attributes)
	buf.WriteByte('}')

	buf.WriteByte('}')
}

type logCommonBlock struct {
	attributes *commonAttributes
}

// DataTypeKey returns the type of data contained in this MapEntry.
func (c *logCommonBlock) DataTypeKey() string {
	return "common"
}

// WriteBytes writes the json serialized bytes of the MapEntry to the buffer.
func (c *logCommonBlock) WriteDataEntry(buf *bytes.Buffer) *bytes.Buffer {
	buf.WriteByte('{')
	if c.attributes != nil {
		w := internal.JSONFieldsWriter{Buf: buf}
		w.AddKey(c.attributes.DataTypeKey())
		c.attributes.WriteDataEntry(buf)
	}
	buf.WriteByte('}')
	return buf
}

// LogCommonBlockOption is a function that can be used to configure a log common block
type LogCommonBlockOption func(*logCommonBlock) error

// NewLogCommonBlock creates a new MapEntry representing data common to all members of a group.
func NewLogCommonBlock(options ...LogCommonBlockOption) (MapEntry, error) {
	l := &logCommonBlock{}
	for _, option := range options {
		err := option(l)
		if err != nil {
			return nil, err
		}
	}
	return l, nil
}

// WithLogAttributes creates a LogCommonBlockOption to specify the common attributes of the common block.
// Invalid attributes will be detected and ignored
func WithLogAttributes(commonAttributes map[string]interface{}) LogCommonBlockOption {
	return func(b *logCommonBlock) error {
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

// LogGroup represents a group of log messages in the New Relic HTTP API.
type logGroup struct {
	Logs []Log
}

// DataTypeKey returns the type of data contained in this MapEntry.
func (group *logGroup) DataTypeKey() string {
	return logTypeName
}

// WriteDataEntry writes the json serialized bytes of the MapEntry to the buffer.
func (group *logGroup) WriteDataEntry(buf *bytes.Buffer) *bytes.Buffer {
	buf.WriteByte('[')
	for idx, s := range group.Logs {
		if idx > 0 {
			buf.WriteByte(',')
		}
		s.writeJSON(buf)
	}
	buf.WriteByte(']')
	return buf
}

func (group *logGroup) split() []splittablePayloadEntry {
	if len(group.Logs) < 2 {
		return nil
	}
	middle := len(group.Logs) / 2
	return []splittablePayloadEntry{&logGroup{Logs: group.Logs[0:middle]}, &logGroup{Logs: group.Logs[middle:]}}
}

// NewLogGroup creates a new MapEntry representing a group of logs in a batch.
func NewLogGroup(logs []Log) MapEntry {
	return &logGroup{Logs: logs}
}
