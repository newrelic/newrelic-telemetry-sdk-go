// Copyright 2019 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package telemetry

import (
	"bytes"
	"encoding/json"
	"github.com/newrelic/newrelic-telemetry-sdk-go/internal"
	"io/ioutil"
	"math/rand"
	"net/http"
	"testing"
)

type testUnsplittablePayloadEntry struct {
	rawData json.RawMessage
}

func (p *testUnsplittablePayloadEntry) DataTypeKey() string {
	return "testUnsplittable"
}

func (p *testUnsplittablePayloadEntry) WriteDataEntry(buf *bytes.Buffer) *bytes.Buffer {
	buf.Write(p.rawData)
	return buf
}

type testSplittablePayloadEntry struct {
	rawData       json.RawMessage
	splitPayloads []*testSplittablePayloadEntry
}

func (p *testSplittablePayloadEntry) DataTypeKey() string {
	return "testSplittable"
}

func (p *testSplittablePayloadEntry) WriteDataEntry(buf *bytes.Buffer) *bytes.Buffer {
	buf.Write(p.rawData)
	return buf
}

func (p *testSplittablePayloadEntry) split() []splittablePayloadEntry {
	if nil == p.splitPayloads {
		return nil
	}

	splitPayloads := make([]splittablePayloadEntry, len(p.splitPayloads))
	for i := 0; i < len(p.splitPayloads); i++ {
		splitPayloads[i] = p.splitPayloads[i]
	}
	return splitPayloads
}

func TestNewRequestNoSplitNeeded(t *testing.T) {
	testPayload := testUnsplittablePayloadEntry{rawData: json.RawMessage(`123456789`)}
	entries := []MapEntry{&testPayload}
	reqs, err := newRequestsInternal([]Batch{entries}, testFactory(), func(r *http.Request) bool {
		return false
	})
	if err != nil {
		t.Error(err)
	}
	if len(reqs) != 1 {
		t.Error(len(reqs))
	}
}

func TestNewRequestSplitNeeded(t *testing.T) {
	testPayload := testSplittablePayloadEntry{
		rawData: json.RawMessage(`"123456789"`),
		splitPayloads: []*testSplittablePayloadEntry{
			{rawData: json.RawMessage(`"1234"`)},
			{rawData: json.RawMessage(`"56789"`)},
		},
	}
	entries := []MapEntry{&testPayload}
	reqs, err := newRequestsInternal([]Batch{entries}, testFactory(), func(r *http.Request) bool {
		shouldSplit, err := payloadContains(r, "testSplittable", "123456789")

		if nil != err {
			t.Error(err)
		}

		return shouldSplit
	})
	if err != nil {
		t.Error(err)
	}
	if len(reqs) != 2 {
		t.Error(len(reqs))
	}
}

func TestNewRequestSplittingMultiplePayloadsNeeded(t *testing.T) {
	testUnsplittablePayloadEntry := testUnsplittablePayloadEntry{
		rawData: json.RawMessage(`"abc"`),
	}
	testSplittablePayload := testSplittablePayloadEntry{
		rawData: json.RawMessage(`"123456789"`),
		splitPayloads: []*testSplittablePayloadEntry{
			{rawData: json.RawMessage(`"1234"`)},
			{rawData: json.RawMessage(`"56789"`)},
		},
	}
	entries := []MapEntry{&testUnsplittablePayloadEntry, &testSplittablePayload}
	reqs, err := newRequestsInternal([]Batch{entries}, testFactory(), func(r *http.Request) bool {
		shouldSplit, err := payloadContains(r, "testSplittable", "123456789")

		if nil != err {
			t.Error(err)
		}

		return shouldSplit
	})
	if err != nil {
		t.Error(err)
	}
	if len(reqs) != 2 {
		t.Error(len(reqs))
	}

	expectedSplitPayloads := []string{"1234", "56789"}
	for i := 0; i < 2; i++ {
		hasUnsplittablePayload, err := payloadContains(reqs[i], "testUnsplittable", "abc")
		if err != nil {
			t.Error(err)
		}
		if !hasUnsplittablePayload {
			t.Error("Each request should contain the unsplittable payload")
		}

		hasSplittablePayload, err := payloadContains(reqs[i], "testSplittable", expectedSplitPayloads[i])
		if err != nil {
			t.Error(err)
		}
		if !hasSplittablePayload {
			t.Errorf("testSplittable did not contain %q", expectedSplitPayloads[i])
		}
	}
}

