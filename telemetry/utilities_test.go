// Copyright 2019 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package telemetry

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"
)

func TestJSONString(t *testing.T) {
	// Test that the jsonString type has the intended behavior.
	var emptySlice []byte
	jstr := jsonString(emptySlice)
	if js, _ := json.Marshal(jstr); string(js) != `null` {
		t.Error(string(js))
	}
	if s := fmt.Sprintf("%v", jstr); s != "" {
		t.Error(s)
	}
	jstr = jsonString(`{"zip":"zap"}`)
	if js, _ := json.Marshal(jstr); string(js) != `{"zip":"zap"}` {
		t.Error(string(js))
	}
	if s := fmt.Sprintf("%v", jstr); s != `{"zip":"zap"}` {
		t.Error(s)
	}
}

func TestJSONOrString(t *testing.T) {
	// Test that jsonOrString has the intended behavior.

	out := jsonOrString(nil)
	if js, _ := json.Marshal(out); string(js) != `""` {
		t.Error(string(js))
	}
	if s := fmt.Sprintf("%v", out); s != "" {
		t.Error(s)
	}
	out = jsonOrString([]byte("this is not json"))
	if js, _ := json.Marshal(out); string(js) != `"this is not json"` {
		t.Error(string(js))
	}
	if s := fmt.Sprintf("%v", out); s != "this is not json" {
		t.Error(s)
	}
	out = jsonOrString([]byte(`{"this is":"json"}`))
	if js, _ := json.Marshal(out); string(js) != `{"this is":"json"}` {
		t.Error(string(js))
	}
	if s := fmt.Sprintf("%v", out); s != `{"this is":"json"}` {
		t.Error(s)
	}
}

func checkMinDuration(t *testing.T, expected time.Duration, actual time.Duration) {
	if expected != actual {
		t.Errorf("\nexpect=%s\nactual=%s\n", expected, actual)
	}
}

func TestMinDuration(t *testing.T) {
	t.Parallel()

	t1 := time.Duration(1)
	t2 := time.Duration(5)

	checkMinDuration(t, t1, minDuration(t1, t2))
	checkMinDuration(t, t1, minDuration(t1, t1))
	checkMinDuration(t, t1, minDuration(t2, t1))
	checkMinDuration(t, t2, minDuration(t2, t2))
}
