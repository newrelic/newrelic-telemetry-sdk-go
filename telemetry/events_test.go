// Copyright 2019 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package telemetry

import (
	"io/ioutil"
	"testing"
	"time"

	"github.com/newrelic/newrelic-telemetry-sdk-go/internal"
)

func testHarvesterEvents(t testing.TB, h *Harvester, expect string) {
	reqs := h.swapOutEvents()
	if nil == reqs {
		if expect != "null" {
			t.Error("nil spans", expect)
		}
		return
	}

	if len(reqs) != 1 {
		t.Fatal(reqs)
	}
	if u := reqs[0].URL.String(); u != defaultEventURL {
		t.Fatal(u)
	}

	bodyReader, _ := reqs[0].GetBody()
	compressedBytes, _ := ioutil.ReadAll(bodyReader)
	js, _ := internal.Uncompress(compressedBytes)
	actual := string(js)
	if th, ok := t.(interface{ Helper() }); ok {
		th.Helper()
	}
	compactExpect := compactJSONString(expect)
	if compactExpect != actual {
		t.Errorf("\nexpect=%s\nactual=%s\n", compactExpect, actual)
	}
}

func TestEvent(t *testing.T) {
	t.Parallel()

	tm := time.Date(2014, time.November, 28, 1, 1, 0, 0, time.UTC)
	h, err := NewHarvester(configTesting)
	if nil == h || err != nil {
		t.Fatal(h, err)
	}

	err = h.RecordEvent(Event{
		EventType: "testEvent",
		Timestamp: tm,
		Attributes: map[string]interface{}{
			"zip": "zap",
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	expect := `[{"events":[{
		"eventType":"testEvent",
		"timestamp":1417136460000,
		"zip":"zap"
	}]}]`

	testHarvesterEvents(t, h, expect)
}

func TestEventInvalidAttribute(t *testing.T) {
	t.Parallel()

	tm := time.Date(2014, time.November, 28, 1, 1, 0, 0, time.UTC)
	h, err := NewHarvester(configTesting)
	if nil == h || err != nil {
		t.Fatal(h, err)
	}

	err = h.RecordEvent(Event{
		EventType: "testEvent",
		Timestamp: tm,
		Attributes: map[string]interface{}{
			"weird-things-get-turned-to-strings": struct{}{},
			"nil-gets-removed":                   nil,
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	expect := `[{"events":[{
		"eventType":"testEvent",
		"timestamp":1417136460000,
		"weird-things-get-turned-to-strings":"struct {}"
	}]}]`

	testHarvesterEvents(t, h, expect)
}

func TestRecordEventZeroTime(t *testing.T) {
	t.Parallel()

	h, err := NewHarvester(configTesting)
	if nil == h || err != nil {
		t.Fatal(h, err)
	}

	err = h.RecordEvent(Event{
		EventType: "testEvent",
		Attributes: map[string]interface{}{
			"zip": "zap",
			"zop": 123,
		},
	})

	if err != nil {
		t.Fatal(err)
	}
}
func TestRecordEventEmptyType(t *testing.T) {
	t.Parallel()

	tm := time.Date(2014, time.November, 28, 1, 1, 0, 0, time.UTC)
	h, err := NewHarvester(configTesting)
	if nil == h || err != nil {
		t.Fatal(h, err)
	}

	err = h.RecordEvent(Event{
		Timestamp: tm,
		Attributes: map[string]interface{}{
			"zip": "zap",
			"zop": 123,
		},
	})

	if err != errEventTypeUnset {
		t.Fatal(h, err)
	}
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

	if err != nil {
		t.Fatal(err)
	}
}
