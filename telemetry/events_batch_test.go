// Copyright 2019 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0
// +build unit

package telemetry

import (
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/newrelic/newrelic-telemetry-sdk-go/internal"
)

func testEventBatchJSON(t testing.TB, batch *eventBatch, expect string) {
	if th, ok := t.(interface{ Helper() }); ok {
		th.Helper()
	}
	reqs, err := newRequests(batch, "apiKey", defaultEventURL, "userAgent")
	require.NoError(t, err)
	require.Equal(t, 1, len(reqs))

	req := reqs[0]
	assert.Equal(t, compactJSONString(expect), string(req.UncompressedBody))

	body, err := ioutil.ReadAll(req.Request.Body)
	req.Request.Body.Close()
	require.NoError(t, err)

	assert.Equal(t, req.compressedBodyLength, len(body))

	uncompressed, err := internal.Uncompress(body)
	require.NoError(t, err)
	assert.Equal(t, string(req.UncompressedBody), string(uncompressed))
}

func TestEventsPayloadSplit(t *testing.T) {
	t.Parallel()

	// test len 0
	ev := &eventBatch{}
	split := ev.split()
	assert.Nil(t, split)

	// test len 1
	ev = &eventBatch{Events: []Event{{EventType: "a"}}}
	split = ev.split()
	assert.Nil(t, split)

	// test len 2
	ev = &eventBatch{Events: []Event{{EventType: "a"}, {EventType: "b"}}}
	split = ev.split()
	assert.Equal(t, 2, len(split))

	testEventBatchJSON(t, split[0].(*eventBatch), `[{"eventType":"a","timestamp":-6795364578871}]`)
	testEventBatchJSON(t, split[1].(*eventBatch), `[{"eventType":"b","timestamp":-6795364578871}]`)

	// test len 3
	ev = &eventBatch{Events: []Event{{EventType: "a"}, {EventType: "b"}, {EventType: "c"}}}
	split = ev.split()
	assert.Equal(t, 2, len(split))
	testEventBatchJSON(t, split[0].(*eventBatch), `[{"eventType":"a","timestamp":-6795364578871}]`)
	testEventBatchJSON(t, split[1].(*eventBatch), `[{"eventType":"b","timestamp":-6795364578871},{"eventType":"c","timestamp":-6795364578871}]`)
}

func TestEventsJSON(t *testing.T) {
	t.Parallel()

	batch := &eventBatch{Events: []Event{
		{}, // Empty
		{ // with everything
			EventType:  "testEvent",
			Timestamp:  testTimestamp,
			Attributes: map[string]interface{}{"zip": "zap"},
		},
	}}

	testEventBatchJSON(t, batch, `[
		{
		  "eventType":"",
			"timestamp":-6795364578871
		},
		{
			"eventType":"testEvent",
			"timestamp":`+testTimeString+`,
			"zip":"zap"
		}
	]`)
}
