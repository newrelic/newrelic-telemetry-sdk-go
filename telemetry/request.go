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

type splittableBatch interface {
	PayloadEntry
	split()  []splittableBatch
}

var (
	errUnableToSplit = fmt.Errorf("unable to split large payload further")
)

func requestNeedsSplit(r http.Request) bool {
	return r.ContentLength >= maxCompressedSizeBytes
}

func newRequests(entries []PayloadEntry, apiKey string, rawUrl string, userAgent string) ([]request, error) {
	return newRequestsInternal(entries, apiKey, rawUrl, userAgent, requestNeedsSplit)
}

func newRequestsInternal(common PayloadEntry, batch splittableBatch, apiKey string, rawUrl string, userAgent string, needsSplit func(http.Request) bool) ([]request, error) {
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
	if nil != common {
		r, err := factory.BuildRequest([]PayloadEntry{common, batch})
	} else {
		r, err := factory.BuildRequest([]PayloadEntry{batch})
	}

	if !needsSplit(r) {
		return []request{r}, nil
	}

	var reqs []request
	batches := batch.split()
	if nil == batches {
		return nil, errUnableToSplit
	}

	for _, b := range batches {
		rs, err := newRequestsInternal(b, apiKey, url, userAgent, needsSplit)
		if nil != err {
			return nil, err
		}
		reqs = append(reqs, rs...)
	}
	return reqs, nil
}
