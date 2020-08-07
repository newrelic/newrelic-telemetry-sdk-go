// Copyright 2019 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0
// +build unit

package telemetry

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testHarvesterEvents(t testing.TB, h *Harvester, expect string) {
	reqs := h.swapOutEvents()
	if expect != "null" {
		require.NotNil(t, reqs)
	}
	require.Equal(t, 1, len(reqs))
	require.Equal(t, defaultEventURL, reqs[0].Request.URL.String())

	js := reqs[0].UncompressedBody
	actual := string(js)
	if th, ok := t.(interface{ Helper() }); ok {
		th.Helper()
	}
	assert.Equal(t, compactJSONString(expect), actual)
}

func TestEvent(t *testing.T) {
	t.Parallel()

	tm := time.Date(2014, time.November, 28, 1, 1, 0, 0, time.UTC)
	h, err := NewHarvester(configTesting)
	assert.NoError(t, err)
	assert.NotNil(t, h)

	err = h.RecordEvent(Event{
		EventType: "testEvent",
		Timestamp: tm,
		Attributes: map[string]interface{}{
			"zip": "zap",
		},
	})
	assert.NoError(t, err)

	expect := `[{
		"eventType":"testEvent",
		"timestamp":1417136460000,
		"zip":"zap"
	}]`

	testHarvesterEvents(t, h, expect)
}

func TestEventInvalidAttribute(t *testing.T) {
	t.Parallel()

	tm := time.Date(2014, time.November, 28, 1, 1, 0, 0, time.UTC)
	h, err := NewHarvester(configTesting)
	assert.NoError(t, err)
	assert.NotNil(t, h)

	err = h.RecordEvent(Event{
		EventType: "testEvent",
		Timestamp: tm,
		Attributes: map[string]interface{}{
			"weird-things-get-turned-to-strings": struct{}{},
			"nil-gets-removed":                   nil,
		},
	})
	assert.NoError(t, err)

	expect := `[{
		"eventType":"testEvent",
		"timestamp":1417136460000,
		"weird-things-get-turned-to-strings":"struct {}"
	}]`

	testHarvesterEvents(t, h, expect)
}

func TestRecordEventZeroTime(t *testing.T) {
	t.Parallel()

	h, err := NewHarvester(configTesting)
	assert.NoError(t, err)
	assert.NotNil(t, h)

	err = h.RecordEvent(Event{
		EventType: "testEvent",
		Attributes: map[string]interface{}{
			"zip": "zap",
			"zop": 123,
		},
	})

	assert.NoError(t, err)
}
func TestRecordEventEmptyType(t *testing.T) {
	t.Parallel()

	tm := time.Date(2014, time.November, 28, 1, 1, 0, 0, time.UTC)
	h, err := NewHarvester(configTesting)
	assert.NoError(t, err)
	assert.NotNil(t, h)

	err = h.RecordEvent(Event{
		Timestamp: tm,
		Attributes: map[string]interface{}{
			"zip": "zap",
			"zop": 123,
		},
	})

	assert.Error(t, err)
	assert.Equal(t, errEventTypeUnset, err)
}

func TestRecordEventNilHarvester(t *testing.T) {
	t.Parallel()

	tm := time.Date(2014, time.November, 28, 1, 1, 0, 0, time.UTC)
	var h *Harvester
	err := h.RecordEvent(Event{
		EventType: "testEvent",
		Timestamp: tm,
		Attributes: map[string]interface{}{
			"zip": "zap",
			"zop": 123,
		},
	})

	assert.NoError(t, err)
}
