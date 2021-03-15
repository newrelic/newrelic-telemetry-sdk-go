// Copyright 2019 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package telemetry

import (
	"encoding/json"
	"fmt"
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
func vetAttributes(attributes map[string]interface{}, errorLogger func(map[string]interface{})) map[string]interface{} {
	valid := true
	for _, val := range attributes {
		if !attributeValueValid(val) {
			valid = false
			break
		}
	}
	if valid {
		return attributes
	}
	// Note that the map is only copied if elements are to be removed to
	// improve performance.
	validAttributes := make(map[string]interface{}, len(attributes))
	for key, val := range attributes {
		if attributeValueValid(val) {
			validAttributes[key] = val
		} else if nil != errorLogger {
			errorLogger(map[string]interface{}{
				"err": fmt.Sprintf(`attribute "%s" has invalid type %T`, key, val),
			})
		}
	}
	return validAttributes
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

func newCommonAttributes(attributes map[string]interface{}, errorLogger func(map[string]interface{})) *commonAttributes {
	attrs := vetAttributes(attributes, errorLogger)
	attributesJSON, err := json.Marshal(attrs)

	if err != nil {
		errorLogger(map[string]interface{}{
			"err":     err.Error(),
			"message": "error marshaling common attributes",
		})
		return nil
	}
	return &commonAttributes{RawJSON: attributesJSON}
}
