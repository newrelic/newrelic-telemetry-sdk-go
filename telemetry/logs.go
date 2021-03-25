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
	Attributes *commonAttributes
}

// Type returns the type of data contained in this MapEntry.
func (c *logCommonBlock) Type() string {
	return "common"
}

// WriteBytes writes the json serialized bytes of the MapEntry to the buffer.
func (c *logCommonBlock) WriteBytes(buf *bytes.Buffer) {
	buf.WriteByte('{')
	if c.Attributes != nil {
		w := internal.JSONFieldsWriter{Buf: buf}
		w.AddKey(c.Attributes.Type())
		c.Attributes.WriteBytes(buf)
	}
	buf.WriteByte('}')
}

// LogBatch represents a single batch of log messages to report to New Relic.
type LogBatch struct {
	Logs []Log
}

// Type returns the type of data contained in this MapEntry.
func (batch *LogBatch) Type() string {
	return logTypeName
}

// WriteBytes writes the json serialized bytes of the MapEntry to the buffer.
func (batch *LogBatch) WriteBytes(buf *bytes.Buffer) {
	buf.WriteByte('[')
	for idx, s := range batch.Logs {
		if idx > 0 {
			buf.WriteByte(',')
		}
		s.writeJSON(buf)
	}
	buf.WriteByte(']')
}

func (batch *LogBatch) split() []splittablePayloadEntry {
	if len(batch.Logs) < 2 {
		return nil
	}
	middle := len(batch.Logs) / 2
	return []splittablePayloadEntry{&LogBatch{Logs: batch.Logs[0:middle]}, &LogBatch{Logs: batch.Logs[middle:]}}
}
