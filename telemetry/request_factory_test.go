package telemetry

import "testing"

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