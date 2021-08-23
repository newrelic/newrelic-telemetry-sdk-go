// Copyright 2019 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package telemetry

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/newrelic/newrelic-telemetry-sdk-go/internal"
)

func attributeValueValid(val interface{}) bool {
	switch val.(type) {
	case string, bool, uint8, uint16, uint32, uint64, int8, int16,
		int32, int64, float32, float64, uint, int, uintptr:
		return true
	default:
		return false
	}
}

type errInvalidAttributes struct {
	msg string
}

func (e errInvalidAttributes) Error() string {
	return e.msg
}

// vetAttributes returns the attributes that are valid.  vetAttributes does not
// modify or remove any elements from its parameter.
func vetAttributes(attributes map[string]interface{}) (map[string]interface{}, error) {
	valid := true
	for _, val := range attributes {
		if !attributeValueValid(val) {
			valid = false
			break
		}
	}
	if valid {
		return attributes, nil
	}
	// Note that the map is only copied if elements are to be removed to
	// improve performance.
	validAttributes := make(map[string]interface{}, len(attributes))
	var errStrs []string
	for key, val := range attributes {
		if attributeValueValid(val) {
			validAttributes[key] = val
		} else {
			errStrs = append(errStrs, fmt.Sprintf(`attribute "%s" has invalid type %T`, key, val))
		}
	}
	return validAttributes, errInvalidAttributes{strings.Join(errStrs, ",")}
}

type commonAttributes struct {
	Attributes map[string]interface{}
}

func (ca *commonAttributes) DataTypeKey() string {
	return "attributes"
}

func (ca *commonAttributes) WriteDataEntry(buf *bytes.Buffer) *bytes.Buffer {
	w := internal.JSONFieldsWriter{Buf: buf}
	buf.WriteByte('{')
	internal.AddAttributes(&w, ca.Attributes)
	buf.WriteByte('}')
	return buf
}

// newCommonAttributes vets the attributes map. If invalid attributes are
// detected, the response will contain the valid attributes and an error describing which
// keys were invalid will be returned.
func newCommonAttributes(attributes map[string]interface{}) (*commonAttributes, error) {
	if len(attributes) == 0 {
		return nil, nil
	}
	validAttrs, err := vetAttributes(attributes)
	return &commonAttributes{
		Attributes: validAttrs,
	}, err
}
