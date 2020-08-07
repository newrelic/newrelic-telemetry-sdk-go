// Copyright 2019 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0
// +build unit

package internal

import (
	"bytes"
	"math"
	"testing"
)

func TestAttributesWriteJSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		key    string
		val    interface{}
		expect string
	}{
		{"string", "string", `{"string":"string"}`},
		{"true", true, `{"true":true}`},
		{"false", false, `{"false":false}`},
		{"uint8", uint8(1), `{"uint8":1}`},
		{"uint16", uint16(1), `{"uint16":1}`},
		{"uint32", uint32(1), `{"uint32":1}`},
		{"uint64", uint64(1), `{"uint64":1}`},
		{"uint", uint(1), `{"uint":1}`},
		{"uintptr", uintptr(1), `{"uintptr":1}`},
		{"int8", int8(1), `{"int8":1}`},
		{"int16", int16(1), `{"int16":1}`},
		{"int32", int32(1), `{"int32":1}`},
		{"int64", int64(1), `{"int64":1}`},
		{"int", int(1), `{"int":1}`},
		{"float32", float32(1), `{"float32":1}`},
		{"float64", float64(1), `{"float64":1}`},
		{"default", func() {}, `{"default":"func()"}`},
		{"NaN", math.NaN(), `{"NaN":"NaN"}`},
		{"positive-infinity", math.Inf(1), `{"positive-infinity":"infinity"}`},
		{"negative-infinity", math.Inf(-1), `{"negative-infinity":"infinity"}`},
	}

	for _, test := range tests {
		buf := &bytes.Buffer{}
		ats := Attributes(map[string]interface{}{
			test.key: test.val,
		})
		ats.WriteJSON(buf)
		got := buf.String()
		if got != test.expect {
			t.Errorf("key='%s' val=%v expect='%s' got='%s'",
				test.key, test.val, test.expect, got)
		}
	}
}

func TestEmptyAttributesWriteJSON(t *testing.T) {
	t.Parallel()

	var ats Attributes
	buf := &bytes.Buffer{}
	ats.WriteJSON(buf)
	got := buf.String()
	if got != `{}` {
		t.Error(got)
	}
}

func TestOrderedAttributesWriteJSON(t *testing.T) {
	t.Parallel()

	ats := map[string]interface{}{
		"z": 123,
		"b": "hello",
		"a": true,
		"x": 13579,
		"m": "zap",
		"c": "zip",
	}
	got := string(MarshalOrderedAttributes(ats))
	if got != `{"a":true,"b":"hello","c":"zip","m":"zap","x":13579,"z":123}` {
		t.Error(got)
	}
}

func TestEmptyOrderedAttributesWriteJSON(t *testing.T) {
	t.Parallel()

	got := string(MarshalOrderedAttributes(nil))
	if got != `{}` {
		t.Error(got)
	}
}
