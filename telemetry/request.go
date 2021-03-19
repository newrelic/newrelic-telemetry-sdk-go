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

func newRequests(batches []PayloadBatch, factory RequestFactory) ([]*http.Request, error) {
	return newRequestsInternal(batches, factory, requestNeedsSplit)
}

func newRequestsInternal(batches []PayloadBatch, factory RequestFactory, needsSplit func(*http.Request) bool) ([]*http.Request, error) {
	r, err := factory.BuildRequest(batches)
	if nil != err {
		return nil, err
	}

	if !needsSplit(r) {
		return []*http.Request{r}, nil
	}

	var reqs []*http.Request
	var splitBatches1 []PayloadBatch
	var splitBatches2 []PayloadBatch
	payloadWasSplit := false

	if len(batches) > 1 {
		middle := len(batches) / 2
		splitBatches1 = batches[0:middle]
		splitBatches2 = batches[middle:]
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
		splitBatches1 = []PayloadBatch{payload1Entries}
		splitBatches2 = []PayloadBatch{payload2Entries}
	}

	if !payloadWasSplit {
		return nil, errUnableToSplit
	}

	for _, b := range [][]PayloadBatch{splitBatches1, splitBatches2} {
		rs, err := newRequestsInternal(b, factory, needsSplit)
		if nil != err {
			return nil, err
		}
		reqs = append(reqs, rs...)
	}
	return reqs, nil
}
