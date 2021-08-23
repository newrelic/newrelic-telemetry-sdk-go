// Copyright 2019 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package telemetry

import (
	"bytes"
	"io/ioutil"
	"testing"
	"time"

	"github.com/newrelic/newrelic-telemetry-sdk-go/internal"
)

func BenchmarkLogsJSON(b *testing.B) {
	// This benchmark tests the overhead of turning logs into JSON.
	group := &logGroup{}
	numLogs := 10 * 1000
	tm := time.Date(2014, time.November, 28, 1, 1, 0, 0, time.UTC)

	for i := 0; i < numLogs; i++ {
		group.Logs = append(group.Logs, Log{
			Message:   "This is a log message.",
			Timestamp: tm,
		})
	}

	b.ResetTimer()
	b.ReportAllocs()

	buf := &bytes.Buffer{}
	for i := 0; i < b.N; i++ {
		buf.Reset()
		if group.WriteDataEntry(buf); nil == buf.Bytes() || len(buf.Bytes()) == 0 {
			b.Fatal(buf.String())
		}
	}
}

func testHarvesterLogs(t testing.TB, h *Harvester, expect string) {
	reqs := h.swapOutLogs()
	if nil == reqs {
		if expect != "null" {
			t.Error("nil logs", expect)
		}
		return
	}
	if len(reqs) != 1 {
		t.Fatal(reqs)
	}
	if u := reqs[0].URL.String(); u != defaultLogURL {
		t.Fatal(u)
	}
	bodyReader, _ := reqs[0].GetBody()
	compressedBytes, _ := ioutil.ReadAll(bodyReader)
	uncompressedBytes, _ := internal.Uncompress(compressedBytes)
	js := string(uncompressedBytes)
	actual := string(js)
	if th, ok := t.(interface{ Helper() }); ok {
		th.Helper()
	}
	compactExpect := compactJSONString(expect)
	if compactExpect != actual {
		t.Errorf("\nexpect=%s\nactual=%s\n", compactExpect, actual)
	}
}

func TestLog(t *testing.T) {
	tm := time.Date(2014, time.November, 28, 1, 1, 0, 0, time.UTC)
	h, _ := NewHarvester(configTesting)
	h.RecordLog(Log{
		Message:   "This is a log message.",
		Timestamp: tm,
		Attributes: map[string]interface{}{
			"zip": "zap",
		},
	})
	expect := `[{"logs":[{
		"message":"This is a log message.",
		"timestamp":1417136460000,
		"attributes": {
			"zip":"zap"
		}
	}]}]`
	testHarvesterLogs(t, h, expect)
}

func TestLogInvalidAttribute(t *testing.T) {
	tm := time.Date(2014, time.November, 28, 1, 1, 0, 0, time.UTC)
	h, _ := NewHarvester(configTesting)
	h.RecordLog(Log{
		Message:   "This is a log message.",
		Timestamp: tm,
		Attributes: map[string]interface{}{
			"weird-things-get-turned-to-strings": struct{}{},
			"nil-gets-removed":                   nil,
		},
	})
	expect := `[{"logs":[{
		"message":"This is a log message.",
		"timestamp":1417136460000,
		"attributes": {
			"weird-things-get-turned-to-strings":"struct {}"
		}
	}]}]`
	testHarvesterLogs(t, h, expect)
}

func TestRecordLogNilHarvester(t *testing.T) {
	tm := time.Date(2014, time.November, 28, 1, 1, 0, 0, time.UTC)
	var h *Harvester
	err := h.RecordLog(Log{
		Message:   "This is a log message.",
		Timestamp: tm,
		Attributes: map[string]interface{}{
			"zip": "zap",
			"zop": 123,
		},
	})
	if err != nil {
		t.Error(err)
	}
}

func BenchmarkLogCommonBlock(b *testing.B) {
	block, err := NewLogCommonBlock(WithLogAttributes(map[string]interface{}{"zup": "wup"}))
	if err != nil {
		b.Fatal(err)
	}

	buf := &bytes.Buffer{}

	for i := 0; i<b.N; i++ {
		buf.Reset()
		buf.WriteString(block.DataTypeKey())
		block.WriteDataEntry(buf)
	}
}
