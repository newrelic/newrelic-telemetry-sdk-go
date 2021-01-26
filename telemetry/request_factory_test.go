package telemetry

import "testing"

func TestNewRequestFactoryNoInsertKeyConfigSuccess(t *testing.T) {
	f, err := NewRequestFactory(WithNoDefaultKey())
	if f == nil {
		t.Error("Factory was not created")
	}
	if err != nil {
		t.Error(err)
	}
}

func TestNewRequestFactoryInsertKeyConfigSuccess(t *testing.T) {
	f, err := NewRequestFactory(WithInsertKey("key!"))
	if f == nil {
		t.Error("Factory was not created")
	}
	if err != nil {
		t.Error(err)
	}
}

func TestNewRequestFactoryNoKeyFail(t *testing.T) {
	f, err := NewRequestFactory()
	if f != nil {
		t.Error("Factory was created without a specified api key or mode")
	}

	if err == nil {
		t.Error("Expected an error, but one was not generated.")
	}
}