package send

import (
	"testing"

	"github.com/tychoish/grip/level"
	"github.com/tychoish/grip/message"
)

func TestInterceptor(t *testing.T) {
	base, err := NewInternalLogger("test", LevelInfo{Default: level.Info, Threshold: level.Debug})
	if err != nil {
		t.Fatal(err)
	}

	var count int
	filter := func(m message.Composer) { count++ }

	icept := MakeFilter(base, filter)

	if base.Len() != 0 {
		t.Error("elements should be equal")
	}
	icept.Send(NewSimpleString(level.Info, "hello"))
	if base.Len() != 1 {
		t.Error("elements should be equal")
	}
	if count != 1 {
		t.Error("elements should be equal")
	}

	icept.Send(NewSimpleString(level.Trace, "hello"))
	if base.Len() != 2 {
		t.Error("elements should be equal")
	}
	if count != 2 {
		t.Error("elements should be equal")
	}
}
