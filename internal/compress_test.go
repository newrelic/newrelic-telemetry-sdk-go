// Copyright 2019 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package internal

import "testing"

func TestCompress(t *testing.T) {
	input := "this is the input string that needs to be compressed"
	buf, err := Compress([]byte(input))
	if err != nil {
		t.Fatal(err)
	}
	back, err := Uncompress(buf.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if string(back) != input {
		t.Error(string(back))
	}
}
