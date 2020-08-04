// Copyright 2019 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

// Package cumulative creates Count metrics from cumulative values.
package cumulative

import (
	"sync"
	"time"

	"github.com/newrelic/newrelic-telemetry-sdk-go/internal"
	"github.com/newrelic/newrelic-telemetry-sdk-go/telemetry"
)

// DeltaCalculator is used to create Count metrics from cumulative values.
type DeltaCalculator struct {
	lock                    sync.Mutex
	datapoints              internal.Datapoints
	lastClean               time.Time
	expirationCheckInterval time.Duration
	expirationAge           time.Duration
}

// NewDeltaCalculator creates a new DeltaCalculator.  A single DeltaCalculator
// stores all cumulative values seen in order to compute deltas.
func NewDeltaCalculator() *DeltaCalculator {
	return &DeltaCalculator{
		datapoints: internal.Datapoints{},
		// These defaults are described in the Set method doc comments.
		expirationCheckInterval: 20 * time.Minute,
		expirationAge:           20 * time.Minute,
	}
}

// SetExpirationAge configures how old entries must be for expiration.  The
// default is twenty minutes.
func (dc *DeltaCalculator) SetExpirationAge(age time.Duration) *DeltaCalculator {
	dc.lock.Lock()
	defer dc.lock.Unlock()
	dc.expirationAge = age
	return dc
}

// SetExpirationCheckInterval configures how often to check for expired entries.
// The default is twenty minutes.
func (dc *DeltaCalculator) SetExpirationCheckInterval(interval time.Duration) *DeltaCalculator {
	dc.lock.Lock()
	defer dc.lock.Unlock()
	dc.expirationCheckInterval = interval
	return dc
}

// GetCumulativeCount creates a count metric from the difference between the values and
// timestamps of multiple calls.  If this is the first time the name/attributes
// combination has been seen then the `valid` return value will be false.
func (dc *DeltaCalculator) GetCumulativeCount(name string, attributes map[string]interface{}, val float64, now time.Time) (count telemetry.Count, valid bool) {
	var attributesJSON []byte
	if nil != attributes {
		attributesJSON = internal.MarshalOrderedAttributes(attributes)
	}
	dc.lock.Lock()
	defer dc.lock.Unlock()

	if now.Sub(dc.lastClean) > dc.expirationCheckInterval {
		cutoff := now.Add(-dc.expirationAge)
		for k, v := range dc.datapoints {
			if v.When.Before(cutoff) {
				delete(dc.datapoints, k)
			}
		}
		dc.lastClean = now
	}

	id := internal.MetricIdentity{Name: name, AttributesJSON: string(attributesJSON)}
	var timestampsOrdered bool
	last, ok := dc.datapoints[id]
	if ok {
		delta := val - last.Value
		timestampsOrdered = now.After(last.When)
		if timestampsOrdered && delta >= 0 {
			count.Name = name
			count.AttributesJSON = attributesJSON
			count.Value = delta
			count.Timestamp = last.When
			count.Interval = now.Sub(last.When)
			valid = true
		}
	}
	if !ok || timestampsOrdered {
		dc.datapoints[id] = internal.LastValue{Value: val, When: now}
	}
	return
}
