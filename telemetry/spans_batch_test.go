// Copyright 2019 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package telemetry

import (
	"io/ioutil"
	"testing"
	"time"

	"github.com/newrelic/newrelic-telemetry-sdk-go/internal"
)

func testSpanBatchJSON(t testing.TB, batches []Batch, expect string) {
	if th, ok := t.(interface{ Helper() }); ok {
		th.Helper()
	}
	factory, _ := NewSpanRequestFactory(WithNoDefaultKey())
	reqs, err := newRequests(batches, factory)
	if nil != err {
		t.Fatal(err)
	}
	if len(reqs) != 1 {
		t.Fatal(reqs)
	}
	req := reqs[0]
	bodyReader, _ := req.GetBody()
	compressedBytes, _ := ioutil.ReadAll(bodyReader)
	uncompressedBytes, err := internal.Uncompress(compressedBytes)
	if err != nil {
		t.Fatal("unable to uncompress body", err)
	}
	actual := string(uncompressedBytes)
	compact := compactJSONString(expect)
	if actual != compact {
		t.Errorf("\nexpect=%s\nactual=%s\n", compact, actual)
	}

	body, err := ioutil.ReadAll(req.Body)
	req.Body.Close()
	if err != nil {
		t.Fatal("unable to read body", err)
	}
	if len(body) != int(req.ContentLength) {
		t.Error("compressed body length mismatch",
			len(body), req.ContentLength)
	}
}

func TestSpansPayloadSplit(t *testing.T) {
	// test len 0
	sp := &SpanBatch{}
	split := sp.split()
	if split != nil {
		t.Error(split)
	}

	// test len 1
	sp = &SpanBatch{Spans: []Span{{Name: "a"}}}
	split = sp.split()
	if split != nil {
		t.Error(split)
	}

	// test len 2
	sp = &SpanBatch{Spans: []Span{{Name: "a"}, {Name: "b"}}}
	split = sp.split()
	if len(split) != 2 {
		t.Error("split into incorrect number of slices", len(split))
	}
	testSpanBatchJSON(t, []Batch{{split[0]}}, `[{"spans":[{"id":"","trace.id":"","timestamp":-6795364578871,"attributes":{"name":"a"}}]}]`)
	testSpanBatchJSON(t, []Batch{{split[1]}}, `[{"spans":[{"id":"","trace.id":"","timestamp":-6795364578871,"attributes":{"name":"b"}}]}]`)

	// test len 3
	sp = &SpanBatch{Spans: []Span{{Name: "a"}, {Name: "b"}, {Name: "c"}}}
	split = sp.split()
	if len(split) != 2 {
		t.Error("split into incorrect number of slices", len(split))
	}
	testSpanBatchJSON(t, []Batch{{split[0]}}, `[{"spans":[{"id":"","trace.id":"","timestamp":-6795364578871,"attributes":{"name":"a"}}]}]`)
	testSpanBatchJSON(t, []Batch{{split[1]}}, `[{"spans":[{"id":"","trace.id":"","timestamp":-6795364578871,"attributes":{"name":"b"}},{"id":"","trace.id":"","timestamp":-6795364578871,"attributes":{"name":"c"}}]}]`)
}

func TestSpansJSON(t *testing.T) {
	batch := &SpanBatch{Spans: []Span{
		{}, // Empty span
		{ // Span with everything
			ID:          "myid",
			TraceID:     "mytraceid",
			Name:        "myname",
			ParentID:    "myparentid",
			Timestamp:   time.Date(2014, time.November, 28, 1, 1, 0, 0, time.UTC),
			Duration:    2 * time.Second,
			ServiceName: "myentity",
			Attributes:  map[string]interface{}{"zip": "zap"},
		},
	}}
	testSpanBatchJSON(t, []Batch{{batch}}, `[{"spans":[
		{
			"id":"",
			"trace.id":"",
			"timestamp":-6795364578871,
			"attributes": {
			}
		},
		{
			"id":"myid",
			"trace.id":"mytraceid",
			"timestamp":1417136460000,
			"attributes": {
				"name":"myname",
				"parent.id":"myparentid",
				"duration.ms":2000,
				"service.name":"myentity",
				"zip":"zap"
			}
		}
	]}]`)
}

func TestSpansJSONWithCommonAttributesJSON(t *testing.T) {
	commonBlock, err := NewSpanCommonBlock(WithSpanAttributes(map[string]interface{}{"zup": "wup"}))
	if err != nil {
		t.Fail()
	}

	batch1 := &SpanBatch{
		Spans: []Span{
			{
				ID:          "myid1",
				TraceID:     "mytraceid1",
				Name:        "myname",
				ParentID:    "myparentid",
				Timestamp:   time.Date(2014, time.November, 28, 1, 1, 0, 0, time.UTC),
				Duration:    2 * time.Second,
				ServiceName: "myentity",
				Attributes:  map[string]interface{}{"zip": "zap"},
			},
		}}
	batch2 := &SpanBatch{
		Spans: []Span{
			{
				ID:        "myid2",
				TraceID:   "mytraceid2",
				Timestamp: time.Date(2014, time.November, 28, 1, 1, 0, 0, time.UTC),
			},
		},
	}
	testSpanBatchJSON(t, []Batch{{commonBlock, batch1}, {batch2}}, `[
		{
			"common": {
				"attributes": {
					"zup":"wup"
				}
			},
			"spans":[
				{
					"id":"myid1",
					"trace.id":"mytraceid1",
					"timestamp":1417136460000,
					"attributes": {
						"name":"myname",
						"parent.id":"myparentid",
						"duration.ms":2000,
						"service.name":"myentity",
						"zip":"zap"
					}
				}
			]
		},
		{
			"spans":[
				{
					"id":"myid2",
					"trace.id":"mytraceid2",
					"timestamp":1417136460000,
					"attributes": {}
				}
			]
		}
	]`)
}

func TestSpanBatchSplittable(t *testing.T) {
	batch := &SpanBatch{
		Spans: []Span{
			{
				ID:        "myid2",
				TraceID:   "mytraceid2",
				Timestamp: time.Date(2014, time.November, 28, 1, 1, 0, 0, time.UTC),
			},
		},
	}
	_ = splittablePayloadEntry(batch)
}
