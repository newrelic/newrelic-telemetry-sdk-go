// Copyright 2019 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package telemetry

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"testing"
)

type testRequestBuilder struct {
	bodies []json.RawMessage
}

func (ts testRequestBuilder) makeBody() json.RawMessage {
	return ts.bodies[0]
}

func (ts testRequestBuilder) split() []requestsBuilder {
	reqs := ts.bodies[1:]
	if len(reqs) == 0 {
		return nil
	}
	return []requestsBuilder{
		testRequestBuilder{bodies: reqs},
		testRequestBuilder{bodies: reqs},
	}
}

func defaultTestNoopFunc(_ DataBatch) []RequestHeader {
	return []RequestHeader{}
}

func TestNewRequestsSplitSuccess(t *testing.T) {
	ts := testRequestBuilder{
		bodies: []json.RawMessage{
			json.RawMessage(`12345678901234567890`),
			json.RawMessage(`123456789012345`),
			json.RawMessage(`12345678901`),
			json.RawMessage(`123456789`),
		},
	}
	reqs, err := newRequestsInternal(ts, "", "", "", defaultTestNoopFunc, func(r request) bool {
		return len(r.UncompressedBody) >= 10
	})
	if err != nil {
		t.Error(err)
	}
	if len(reqs) != 8 {
		t.Error(len(reqs))
	}
}

func TestNewRequestsCantSplit(t *testing.T) {
	ts := testRequestBuilder{
		bodies: []json.RawMessage{
			json.RawMessage(`12345678901234567890`),
			json.RawMessage(`123456789012345`),
			json.RawMessage(`12345678901`),
		},
	}
	reqs, err := newRequestsInternal(ts, "", "", "", defaultTestNoopFunc, func(r request) bool {
		return len(r.UncompressedBody) >= 10
	})
	if err != errUnableToSplit {
		t.Error(err)
	}
	if len(reqs) != 0 {
		t.Error(len(reqs))
	}
}

func TestNewRequestsCanAddHeadersToRequest(t *testing.T) {
	c := Count{
		Name: "Count",
		Attributes: map[string]interface{}{
			"some-id": 1,
		},
	}
	s := Summary{
		Name: "Summary",
		Attributes: map[string]interface{}{
			"some-id": 2,
		},
	}
	g := Gauge{
		Name: "Gauge",
		Attributes: map[string]interface{}{
			"some-id": 3,
		},
	}
	mb := &metricBatch{
		Metrics: metricsArray{c, s, g},
	}
	reqs, err := newRequests(mb, "apiKey", defaultMetricURL, "userAgent", func(dataType DataBatch) []RequestHeader {
		rh := []RequestHeader{{Key: "names", Value: ""}, {Key: "ids", Value: ""}}
		for i := range dataType.GetDataTypes() {
			rh[0].Value = rh[0].Value + dataType.GetDataTypes()[i].GetName() + ","
			rh[1].Value = rh[1].Value + fmt.Sprint(dataType.GetDataTypes()[i].GetAttributes()["some-id"]) + ","
		}
		for i := range rh {
			rh[i].Value = rh[i].Value[:len(rh[i].Value)-1]
		}
		return rh
	})
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	if len(reqs) != 1 {
		t.Fatal(len(reqs))
	}

	req := reqs[0]
	names := req.Request.Header.Get("names")
	if names != "Count,Summary,Gauge" {
		t.Fatal(names)
	}

	ids := req.Request.Header.Get("ids")
	if ids != "1,2,3" {
		t.Fatal(ids)
	}
}

func randomJSON(numBytes int) json.RawMessage {
	digits := []byte{'1', '2', '3', '4', '5', '6', '7', '8', '9'}
	js := make([]byte, numBytes)
	for i := 0; i < len(js); i++ {
		js[i] = digits[rand.Intn(len(digits))]
	}
	return js
}

func TestLargeRequestNeedsSplit(t *testing.T) {
	js := randomJSON(4 * maxCompressedSizeBytes)
	reqs, err := newRequests(testRequestBuilder{bodies: []json.RawMessage{js}}, "apiKey", defaultMetricURL, "userAgent", defaultTestNoopFunc)
	if reqs != nil {
		t.Error(reqs)
	}
	if err != errUnableToSplit {
		t.Error(err)
	}
}

func TestLargeRequestNoSplit(t *testing.T) {
	js := randomJSON(maxCompressedSizeBytes / 2)
	reqs, err := newRequests(testRequestBuilder{bodies: []json.RawMessage{js}}, "apiKey", defaultMetricURL, "userAgent", defaultTestNoopFunc)
	if err != nil {
		t.Fatal(err)
	}
	if len(reqs) != 1 {
		t.Fatal(len(reqs))
	}
	req := reqs[0]
	if u := req.Request.URL.String(); u != defaultMetricURL {
		t.Error(u)
	}
	if len(req.UncompressedBody) != maxCompressedSizeBytes/2 {
		t.Error(len(req.UncompressedBody))
	}
}
