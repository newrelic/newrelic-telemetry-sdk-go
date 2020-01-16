// Copyright 2019 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"time"

	"github.com/newrelic/newrelic-telemetry-sdk-go/telemetry"
)

func databaseCall(collection string) {
	if rand.Intn(10) < 5 {
		databaseMisses.Increment()
		time.Sleep(10)
	} else {
		databaseHits.Increment()
		time.Sleep(1)
	}
}

func fetch(w http.ResponseWriter, r *http.Request) {
	databaseCall("users")
	io.WriteString(w, "fetch!")
}

func index(w http.ResponseWriter, r *http.Request) {
	time.Sleep(5)
	io.WriteString(w, "index!")
}

func outboundCall(u *url.URL) {
	req, _ := http.NewRequest("GET", u.String(), nil)

	before := time.Now()
	http.DefaultClient.Do(req)

	statuses := []int{200, 200, 200, 200, 200, 404, 503}
	status := statuses[rand.Int()%len(statuses)]

	h.MetricAggregator().Summary("service.span.responseTime", map[string]interface{}{
		"host":        u.Host,
		"method":      "GET",
		"http.status": status,
	}).RecordDuration(time.Since(before))
}

func outbound(w http.ResponseWriter, r *http.Request) {
	u, _ := url.Parse("http://www.example.com")
	outboundCall(u)
	io.WriteString(w, "outbound!")
}

var (
	h              *telemetry.Harvester
	databaseHits   *telemetry.AggregatedCount
	databaseMisses *telemetry.AggregatedCount
)

func randomID() string {
	// rand.Uint64 is Go 1.8+
	u1 := rand.Uint32()
	u2 := rand.Uint32()
	u := (uint64(u1) << 32) | uint64(u2)
	return fmt.Sprintf("%016x", u)
}

func wrapHandler(path string, handler func(http.ResponseWriter, *http.Request)) (string, func(http.ResponseWriter, *http.Request)) {
	return path, func(rw http.ResponseWriter, req *http.Request) {
		s := h.MetricAggregator().Summary("service.responseTime", map[string]interface{}{
			"name":        path,
			"http.method": req.Method,
			"isWeb":       true,
		})
		before := time.Now()
		handler(rw, req)
		s.RecordDuration(time.Since(before))

		h.RecordSpan(telemetry.Span{
			ID:          randomID(),
			TraceID:     randomID(),
			Name:        "service.responseTime",
			Timestamp:   before,
			Duration:    time.Since(before),
			ServiceName: "Telemetry SDK Example",
			Attributes: map[string]interface{}{
				"name":        path,
				"http.method": req.Method,
				"isWeb":       true,
			},
		})
	}
}

func gatherMemStats() {
	allocations := h.MetricAggregator().Gauge("runtime.MemStats.heapAlloc", map[string]interface{}{})
	var rtm runtime.MemStats
	var interval = 1 * time.Second
	for {
		<-time.After(interval)
		runtime.ReadMemStats(&rtm)
		allocations.Value(float64(rtm.HeapAlloc))
	}
}

func mustGetEnv(v string) string {
	val := os.Getenv(v)
	if val == "" {
		panic(fmt.Sprintf("%s unset", v))
	}
	return val
}

func main() {
	rand.Seed(time.Now().UnixNano())
	var err error
	h, err = telemetry.NewHarvester(
		telemetry.ConfigAPIKey(mustGetEnv("NEW_RELIC_INSIGHTS_INSERT_API_KEY")),
		telemetry.ConfigCommonAttributes(map[string]interface{}{
			"app.name":  "myServer",
			"host.name": "dev.server.com",
			"env":       "staging",
		}),
		telemetry.ConfigBasicErrorLogger(os.Stderr),
		telemetry.ConfigBasicDebugLogger(os.Stdout),
		func(cfg *telemetry.Config) {
			cfg.MetricsURLOverride = os.Getenv("NEW_RELIC_METRICS_URL")
			cfg.SpansURLOverride = os.Getenv("NEW_RELIC_SPANS_URL")
		},
	)
	if nil != err {
		panic(err)
	}
	databaseAttributes := map[string]interface{}{
		"db.type":     "sql",
		"db.instance": "customers",
	}
	databaseHits = h.MetricAggregator().Count("database.cache.hits", databaseAttributes)
	databaseMisses = h.MetricAggregator().Count("database.cache.misses", databaseAttributes)

	go gatherMemStats()

	http.HandleFunc(wrapHandler("/", index))
	http.HandleFunc(wrapHandler("/fetch", fetch))
	http.HandleFunc(wrapHandler("/outbound", outbound))
	http.ListenAndServe(":8000", nil)
}
