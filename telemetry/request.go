// Copyright 2019 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package telemetry

import (
	"fmt"
	"net/http"
	"net/url"
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

func requestNeedsSplit(r http.Request) bool {
	return r.ContentLength >= maxCompressedSizeBytes
}

func newRequests(entries []PayloadEntry, apiKey string, rawUrl string, userAgent string) ([]http.Request, error) {
	return newRequestsInternal(entries, apiKey, rawUrl, userAgent, requestNeedsSplit)
}

func newRequestsInternal(entries []PayloadEntry, apiKey string, rawUrl string, userAgent string, needsSplit func(http.Request) bool) ([]http.Request, error) {
	url, err := url.Parse(rawUrl)
	if nil != err {
		return nil, err
	}
	factory := &requestFactory{
		insertKey:    apiKey,
		noDefaultKey: false,
		host:         url.Host,
		path:         url.Path,
		userAgent:    userAgent,
	}

	r, err := factory.BuildRequest(entries)

	if !needsSplit(r) {
		return []http.Request{r}, nil
	}

	var reqs []http.Request
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
		rs, err := newRequestsInternal(b, apiKey, rawUrl, userAgent, needsSplit)
		if nil != err {
			return nil, err
		}
		reqs = append(reqs, rs...)
	}
	return reqs, nil
}
