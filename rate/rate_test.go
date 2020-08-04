package rate

import (
	"fmt"
	"github.com/newrelic/newrelic-telemetry-sdk-go/telemetry"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
	"time"
)

func Example() {
	h, err := telemetry.NewHarvester(
		telemetry.ConfigAPIKey(os.Getenv("NEW_RELIC_INSIGHTS_INSERT_API_KEY")),
	)
	if err != nil {
		fmt.Println(err)
	}
	dc := NewRateCalculator()

	attributes := map[string]interface{}{
		"id":  123,
		"zip": "zap",
	}
	for {
		currentValue := 10.0
		if m, ok := dc.GetRate("errorsPerSecond", attributes, currentValue, time.Now()); ok {
			h.RecordMetric(m)
		}
		time.Sleep(1 * time.Second)
		currentValue = 22.0
		if m, ok := dc.GetRate("errorsPerSecond", attributes, currentValue, time.Now()); ok {
			h.RecordMetric(m)
		}
		time.Sleep(1 * time.Second)
		currentValue = 10.0
		if m, ok := dc.GetRate("errorsPerSecond", attributes, currentValue, time.Now()); ok {
			h.RecordMetric(m)
		}
		time.Sleep(1 * time.Second)
	}
}

func TestRateCalculator_BasicUsage(t *testing.T) {

	now := time.Unix(1000000, 0)
	rc := NewRateCalculator()

	attrs := map[string]interface{}{"abc": "123"}
	_, valid := rc.GetRate("errorsPerSecond", attrs, 10, now)
	// no previous timestamp
	assert.False(t, valid)

	g, valid := rc.GetRate("errorsPerSecond", attrs, 20, now.Add(1*time.Second))
	assert.True(t, valid)
	assert.Equal(t, 20.0, g.Value)

	g, valid = rc.GetRate("errorsPerSecond", attrs, 10, now.Add(5*time.Second))
	assert.True(t, valid)
	assert.Equal(t, 2.0, g.Value)
}

func TestRateCalculator_OlderTimestampsNotAccepted(t *testing.T) {

	now := time.Unix(1000000, 0)
	rc := NewRateCalculator()

	attrs := map[string]interface{}{"abc": "123"}
	_, valid := rc.GetRate("errorsPerSecond", attrs, 10, now)
	// no previous timestamp
	assert.False(t, valid)

	g, valid := rc.GetRate("errorsPerSecond", attrs, 20, now.Add(1*time.Second))
	assert.True(t, valid)
	assert.Equal(t, 20.0, g.Value)

	_, valid = rc.GetRate("errorsPerSecond", attrs, 10, now.Add(-5*time.Second))
	assert.False(t, valid)

	g, valid = rc.GetRate("errorsPerSecond", attrs, 10, now.Add(2*time.Second))
	assert.True(t, valid)
	assert.Equal(t, 5.0, g.Value)
}

func TestRateCalculator_GetCumulativeRate(t *testing.T) {

	now := time.Unix(1000000, 0)
	rc := NewRateCalculator()

	attrs := map[string]interface{}{"abc": "123"}
	_, valid := rc.GetCumulativeRate("requestsPerSecond", attrs, 10, now)
	// no previous value
	assert.False(t, valid)

	g, valid := rc.GetCumulativeRate("requestsPerSecond", attrs, 20, now.Add(1*time.Second))
	assert.True(t, valid)
	assert.Equal(t, 10.0, g.Value)

	g, valid = rc.GetCumulativeRate("requestsPerSecond", attrs, 10, now.Add(2*time.Second))
	assert.True(t, valid)
	assert.Equal(t, 0.0, g.Value)

	g, valid = rc.GetCumulativeRate("requestsPerSecond", attrs, 20, now.Add(10*time.Second))
	assert.True(t, valid)
	assert.Equal(t, 1.0, g.Value)
}
