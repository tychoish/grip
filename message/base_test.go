package message

import (
	"testing"
)

func TestCollectWorksWithUnsetPids(t *testing.T) {
	base := &Base{}
	if !base.IsZero() {
		t.Fatal("base must be zero on init")
	}
	if base.Host != "" {
		t.Error("values should be equal")
	}
	base.Pid = 0
	base.Collect()
	if base.Host == "" {
		t.Error("hostname should be populated")
	}
}

func TestCollectNoopsIfPidIsSet(t *testing.T) {
	base := &Base{}
	if !base.IsZero() {
		t.Fatal("base must be zero on init")
	}
	if base.Host != "" {
		t.Error("values should be equal")
	}
	base.Pid = 1
	base.Collect()
	if base.Host != "" {
		t.Error("values should be equal")
	}
}

func TestAnnotateAddsFields(t *testing.T) {
	base := &Base{}
	if !base.IsZero() {
		t.Fatal("base must be zero on init")
	}
	if base.Context != nil {
		t.Fatal("context should not be populated yet")
	}

	base.Annotate("k", "foo")

	if base.Context == nil {
		t.Fatal("context should be populated")
	}
	if _, ok := base.Context["k"]; !ok {
		t.Error("annotate should have value", base.Context)
	}
}

func TestAnnotateErrorsForSameValue(t *testing.T) {
	base := &Base{}
	if !base.IsZero() {
		t.Fatal("base must be zero on init")
	}
	base.Annotate("k", "foo")
	base.Annotate("k", "bar")
	if base.Context["k"] != "bar" {
		t.Error("values should be equal")
	}
}

func TestAnnotateMultipleValues(t *testing.T) {
	base := &Base{}
	if !base.IsZero() {
		t.Fatal("base must be zero on init")
	}
	if base.Structured() {
		t.Fatal("should not be structured yet")

	}
	base.Annotate("kOne", "foo")
	base.Annotate("kTwo", "foo")
	if base.Context["kOne"] != "foo" {
		t.Error("values should be equal")
	}
	if base.Context["kTwo"] != "foo" {
		t.Error("values should be equal")
	}
	if !base.Structured() {
		t.Fatal("should be structured")
	}
}
