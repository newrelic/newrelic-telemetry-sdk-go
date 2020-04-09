package internal

import "time"

type MetricIdentity struct {
	Name           string
	AttributesJSON string
}

type LastValue struct {
	When  time.Time
	Value float64
}

type Datapoints map[MetricIdentity]LastValue