func TestNewRequestCantSplitPayload(t *testing.T) {
	testPayload := testSplittablePayloadEntry{
		rawData: json.RawMessage(`"123456789"`),
	}
	entries := []MapEntry{&testPayload}
	reqs, err := newRequestsInternal([]Batch{entries}, testFactory(), func(r *http.Request) bool {
		shouldSplit, err := payloadContains(r, "testSplittable", "123456789")

		if nil != err {
			t.Error(err)
		}

		return shouldSplit
	})

	if err != errUnableToSplit {
		t.Error(err)
	}
	if reqs != nil {
		t.Error("reqs should be nil")
	}
}

func TestNewRequestCantSplitPayloadsEnough(t *testing.T) {
	testPayload := testSplittablePayloadEntry{
		rawData: json.RawMessage(`"123456789"`),
		splitPayloads: []*testSplittablePayloadEntry{
			{rawData: json.RawMessage(`"1234"`)},
			{rawData: json.RawMessage(`"56789"`)},
		},
	}
	entries := []MapEntry{&testPayload}
	reqs, err := newRequestsInternal([]Batch{entries}, testFactory(), func(r *http.Request) bool {
		isOriginalPayload, err := payloadContains(r, "testSplittable", "123456789")

		if nil != err {
			t.Error(err)
		}

		isPayloadThatCantBeSplitAgain, err := payloadContains(r, "testSplittable", "56789")

		if nil != err {
			t.Error(err)
		}

		return isOriginalPayload || isPayloadThatCantBeSplitAgain
	})

	if err != errUnableToSplit {
		t.Error(err)
	}
	if reqs != nil {
		t.Error("reqs should be nil")
	}
}

func TestNeedsToSplitBatchesAndEntries(t *testing.T) {
	// Create a set of batches that must be first be split into two requests.
	// The group1 request must be further subdivided into two requests, creating
	// a total of 3 requests.
	group1 := []MapEntry{
		&testSplittablePayloadEntry{
			rawData: randomJSON(maxCompressedSizeBytes * 4),
			splitPayloads: []*testSplittablePayloadEntry{
				{rawData: randomJSON(maxCompressedSizeBytes * 2)},
				{rawData: randomJSON(maxCompressedSizeBytes * 2)},
			},
		},
	}
	group2 := []MapEntry{
		&testSplittablePayloadEntry{
			rawData: randomJSON(maxCompressedSizeBytes),
		},
	}
	reqs, err := newRequests([]Batch{group1, group2}, testFactory())
	if err != nil {
		t.Fatal(err)
	}
	if len(reqs) != 3 {
		t.Fatal(len(reqs))
	}
}

func TestLargeRequestNeedsSplit(t *testing.T) {
	js := randomJSON(4 * maxCompressedSizeBytes)
	payloadEntry := testUnsplittablePayloadEntry{rawData: js}
	reqs, err := newRequests([]Batch{{&payloadEntry}}, testFactory())
	if reqs != nil {
		t.Error(reqs)
	}
	if err != errUnableToSplit {
		t.Error(err)
	}
}

func TestLargeRequestNoSplit(t *testing.T) {
	js := randomJSON(maxCompressedSizeBytes / 2)
	payloadEntry := testUnsplittablePayloadEntry{rawData: js}
	reqs, err := newRequests([]Batch{{&payloadEntry}}, testFactory())
	if err != nil {
		t.Fatal(err)
	}
	if len(reqs) != 1 {
		t.Fatal(len(reqs))
	}
	req := reqs[0]
	if u := req.URL.String(); u != defaultMetricURL {
		t.Error(u)
	}
}

func payloadContains(r *http.Request, fieldName string, value string) (bool, error) {
	bodyReader, _ := r.GetBody()
	compressedBytes, _ := ioutil.ReadAll(bodyReader)
	uncompressedBytes, _ := internal.Uncompress(compressedBytes)
	var entry []map[string]string
	err := json.Unmarshal(uncompressedBytes, &entry)

	return entry[0][fieldName] == value, err
}

func randomJSON(numBytes int) json.RawMessage {
	digits := []byte{'1', '2', '3', '4', '5', '6', '7', '8', '9'}
	js := make([]byte, numBytes)
	for i := 0; i < len(js); i++ {
		js[i] = digits[rand.Intn(len(digits))]
	}
	return js
}

func testFactory() RequestFactory {
	factory, _ := NewMetricRequestFactory(WithNoDefaultKey())
	return factory
}
