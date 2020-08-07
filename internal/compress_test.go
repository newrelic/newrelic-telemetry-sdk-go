// Copyright 2019 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0
// +build unit

package internal

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompress(t *testing.T) {
	t.Parallel()

	input := "this is the input string that needs to be compressed"
	buf, err := Compress([]byte(input))
	require.NoError(t, err)
	assert.NotNil(t, buf)

	back, err := Uncompress(buf.Bytes())
	require.NoError(t, err)
	assert.NotNil(t, back)

	assert.Equal(t, input, string(back))
}
