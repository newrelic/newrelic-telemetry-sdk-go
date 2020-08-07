// Copyright 2019 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0
// +build benchmark

package telemetry

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"
)

func BenchmarkSpansJSON(b *testing.B) {
	// This benchmark tests the overhead of turning spans into JSON.
	batch := &spanBatch{}
	numSpans := 10 * 1000
	tm := time.Date(2014, time.November, 28, 1, 1, 0, 0, time.UTC)

	for i := 0; i < numSpans; i++ {
		batch.Spans = append(batch.Spans, Span{
			ID:             "myid",
			TraceID:        "mytraceid",
			Name:           "myname",
			ParentID:       "myparent",
			Timestamp:      tm,
			Duration:       2 * time.Second,
			ServiceName:    "myentity",
			AttributesJSON: json.RawMessage(`{"zip":"zap","zop":123}`),
		})
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		buf := &bytes.Buffer{}
		batch.writeJSON(buf)
		if bts := buf.Bytes(); nil == bts || len(bts) == 0 {
			b.Fatal(string(bts))
		}
	}
}
