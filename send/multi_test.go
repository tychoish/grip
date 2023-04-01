package send

import (
	"context"
	"testing"

	"github.com/tychoish/grip/level"
)

func TestMultiSenderRespectsLevel(t *testing.T) {
	t.Parallel()

	// this is a limited test to prevent level filtering to behave
	// differently than expected

	mock := MakeInternalLogger()
	mock.SetName("mock")
	mock.SetPriority(level.Alert)
	s := MakeStdError()
	s.SetName("mock2")
	multi := MakeMulti(s, mock)

	if 0 != mock.Len() {
		t.Error("elements should be equal")
	}
	multi.Send(NewString(level.Info, "hello"))
	if 1 != mock.Len() {
		t.Error("elements should be equal")
	}
	m, ok := mock.GetMessageSafe()
	if !ok {
		t.Error("should be true")
	}
	if m.Logged {
		t.Error("should be false")
	}

	multi.Send(NewString(level.Alert, "hello"))
	if 1 != mock.Len() {
		t.Error("elements should be equal")
	}
	m, ok = mock.GetMessageSafe()
	if !ok {
		t.Error("should be true")
	}
	if !m.Logged {
		t.Error("should be true")
	}

	multi.Send(NewString(level.Alert, "hello"))
	if 1 != mock.Len() {
		t.Error("elements should be equal")
	}
	m, ok = mock.GetMessageSafe()
	if !ok {
		t.Error("should be true")
	}
	if !m.Logged {
		t.Error("should be true")
	}

	if err := multi.Flush(context.TODO()); err != nil {
		t.Error(err)
	}
}
