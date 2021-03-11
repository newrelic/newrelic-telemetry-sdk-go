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

func newRequests(entries []PayloadEntry, factory RequestFactory) ([]*http.Request, error) {
	return newRequestsInternal(entries, factory, requestNeedsSplit)
}

func newRequestsInternal(entries []PayloadEntry, factory RequestFactory, needsSplit func(*http.Request) bool) ([]*http.Request, error) {
	r, err := factory.BuildRequest(entries)
	if nil != err {
		return nil, err
	}

	if !needsSplit(r) {
		return []*http.Request{r}, nil
	}

	var reqs []*http.Request
	var splitPayload1 []PayloadEntry
	var splitPayload2 []PayloadEntry
	payloadWasSplit := false
	for _, e := range entries {
		splittable, isPayloadSplittable := e.(splittablePayloadEntry)
		if isPayloadSplittable {
			splitEntry := splittable.split()
			if splitEntry != nil {
				splitPayload1 = append(splitPayload1, splitEntry[0].(PayloadEntry))
				splitPayload2 = append(splitPayload2, splitEntry[1].(PayloadEntry))
				payloadWasSplit = true
				continue
			}
		}

		splitPayload1 = append(splitPayload1, e)
		splitPayload2 = append(splitPayload2, e)
	}

	if !payloadWasSplit {
		return nil, errUnableToSplit
	}

	for _, b := range [][]PayloadEntry{splitPayload1, splitPayload2} {
		rs, err := newRequestsInternal(b, factory, needsSplit)
		if nil != err {
			return nil, err
		}
		reqs = append(reqs, rs...)
	}
	return reqs, nil
}
