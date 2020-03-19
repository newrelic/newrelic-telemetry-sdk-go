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

type metricIdentity struct {
	name           string
	attributesJSON string
}

type SummaryValue struct {
	// Count is the count of occurrences of this metric for this time period.
	Count float64
	// Sum is the sum of all occurrences of this metric for this time period.
	Sum float64
}

type lastCountValue struct {
	when  time.Time
	value float64
}

type lastSummaryValue struct {
	when  time.Time
	value SummaryValue
}

// DeltaCalculator is used to create Count metrics from cumulative values.
type DeltaCalculator struct {
	lock                    sync.Mutex
	countDatapoints         map[metricIdentity]lastCountValue
	summaryDatapoints       map[metricIdentity]lastSummaryValue
	lastClean               time.Time
	expirationCheckInterval time.Duration
	expirationAge           time.Duration
}

// NewDeltaCalculator creates a new DeltaCalculator.  A single DeltaCalculator
// stores all cumulative values seen in order to compute deltas.
func NewDeltaCalculator() *DeltaCalculator {
	return &DeltaCalculator{
		countDatapoints:   make(map[metricIdentity]lastCountValue),
		summaryDatapoints: make(map[metricIdentity]lastSummaryValue),
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

// CountMetric creates a count metric from the difference between the values and
// timestamps of multiple calls.  If this is the first time the name/attributes
// combination has been seen then the `valid` return value will be false.
func (dc *DeltaCalculator) CountMetric(name string, attributes map[string]interface{}, val float64, now time.Time) (count telemetry.Count, valid bool) {
	var attributesJSON []byte
	if nil != attributes {
		attributesJSON = internal.MarshalOrderedAttributes(attributes)
	}
	dc.lock.Lock()
	defer dc.lock.Unlock()

	if now.Sub(dc.lastClean) > dc.expirationCheckInterval {
		cutoff := now.Add(-dc.expirationAge)
		for k, v := range dc.countDatapoints {
			if v.when.Before(cutoff) {
				delete(dc.countDatapoints, k)
			}
		}
		dc.lastClean = now
	}

	id := metricIdentity{name: name, attributesJSON: string(attributesJSON)}
	var timestampsOrdered bool
	last, ok := dc.countDatapoints[id]
	if ok {
		delta := val - last.value
		timestampsOrdered = now.After(last.when)
		if timestampsOrdered && delta >= 0 {
			count.Name = name
			count.AttributesJSON = attributesJSON
			count.Value = delta
			count.Timestamp = last.when
			count.Interval = now.Sub(last.when)
			valid = true
		}
	}
	if !ok || timestampsOrdered {
		dc.countDatapoints[id] = lastCountValue{value: val, when: now}
	}
	return
}

// SummaryMetric creates a summary metric from the difference between the values and
// timestamps of multiple calls.  If this is the first time the name/attributes
// combination has been seen then the `valid` return value will be false.
func (dc *DeltaCalculator) SummaryMetric(name string, attributes map[string]interface{}, val SummaryValue, now time.Time) (summary telemetry.Summary, valid bool) {
	var attributesJSON []byte
	if nil != attributes {
		attributesJSON = internal.MarshalOrderedAttributes(attributes)
	}
	dc.lock.Lock()
	defer dc.lock.Unlock()

	if now.Sub(dc.lastClean) > dc.expirationCheckInterval {
		cutoff := now.Add(-dc.expirationAge)
		for k, v := range dc.summaryDatapoints {
			if v.when.Before(cutoff) {
				delete(dc.summaryDatapoints, k)
			}
		}
		dc.lastClean = now
	}

	id := metricIdentity{name: name, attributesJSON: string(attributesJSON)}
	var timestampsOrdered bool
	last, ok := dc.summaryDatapoints[id]
	if ok {
		deltaCount := val.Count - last.value.Count
		deltaSum := val.Sum - last.value.Sum
		timestampsOrdered = now.After(last.when)
		if timestampsOrdered && deltaCount >= 0 {
			summary.Name = name
			summary.AttributesJSON = attributesJSON
			summary.Sum = deltaSum
			summary.Count = deltaCount
			summary.Timestamp = last.when
			summary.Interval = now.Sub(last.when)
			valid = true
		}
	}
	if !ok || timestampsOrdered {
		dc.summaryDatapoints[id] = lastSummaryValue{value: val, when: now}
	}
	return
}
