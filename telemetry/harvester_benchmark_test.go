// Copyright 2019 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0
// +build unit

package telemetry

import (
	"context"
	"testing"
)

func benchmarkRetryBodyN(b *testing.B, n int) {
	// Disable backoff delay.
	oBOSS := backoffSequenceSeconds
	backoffSequenceSeconds = make([]int, n+1)

	count := Count{}
	ctx := context.Background()
	h, _ := NewHarvester(func(cfg *Config) {
		cfg.HarvestPeriod = 0
		cfg.APIKey = "APIKey"
		cfg.Client.Transport = multiAttemptRoundTripper(n)
	})

	b.ReportAllocs()
	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		h.RecordMetric(count)
		h.HarvestNow(ctx)
	}

	b.StopTimer()
	backoffSequenceSeconds = oBOSS
}

// Baseline for the rest. This does not retry.
func BenchmarkRetryBody0(b *testing.B) { benchmarkRetryBodyN(b, 0) }
func BenchmarkRetryBody1(b *testing.B) { benchmarkRetryBodyN(b, 1) }
func BenchmarkRetryBody2(b *testing.B) { benchmarkRetryBodyN(b, 2) }
func BenchmarkRetryBody4(b *testing.B) { benchmarkRetryBodyN(b, 4) }
func BenchmarkRetryBody8(b *testing.B) { benchmarkRetryBodyN(b, 8) }
