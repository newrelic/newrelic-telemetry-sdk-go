// Copyright 2019 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package telemetry

import (
	"encoding/json"
	"io/ioutil"
	"testing"
	"time"

	"github.com/newrelic/newrelic-telemetry-sdk-go/internal"
)

func testLogBatchJSON(t testing.TB, batches [][]PayloadEntry, expect string) {
	if th, ok := t.(interface{ Helper() }); ok {
		th.Helper()
	}
	factory, _ := NewLogRequestFactory(WithNoDefaultKey())
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
	js := string(uncompressedBytes)
	actual := string(js)
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

func TestLogsPayloadSplit(t *testing.T) {
	// test len 0
	sp := &LogBatch{}
	split := sp.split()
	if split != nil {
		t.Error(split)
	}

	// test len 1
	sp = &LogBatch{Logs: []Log{{Message: "a"}}}
	split = sp.split()
	if split != nil {
		t.Error(split)
	}

	// test len 2
	sp = &LogBatch{Logs: []Log{{Message: "a"}, {Message: "b"}}}
	split = sp.split()
	if len(split) != 2 {
		t.Error("split into incorrect number of slices", len(split))
	}
	testLogBatchJSON(t, [][]PayloadEntry{{split[0]}}, `[{"logs":[{"message":"a","timestamp":-6795364578871,"attributes":{}}]}]`)
	testLogBatchJSON(t, [][]PayloadEntry{{split[1]}}, `[{"logs":[{"message":"b","timestamp":-6795364578871,"attributes":{}}]}]`)

	// test len 3
	sp = &LogBatch{Logs: []Log{{Message: "a"}, {Message: "b"}, {Message: "c"}}}
	split = sp.split()
	if len(split) != 2 {
		t.Error("split into incorrect number of slices", len(split))
	}
	testLogBatchJSON(t, [][]PayloadEntry{{split[0]}}, `[{"logs":[{"message":"a","timestamp":-6795364578871,"attributes":{}}]}]`)
	testLogBatchJSON(t, [][]PayloadEntry{{split[1]}}, `[{"logs":[{"message":"b","timestamp":-6795364578871,"attributes":{}},{"message":"c","timestamp":-6795364578871,"attributes":{}}]}]`)
}

func TestLogsJSON(t *testing.T) {
	batch := &LogBatch{Logs: []Log{
		{}, // Empty log
		{ // Log with everything
			Message:    "This is a log message.",
			Timestamp:  time.Date(2014, time.November, 28, 1, 1, 0, 0, time.UTC),
			Attributes: map[string]interface{}{"zip": "zap"},
		},
	}}
	testLogBatchJSON(t, [][]PayloadEntry{{batch}}, `[{"logs":[
		{
			"message":"",
			"timestamp":-6795364578871,
			"attributes": {
			}
		},
		{
			"message":"This is a log message.",
			"timestamp":1417136460000,
			"attributes": {
				"zip":"zap"
			}
		}
	]}]`)
}

func TestLogsJSONWithCommonAttributesJSON(t *testing.T) {
	commonBlock := &logCommonBlock{
		Attributes: &commonAttributes{RawJSON: json.RawMessage(`{"zup":"wup"}`)},
	}

	batch1 := &LogBatch{
		Logs: []Log{
			{
				Message:    "This is a log message.",
				Timestamp:  time.Date(2014, time.November, 28, 1, 1, 0, 0, time.UTC),
				Attributes: map[string]interface{}{"zip": "zap"},
			},
		}}
	batch2 := &LogBatch{
		Logs: []Log{
			{
				Message:   "Another log message.",
				Timestamp: time.Date(2014, time.November, 28, 1, 1, 0, 0, time.UTC),
			},
		},
	}
	testLogBatchJSON(t, [][]PayloadEntry{{commonBlock, batch1}, {batch2}}, `[
		{
			"common": {
				"attributes": {
					"zup":"wup"
				}
			},
			"logs":[
				{
					"message":"This is a log message.",
					"timestamp":1417136460000,
					"attributes": {
						"zip":"zap"
					}
				}
			]
		},
		{
			"logs":[
				{
					"message":"Another log message.",
					"timestamp":1417136460000,
					"attributes": {}
				}
			]
		}
	]`)
}
