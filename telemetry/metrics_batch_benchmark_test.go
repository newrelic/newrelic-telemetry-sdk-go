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

func BenchmarkMetricsJSON(b *testing.B) {
	// This benchmark tests the overhead of turning metrics into JSON.
	batch := &metricBatch{
		AttributesJSON: json.RawMessage(`{"zip": "zap"}`),
	}
	numMetrics := 10 * 1000
	start := time.Date(2014, time.November, 28, 1, 1, 0, 0, time.UTC)

	for i := 0; i < numMetrics/3; i++ {
		batch.Metrics = append(batch.Metrics, Summary{
			Name:       "mySummary",
			Attributes: map[string]interface{}{"attribute": "string"},
			Count:      3,
			Sum:        15,
			Min:        4,
			Max:        6,
			Timestamp:  start,
			Interval:   5 * time.Second,
		})
		batch.Metrics = append(batch.Metrics, Gauge{
			Name:       "myGauge",
			Attributes: map[string]interface{}{"attribute": true},
			Value:      12.3,
			Timestamp:  start,
		})
		batch.Metrics = append(batch.Metrics, Count{
			Name:       "myCount",
			Attributes: map[string]interface{}{"attribute": 123},
			Value:      100,
			Timestamp:  start,
			Interval:   5 * time.Second,
		})
	}

	b.ResetTimer()
	b.ReportAllocs()

	estimate := len(batch.Metrics) * 256
	for i := 0; i < b.N; i++ {
		buf := bytes.NewBuffer(make([]byte, 0, estimate))
		batch.writeJSON(buf)
		bts := buf.Bytes()
		if len(bts) == 0 {
			b.Fatal(string(bts))
		}
	}
}
