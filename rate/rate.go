package rate

import (
	"sync"
	"time"

	"github.com/newrelic/newrelic-telemetry-sdk-go/internal"
	"github.com/newrelic/newrelic-telemetry-sdk-go/telemetry"
)

// RateCalculator is used to create Gauge metrics from rate values.
type RateCalculator struct {
	lock                    sync.Mutex
	datapoints              internal.Datapoints
	lastClean               time.Time
	expirationCheckInterval time.Duration
	expirationAge           time.Duration
}

// NewRateCalculator creates a new RateCalculator.  A single RateCalculator
// stores the last timestamp in order to compute the rate.
func NewRateCalculator() *RateCalculator {
	return &RateCalculator{
		datapoints: internal.Datapoints{},
		// These defaults are described in the Set method doc comments.
		expirationCheckInterval: 20 * time.Minute,
		expirationAge:           20 * time.Minute,
	}
}

// GetRate creates a Gauge metric with rate of change based on the previous timestamp.
// If no previous timestamp is NOT found, returns false (as no calculation is made)
// If a previous timestamp is found use it to get the elapsed time (in seconds) and use that as the denominator
// Rate = value / (now - before)[s]
func (rc *RateCalculator) GetRate(name string, attributes map[string]interface{}, val float64, now time.Time) (gauge telemetry.Gauge, valid bool) {
	var attributesJSON []byte
	if nil != attributes {
		attributesJSON = internal.MarshalOrderedAttributes(attributes)
	}
	rc.lock.Lock()
	defer rc.lock.Unlock()

	id := internal.MetricIdentity{Name: name, AttributesJSON: string(attributesJSON)}

	last, found := rc.datapoints[id]
	if found {
		// don't accept timestamps older that the last one for this metric
		if last.When.Before(now) {
			elapsedSeconds := now.Sub(last.When).Seconds()
			rate := val / elapsedSeconds

			gauge.Name = name
			gauge.Timestamp = now
			gauge.Value = rate
			gauge.Attributes = attributes
			gauge.AttributesJSON = attributesJSON

			valid = true
		}
	} else {
		rc.datapoints[id] = internal.LastValue{When: now}
	}

	return
}

// GetCumulativeRate creates a Gauge metric with rate of change based on the previous timestamp and value.
// If no previous timestamp is NOT found, returns false (as no calculation is made)
// If a previous timestamp is found use it to get the elapsed time (in seconds) and use that as the denominator
// Rate = value / (now - before)[s]
func (rc *RateCalculator) GetCumulativeRate(name string, attributes map[string]interface{}, val float64, now time.Time) (gauge telemetry.Gauge, valid bool) {
	var attributesJSON []byte
	if nil != attributes {
		attributesJSON = internal.MarshalOrderedAttributes(attributes)
	}
	rc.lock.Lock()
	defer rc.lock.Unlock()

	id := internal.MetricIdentity{Name: name, AttributesJSON: string(attributesJSON)}

	last, found := rc.datapoints[id]
	if found {
		// don't accept timestamps older that the last one for this metric
		if last.When.Before(now) {
			elapsedSeconds := now.Sub(last.When).Seconds()
			diff := val - last.Value
			// only positive deltas accepted
			if diff >= 0 {
			  rate := diff / elapsedSeconds

			  gauge.Name = name
			  gauge.Timestamp = now
			  gauge.Value = rate
			  gauge.Attributes = attributes
			  gauge.AttributesJSON = attributesJSON
			
			  valid = true
			}
		}
	} else {
		rc.datapoints[id] = internal.LastValue{When: now, Value: val}
	}

	return
}
