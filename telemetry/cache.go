// Copyright 2019 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package telemetry

import (
	"bytes"
	"encoding/json"
)

type cachedMapEntry struct {
	key  string
	data json.RawMessage
}

var _ = MapEntry(cachedMapEntry{})

func (c cachedMapEntry) DataTypeKey() string {
	return c.key
}

func (c cachedMapEntry) WriteDataEntry(buf *bytes.Buffer) *bytes.Buffer {
	buf.Write(c.data)
	return buf
}

func newCachedMapEntry(e MapEntry) *cachedMapEntry {
	buf := &bytes.Buffer{}
	e.WriteDataEntry(buf)
	return &cachedMapEntry{
		key:  e.DataTypeKey(),
		data: buf.Bytes(),
	}
}
