// Copyright 2019 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package telemetry

import (
	"testing"
	"time"
)

func TestMetricPayload(t *testing.T) {
	// Test that a metric payload with timestamp, duration, and common
	// attributes correctly marshals into JSON.
	now := time.Date(2014, time.November, 28, 1, 1, 0, 0, time.UTC)
	h, _ := NewHarvester(ConfigCommonAttributes(map[string]interface{}{"zop": "zup"}), configTesting)
	// Use a single metric to avoid sorting.
	h.RecordMetric(Gauge{
		Name:       "metric",
		Attributes: map[string]interface{}{"zip": "zap"},
		Timestamp:  now,
		Value:      1.0,
	})
	h.lastHarvest = now
	end := h.lastHarvest.Add(5 * time.Second)
	reqs := h.swapOutMetrics(end)
	if len(reqs) != 1 {
		t.Fatal(reqs)
	}
	js := reqs[0].UncompressedBody
	actual := string(js)
	expect := `[{
		"common":{
			"timestamp":1417136460000,
			"interval.ms":5000,
			"attributes":{"zop":"zup"}
		},
		"metrics":[
			{"name":"metric","type":"gauge","value":1,"timestamp":1417136460000,"attributes":{"zip":"zap"}}
		]
	}]`
	compactExpect := compactJSONString(expect)
	if compactExpect != actual {
		t.Errorf("\nexpect=%s\nactual=%s\n", compactExpect, actual)
	}
}

func TestVetAttributes(t *testing.T) {
	testcases := []struct {
		Input interface{}
		Valid bool
	}{
		// Valid attribute types.
		{Input: "string value", Valid: true},
		{Input: true, Valid: true},
		{Input: uint8(0), Valid: true},
		{Input: uint16(0), Valid: true},
		{Input: uint32(0), Valid: true},
		{Input: uint64(0), Valid: true},
		{Input: int8(0), Valid: true},
		{Input: int16(0), Valid: true},
		{Input: int32(0), Valid: true},
		{Input: int64(0), Valid: true},
		{Input: float32(0), Valid: true},
		{Input: float64(0), Valid: true},
		{Input: uint(0), Valid: true},
		{Input: int(0), Valid: true},
		{Input: uintptr(0), Valid: true},
		// Invalid attribute types.
		{Input: nil, Valid: false},
		{Input: struct{}{}, Valid: false},
		{Input: &struct{}{}, Valid: false},
		{Input: []int{1, 2, 3}, Valid: false},
	}

	for idx, tc := range testcases {
		key := "input"
		input := map[string]interface{}{
			key: tc.Input,
		}
		var errorLogged map[string]interface{}
		output := vetAttributes(input, func(e map[string]interface{}) {
			errorLogged = e
		})
		// Test the the input map has not been modified.
		if len(input) != 1 {
			t.Error("input map modified", input)
		}
		if tc.Valid {
			if len(output) != 1 {
				t.Error(idx, tc.Input, output)
			}
			if _, ok := output[key]; !ok {
				t.Error(idx, tc.Input, output)
			}
			if errorLogged != nil {
				t.Error(idx, "unexpected error present")
			}
		} else {
			if errorLogged == nil {
				t.Error(idx, "expected error missing")
			}
			if len(output) != 0 {
				t.Error(idx, tc.Input, output)
			}
		}
	}
}
