# Go Telemetry SDK [![GoDoc](https://godoc.org/github.com/newrelic/newrelic-telemetry-sdk-go?status.svg)](https://godoc.org/github.com/newrelic/newrelic-telemetry-sdk-go)

What is the New Relic Go Telemetry SDK?

* It's a helper library that supports sending New Relic data from within your Go process
* It’s an example of "best practices" for sending us data

This SDK currently supports sending dimensional metrics and spans to the [Metric API](https://docs.newrelic.com/docs/data-ingest-apis/get-data-new-relic/metric-api/introduction-metric-api) and [Trace API](https://docs.newrelic.com/docs/understand-dependencies/distributed-tracing/trace-api/introduction-trace-api), respectively.


## Requirements

Go 1.13+ is required


## Get started

In order to send metrics or spans to New Relic, you will need an [Insights
Insert API Key](https://docs.newrelic.com/docs/apis/getting-started/intro-apis/understand-new-relic-api-keys#user-api-key).

To install this SDK either use `go get` or clone this repository to
`$GOPATH/src/github.com/newrelic/newrelic-telemetry-sdk-go`

```
go get -u github.com/newrelic/newrelic-telemetry-sdk-go
```

Package
[telemetry](https://godoc.org/github.com/newrelic/newrelic-telemetry-sdk-go/telemetry)
provides basic interaction with the New Relic Metric and Span HTTP APIs,
automatic harvesting on a given schedule, and handling of errors from the API
response.  It also provides the ability to aggregate individual data points into
metrics.

This example code assumes you've set the the `NEW_RELIC_INSIGHTS_INSERT_API_KEY`
environment variable to your Insights Insert API Key.  A larger example is
provided in
[examples/server/main.go](./examples/server/main.go).

```go
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/newrelic/newrelic-telemetry-sdk-go/telemetry"
)

func main() {
	// First create a Harvester.  APIKey is the only required field.
	h, err := telemetry.NewHarvester(telemetry.ConfigAPIKey(os.Getenv("NEW_RELIC_INSIGHTS_INSERT_API_KEY")))
	if err != nil {
		fmt.Println(err)
	}

	// Record Gauge, Count, and Summary metrics using RecordMetric. These
	// metrics are not aggregated.  This is useful for exporting metrics
	// recorded by another system.
	h.RecordMetric(telemetry.Gauge{
		Timestamp: time.Now(),
		Value:     1,
		Name:      "myMetric",
		Attributes: map[string]interface{}{
			"color": "purple",
		},
	})

	// Record spans using RecordSpan.
	h.RecordSpan(telemetry.Span{
		ID:          "12345",
		TraceID:     "67890",
		Name:        "purple-span",
		Timestamp:   time.Now(),
		Duration:    time.Second,
		ServiceName: "ExampleApplication",
		Attributes: map[string]interface{}{
			"color": "purple",
		},
	})

	// Aggregate individual datapoints into metrics using the
	// MetricAggregator.  You can do this in a single line:
	h.MetricAggregator().Count("myCounter", map[string]interface{}{"color": "pink"}).Increment()
	// Or keep a metric reference for fast accumulation:
	counter := h.MetricAggregator().Count("myCounter", map[string]interface{}{"color": "pink"})
	for i := 0; i < 100; i++ {
		counter.Increment()
	}

	// By default, the Harvester sends metrics and spans to the New Relic
	// backend every 5 seconds.  You can force data to be sent at any time
	// using HarvestNow.
	h.HarvestNow(context.Background())
}
```

There are 3 different types of metrics: count, summary, and gauge.  Use
[Harvester.RecordMetric](https://godoc.org/github.com/newrelic/newrelic-telemetry-sdk-go/telemetry#Harvester.RecordMetric)
to record complete metrics that have already been collected. Use
[Harvester.MetricAggregator](https://godoc.org/github.com/newrelic/newrelic-telemetry-sdk-go/telemetry#Harvester.MetricAggregator)
to aggregate numbers into metrics.

| Basic type | Aggregated type | Description | Example |
| ----------- | ----------------- | ----------- | ------- |
| [Gauge](https://godoc.org/github.com/newrelic/newrelic-telemetry-sdk-go/telemetry#Gauge) | [AggregatedGauge](https://godoc.org/github.com/newrelic/newrelic-telemetry-sdk-go/telemetry#AggregatedGauge) | A single value at a single point in time. | Room Temperature. |
| [Count](https://godoc.org/github.com/newrelic/newrelic-telemetry-sdk-go/telemetry#Count) | [AggregatedCount](https://godoc.org/github.com/newrelic/newrelic-telemetry-sdk-go/telemetry#AggregatedCount) | Track the number of occurrences of an event. | Number of errors that have occurred. |
| [Summary](https://godoc.org/github.com/newrelic/newrelic-telemetry-sdk-go/telemetry#Summary) | [AggregatedSummary](https://godoc.org/github.com/newrelic/newrelic-telemetry-sdk-go/telemetry#AggregatedSummary) | Track count, sum, min, and max values over time. | The summarized duration of 100 HTTP requests. |

Count metrics are "delta" counts that indicate the change during the most recent
time period.  You can use the
[cumulative](https://godoc.org/github.com/newrelic/newrelic-telemetry-sdk-go/telemetry)
package to convert "cumulative" count values into delta values.

## Find and use your data

Tips on how to find and query your data in New Relic:
- [Find metric data](https://docs.newrelic.com/docs/data-ingest-apis/get-data-new-relic/metric-api/introduction-metric-api#find-data)
- [Find trace/span data](https://docs.newrelic.com/docs/understand-dependencies/distributed-tracing/trace-api/introduction-trace-api#view-data)

For general querying information, see:
- [Query New Relic data](https://docs.newrelic.com/docs/using-new-relic/data/understand-data/query-new-relic-data)
- [Intro to NRQL](https://docs.newrelic.com/docs/query-data/nrql-new-relic-query-language/getting-started/introduction-nrql)

## Licensing

The New Relic Go Telemetry SDK is licensed under the Apache 2.0 License.
The New Relic Go Telemetry SDK also uses source code from third party
libraries. Full details on which libraries are used and the terms under
which they are licensed can be found in the third party notices document.


## Contributing

Full details are available in our [CONTRIBUTING.md](CONTRIBUTING.md)
file. We'd love to get your contributions to improve the Go Telemetry SDK! Keep in mind when you
submit your pull request, you'll need to sign the CLA via the click-through
using CLA-Assistant. You only have to sign the CLA one time per project.
To execute our corporate CLA, which is required if your contribution is on
behalf of a company, or if you have any questions, please drop us an email
at open-source@newrelic.com.


## Limitations

The New Relic Telemetry APIs are rate limited. Please reference the documentation for [New Relic Metric API](https://docs.newrelic.com/docs/introduction-new-relic-metric-api) and [New Relic Trace API requirements and
limits](https://docs.newrelic.com/docs/apm/distributed-tracing/trace-api/trace-api-general-requirements-limits) on the specifics of the rate limits.
