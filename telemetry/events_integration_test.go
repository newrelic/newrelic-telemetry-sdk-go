// Copyright 2019 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0
// +build integration

package telemetry

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEvent(t *testing.T) {
	t.Parallel()

	cfg := NewIntegrationTestConfig(t)

	h, err := NewHarvester(cfg)
	assert.NoError(t, err)
	assert.NotNil(t, h)

	err = h.RecordEvent(Event{
		EventType: "testEvent",
		Attributes: map[string]interface{}{
			"zip": "zap",
		},
	})
	assert.NoError(t, err)

	h.HarvestNow(context.Background())
}

func TestEventBatch(t *testing.T) {
	t.Parallel()

	cfg := NewIntegrationTestConfig(t)

	h, err := NewHarvester(cfg)
	assert.NoError(t, err)
	assert.NotNil(t, h)

	// Batch up a few events
	for x := 0; x < 10; x++ {
		err = h.RecordEvent(Event{EventType: "testEvent", Attributes: map[string]interface{}{"zip": "zap", "count": x}})
		assert.NoError(t, err)
	}

	h.HarvestNow(context.Background())
}
