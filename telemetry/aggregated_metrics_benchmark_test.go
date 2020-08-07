// Copyright 2019 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0
// +build benchmark

package telemetry

import (
	"testing"
)

func BenchmarkAggregatedMetric(b *testing.B) {
	// This benchmark tests creating and aggregating a summary.
	h, _ := NewHarvester(configTesting)
	attributes := map[string]interface{}{"zip": "zap", "zop": 123}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		summary := h.MetricAggregator().Summary("mySummary", attributes)
		summary.Record(12.3)
		if nil == summary {
			b.Fatal("nil summary")
		}
	}
}
