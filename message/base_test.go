package message

import (
	"testing"
)

func TestCollectWorksWithUnsetPids(t *testing.T) {
	base := &Base{}
	if !base.IsZero() {
		t.Fatal("base must be zero on init")
	}
	if base.Hostname != "" {
		t.Error("values should be equal")
	}
	base.Pid = 0
	base.Collect()
	if base.Hostname == "" {
		t.Error("hostname should be populated")
	}
}

func TestCollectNoopsIfPidIsSet(t *testing.T) {
	base := &Base{}
	if !base.IsZero() {
		t.Fatal("base must be zero on init")
	}
	if base.Hostname != "" {
		t.Error("values should be equal")
	}
	base.Pid = 1
	base.Collect()
	if base.Hostname != "" {
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
	if err := base.Annotate("k", "foo"); err != nil {
		t.Fatal(err)
	}
	if base.Context == nil {
		t.Error("context should be populated")
	}
}

func TestAnnotateErrorsForSameValue(t *testing.T) {
	base := &Base{}
	if !base.IsZero() {
		t.Fatal("base must be zero on init")
	}
	if err := base.Annotate("k", "foo"); err != nil {
		t.Fatal(err)
	}
	if err := base.Annotate("k", "foo"); err == nil {
		t.Error("error should not be nil")
	}
	if base.Context["k"] != "foo" {
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
	if err := base.Annotate("kOne", "foo"); err != nil {
		t.Fatal(err)
	}
	if err := base.Annotate("kTwo", "foo"); err != nil {
		t.Fatal(err)
	}
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
