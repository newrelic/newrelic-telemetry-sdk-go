// Copyright 2019 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package telemetry

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
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

// vetAttributes returns the attributes that are valid.  vetAttributes does not
// modify remove any elements from its parameter.
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
	return validAttributes, errors.New(strings.Join(errStrs, ","))
}

type commonAttributes struct {
	RawJSON json.RawMessage
}

func (ca *commonAttributes) Type() string {
	return "attributes"
}

func (ca *commonAttributes) Bytes() []byte {
	return ca.RawJSON
}

func newCommonAttributes(attributes map[string]interface{}) (*commonAttributes, error) {
	attrs, err := vetAttributes(attributes)
	if err != nil {
		return nil, err
	}
	attributesJSON, err := json.Marshal(attrs)
	if err != nil {
		return nil, err
	}
	return &commonAttributes{RawJSON: attributesJSON}, nil
}
