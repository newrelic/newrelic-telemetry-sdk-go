// Copyright 2019 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0
// +build unit integration benchmark

package telemetry

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

// configTesting is the config function to be used when testing. It sets the
// APIKey but disables the harvest goroutine.
func configTesting(cfg *Config) {
	cfg.APIKey = "api-key"
	cfg.HarvestPeriod = 0
}

// compactJSONString removes the whitespace from a JSON string.  This function
// will panic if the string provided is not valid JSON.
func compactJSONString(js string) string {
	buf := new(bytes.Buffer)
	if err := json.Compact(buf, []byte(js)); err != nil {
		panic(fmt.Errorf("unable to compact JSON: %v", err))
	}
	return buf.String()
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (fn roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

// optional interface required for go1.4 and go1.5
func (fn roundTripperFunc) CancelRequest(*http.Request) {}

func emptyResponse(status int) *http.Response {
	return &http.Response{
		StatusCode: status,
		Body:       ioutil.NopCloser(bytes.NewReader([]byte(""))),
	}
}

// sortedMetricsHelper is used to sort metrics for JSON comparison.
type sortedMetricsHelper []json.RawMessage

func (h sortedMetricsHelper) Len() int {
	return len(h)
}
func (h sortedMetricsHelper) Less(i, j int) bool {
	return string(h[i]) < string(h[j])
}
func (h sortedMetricsHelper) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
}

// multiAttemptRoundTripper will fail the first n requests after reading
// their body with a 418. Subsequent requests will be returned a 200.
func multiAttemptRoundTripper(n int) roundTripperFunc {
	var attempt int
	return roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		defer func() { attempt++ }()
		if _, err := ioutil.ReadAll(req.Body); err != nil {
			return nil, err
		}
		if attempt < n {
			return emptyResponse(418), nil
		}
		return emptyResponse(200), nil
	})
}
