// Copyright 2019 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0
// +build benchmark

package internal

import (
	"bytes"
	"strconv"
	"testing"
)

func sampleAttributes(num int) map[string]interface{} {
	attributes := make(map[string]interface{})
	for i := 0; i < num; i++ {
		istr := strconv.Itoa(i)
		// Mix string and integer attributes:
		if i%2 == 0 {
			attributes[istr] = istr
		} else {
			attributes[istr] = i
		}
	}
	return attributes
}

func BenchmarkAttributes(b *testing.B) {
	attributes := Attributes(sampleAttributes(1000))

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// Compare this to: `js, err := json.Marshal(attributes)`
		buf := &bytes.Buffer{}
		attributes.WriteJSON(buf)
		if 0 == buf.Len() {
			b.Fatal(buf.Len())
		}
	}
}

func BenchmarkOrderedAttributes(b *testing.B) {
	attributes := sampleAttributes(1000)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// Compare this to: `js, err := json.Marshal(attributes)`
		js := MarshalOrderedAttributes(attributes)
		if len(js) == 0 {
			b.Fatal(string(js))
		}
	}
}
