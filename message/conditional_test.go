package message

import (
	"testing"
)

func TestConditionalMessage(t *testing.T) {
	comp := When(true, "foo")
	if !comp.Loggable() {
		t.Error("value should be true")
	}

	comp = When(false, "foo")
	if comp.Loggable() {
		t.Error("value should be false")
	}
	comp = When(true, "")
	if comp.Loggable() {
		t.Errorf("%T: %s", comp.(*condComposer).msg, comp.(*condComposer).msg)
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
		t.Errorf("%T: %s", comp.(*condComposer).msg, comp.(*condComposer).msg)
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
	if comp.Loggable() {
		t.Errorf("%T: %s", comp.(*condComposer).msg, comp.(*condComposer).msg)
	}

	comp = WhenMsg(true, "foo")
	if !comp.Loggable() {
		t.Error("value should be true")
	}
	comp = WhenMsg(false, "bar")
	if comp.Loggable() {
		t.Error("value should be false")
	}
}
