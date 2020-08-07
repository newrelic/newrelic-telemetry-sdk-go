// Copyright 2019 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0
// +build unit

package telemetry

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func testHarvesterSpans(t testing.TB, h *Harvester, expect string) {
	reqs := h.swapOutSpans()
	if nil == reqs {
		if expect != "null" {
			t.Error("nil spans", expect)
		}
		return
	}
	if len(reqs) != 1 {
		t.Fatal(reqs)
	}
	if u := reqs[0].Request.URL.String(); u != defaultSpanURL {
		t.Fatal(u)
	}
	js := reqs[0].UncompressedBody
	actual := string(js)
	if th, ok := t.(interface{ Helper() }); ok {
		th.Helper()
	}
	compactExpect := compactJSONString(expect)
	if compactExpect != actual {
		t.Errorf("\nexpect=%s\nactual=%s\n", compactExpect, actual)
	}
}

func TestSpan(t *testing.T) {
	tm := time.Date(2014, time.November, 28, 1, 1, 0, 0, time.UTC)
	h, err := NewHarvester(configTesting)
	assert.NoError(t, err)
	assert.NotNil(t, h)

	err = h.RecordSpan(Span{
		ID:          "myid",
		TraceID:     "mytraceid",
		Name:        "myname",
		ParentID:    "myparent",
		Timestamp:   tm,
		Duration:    2 * time.Second,
		ServiceName: "myentity",
		Attributes: map[string]interface{}{
			"zip": "zap",
		},
	})
	assert.NoError(t, err)

	expect := `[{"common":{},"spans":[{
		"id":"myid",
		"trace.id":"mytraceid",
		"timestamp":1417136460000,
		"attributes": {
			"name":"myname",
			"parent.id":"myparent",
			"duration.ms":2000,
			"service.name":"myentity",
			"zip":"zap"
		}
	}]}]`
	testHarvesterSpans(t, h, expect)
}

func TestSpanInvalidAttribute(t *testing.T) {
	tm := time.Date(2014, time.November, 28, 1, 1, 0, 0, time.UTC)
	h, err := NewHarvester(configTesting)
	assert.NoError(t, err)
	assert.NotNil(t, h)

	err = h.RecordSpan(Span{
		ID:          "myid",
		TraceID:     "mytraceid",
		Name:        "myname",
		ParentID:    "myparent",
		Timestamp:   tm,
		Duration:    2 * time.Second,
		ServiceName: "myentity",
		Attributes: map[string]interface{}{
			"weird-things-get-turned-to-strings": struct{}{},
			"nil-gets-removed":                   nil,
		},
	})
	assert.NoError(t, err)

	expect := `[{"common":{},"spans":[{
		"id":"myid",
		"trace.id":"mytraceid",
		"timestamp":1417136460000,
		"attributes": {
			"name":"myname",
			"parent.id":"myparent",
			"duration.ms":2000,
			"service.name":"myentity",
			"weird-things-get-turned-to-strings":"struct {}"
		}
	}]}]`
	testHarvesterSpans(t, h, expect)
}

func TestRecordSpanNilHarvester(t *testing.T) {
	tm := time.Date(2014, time.November, 28, 1, 1, 0, 0, time.UTC)
	var h *Harvester
	err := h.RecordSpan(Span{
		ID:          "myid",
		TraceID:     "mytraceid",
		Name:        "myname",
		ParentID:    "myparent",
		Timestamp:   tm,
		Duration:    2 * time.Second,
		ServiceName: "myentity",
		Attributes: map[string]interface{}{
			"zip": "zap",
			"zop": 123,
		},
	})
	if err != nil {
		t.Error(err)
	}
}
