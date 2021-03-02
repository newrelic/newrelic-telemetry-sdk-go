package telemetry

import (
	"compress/gzip"
	"github.com/newrelic/newrelic-telemetry-sdk-go/internal"
	"io/ioutil"
	"testing"
)

func TestNewRequestFactoryNoInsertKeyConfigSuccess(t *testing.T) {
	f, err := NewSpanRequestFactory(WithNoDefaultKey())
	if f == nil {
		t.Error("Factory was not created")
	}
	if err != nil {
		t.Error(err)
	}
}

func TestNewRequestFactoryInsertKeyConfigSuccess(t *testing.T) {
	f, err := NewSpanRequestFactory(WithInsertKey("key!"))
	if f == nil {
		t.Error("Factory was not created")
	}
	if err != nil {
		t.Error(err)
	}
}

func TestNewRequestFactoryNoKeyFail(t *testing.T) {
	f, err := NewSpanRequestFactory()
	if f != nil {
		t.Error("Factory was created without a specified api key or mode")
	}

	if err == nil {
		t.Error("Expected an error, but one was not generated.")
	}
}

func TestClientOptions(t *testing.T) {
	tests := []struct {
		name   string
		option ClientOption
	}{
		{name: "WithInsertKey", option: WithInsertKey("blahblah")},
		{name: "WithNoDefaultKey", option: WithNoDefaultKey()},
		{name: "WithEndpoint", option: WithEndpoint("localhost")},
		{name: "WithUserAgent", option: WithUserAgent("secret-agent")},
		{name: "WithInsecure", option: WithInsecure()},
		{name: "WithGzipCompressionLevel-bad", option: WithGzipCompressionLevel(9000)},
		{name: "WithGzipCompressionLevel-good", option: WithGzipCompressionLevel(gzip.BestCompression)},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			NewSpanRequestFactory(test.option)
		})
	}
}

type MockPayloadEntry struct{}

func (m *MockPayloadEntry) Type() string {
	return "spans"
}

func (m *MockPayloadEntry) Bytes() []byte {
	return []byte{'[', ']'}
}

func TestSpanFactoryRequest(t *testing.T) {
	f, _ := NewSpanRequestFactory(WithInsertKey("key!"))
	request, _ := f.BuildRequest([]PayloadEntry{&MockPayloadEntry{}})
	if request.Method != "POST" {
		t.Error("Method was not POST")
	}
	if request.URL.String() != "https://trace-api.newrelic.com/trace/v1" {
		t.Error("URL is wrong!")
	}
	bytes, err := ioutil.ReadAll(request.Body)
	if err != nil {
		t.Error("Unable to read request body")
	}
	bytes, err = internal.Uncompress(bytes)
	if err != nil {
		t.Error("Error decompressing request body")
	}

	body := string(bytes[:])
	if body != "[{\"spans\":[]}]" {
		t.Error("Body is wrong")
	}

	if request.Header.Get("Content-Type") != "application/json" {
		t.Error("Missing content-type header")
	}

	if request.Header.Get("Api-Key") != "key!" {
		t.Error("Incorrect api key header")
	}

	if request.Header.Get("User-Agent") != defaultUserAgent {
		t.Error("Incorrect user agent")
	}

	if request.Header.Get("Content-Encoding") != "gzip" {
		t.Error("Content-Encoding header must be gzip")
	}
}
