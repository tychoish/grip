package message

import (
	"testing"

	"github.com/tychoish/grip/level"
)

func TestConditionalMessage(t *testing.T) {
	comp := When(true, "foo")
	if !comp.Loggable() {
		t.Error("value should be true")
	}

	comp.SetPriority(level.Error)
	if comp.Priority() != level.Error {
		t.Error(comp.Priority())
	}

	if comp.Structured() {
		t.Error(comp.(*conditional).constructor())
	}

	comp = When(false, "foo")
	if comp.Loggable() {
		t.Error("value should be false")
	}
	comp = When(true, "")
	if comp.Loggable() {
		val := comp.(*conditional).constructor()
		t.Errorf("%T: %s", val, val)
	}

	comp = Whenln(true, "foo", "bar")
	if !comp.Loggable() {
		t.Error("value should be true")
	}
	comp = Whenln(false, "foo", "bar")
	if comp.Loggable() {
		t.Error("value should be false")
	}
	comp = Whenln(true, "", "")
	if comp.Loggable() {
		t.Errorf("%T: %s, %q", comp, comp, comp.String())
	}

	comp = Whenf(true, "f%soo", "bar")
	if !comp.Loggable() {
		t.Error("value should be true")
	}
	comp = Whenf(false, "f%soo", "bar")
	if comp.Loggable() {
		t.Error("value should be false")
	}
	comp = Whenf(true, "", "foo")
	if !comp.Loggable() {
		t.Errorf("%T: %s", comp, comp)
	}

	comp = WhenStr(true, "foo")
	if !comp.Loggable() {
		t.Error("value should be true")
	}
	comp = WhenStr(false, "bar")
	if comp.Loggable() {
		t.Error("value should be false")
	}
}
