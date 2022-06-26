package github

import (
	"testing"

	"github.com/tychoish/grip/level"
)

func TestStatus(t *testing.T) {
	c := NewStatusMessage(level.Info, "example", StatePending, "https://example.com/hi", "description")
	if c == nil {
		t.Fatal("message should not be nil")
	}
	if !c.Loggable() {
		t.Error("should be true")
	}

	raw, ok := c.Raw().(*Status)
	if !ok {
		t.Error("should be true")
	}

	func() {
		defer func() {
			if p := recover(); p != nil {
				t.Errorf("should not have panic'd %q", p)
			}

		}()
		if raw.Context != "example" {
			t.Error("elements should be equal")
		}
		if raw.State != StatePending {
			t.Error("elements should be equal")
		}
		if raw.URL != "https://example.com/hi" {
			t.Error("elements should be equal")
		}
		if raw.Description != "description" {
			t.Error("elements should be equal")
		}
	}()

	if c.String() != "example pending: description (https://example.com/hi)" {
		t.Error("elements should be equal")
	}
}

func TestStatusInvalidStatusesAreNotLoggable(t *testing.T) {
	c := NewStatusMessage(level.Info, "", StatePending, "https://example.com/hi", "description")
	if c.Loggable() {
		t.Error("should be false")
	}
	c = NewStatusMessage(level.Info, "example", "nope", "https://example.com/hi", "description")
	if c.Loggable() {
		t.Error("should be false")
	}
	c = NewStatusMessage(level.Info, "example", StatePending, ":foo", "description")
	if c.Loggable() {
		t.Error("should be false")
	}

	p := Status{
		Owner:       "",
		Repo:        "grip",
		Ref:         "master",
		Context:     "example",
		State:       StatePending,
		URL:         "https://example.com/hi",
		Description: "description",
	}
	c = NewStatusMessageWithRepo(level.Info, p)
	if c.Loggable() {
		t.Error("should be false")
	}

	p.Owner = "tychoish"
	p.Repo = ""
	c = NewStatusMessageWithRepo(level.Info, p)
	if c.Loggable() {
		t.Error("should be false")
	}

	p.Repo = "grip"
	p.Ref = ""
	c = NewStatusMessageWithRepo(level.Info, p)
	if c.Loggable() {
		t.Error("should be false")
	}
}
