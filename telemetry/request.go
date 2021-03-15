// Copyright 2019 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package telemetry

import (
	"fmt"
	"net/http"
)

const (
	maxCompressedSizeBytes = 1e6
)

type splittablePayloadEntry interface {
	PayloadEntry
	split() []splittablePayloadEntry
}

var (
	errUnableToSplit = fmt.Errorf("unable to split large payload further")
)

func requestNeedsSplit(r *http.Request) bool {
	return r.ContentLength >= maxCompressedSizeBytes
}

func newRequests(batches [][]PayloadEntry, factory RequestFactory) ([]*http.Request, error) {
	return newRequestsInternal(batches, factory, requestNeedsSplit)
}

func newRequestsInternal(batches [][]PayloadEntry, factory RequestFactory, needsSplit func(*http.Request) bool) ([]*http.Request, error) {
	r, err := factory.BuildRequest(batches)
	if nil != err {
		return nil, err
	}

	if !needsSplit(r) {
		return []*http.Request{r}, nil
	}

	var reqs []*http.Request
	var splitPayload1 [][]PayloadEntry
	var splitPayload2 [][]PayloadEntry
	payloadWasSplit := false

	if len(batches) > 1 {
		middle := len(batches) / 2
		splitPayload1 = batches[0:middle]
		splitPayload2 = batches[middle:]
		payloadWasSplit = true
	} else if len(batches) == 1 {
		var payload1Entries []PayloadEntry
		var payload2Entries []PayloadEntry
		for _, e := range batches[0] {
			splittable, isPayloadSplittable := e.(splittablePayloadEntry)
			if isPayloadSplittable {
				splitEntry := splittable.split()
				if splitEntry != nil {
					payload1Entries = append(payload1Entries, splitEntry[0].(PayloadEntry))
					payload2Entries = append(payload2Entries, splitEntry[1].(PayloadEntry))
					payloadWasSplit = true
					continue
				}
			}

			payload1Entries = append(payload1Entries, e)
			payload2Entries = append(payload2Entries, e)
		}
		splitPayload1 = [][]PayloadEntry{payload1Entries}
		splitPayload2 = [][]PayloadEntry{payload2Entries}
	}

	if !payloadWasSplit {
		return nil, errUnableToSplit
	}

	for _, b := range [][][]PayloadEntry{splitPayload1, splitPayload2} {
		rs, err := newRequestsInternal(b, factory, needsSplit)
		if nil != err {
			return nil, err
		}
		reqs = append(reqs, rs...)
	}
	return reqs, nil
}
