package recovery

import (
	"fmt"
	"testing"
)

func TestPanicStringConverter(t *testing.T) {
	if panicString(nil) != "" {
		t.Error("elements should be equal")
	}
	if panicString("foo") != "foo" {
		t.Error("elements should be equal")
	}
	if panicString(fmt.Errorf("foo")) != "foo" {
		t.Error("elements should be equal")
	}
}

func TestPanicErrorHandler(t *testing.T) {
	if err := panicError(nil); err != nil {
		t.Error(err)
	}
	if err := panicError("foo"); err == nil {
		t.Error("error should not be nil")
	}
	if err := panicError(""); err == nil {
		t.Error("error should not be nil")
	}
}
