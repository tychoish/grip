package recovery

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
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
	assert := assert.New(t)

	if err := panicError(nil); err != nil {
		t.Error(err)
	}
	assert.Error(panicError("foo"))
	assert.Error(panicError(""))
}
