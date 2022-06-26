package send

import (
	"context"
	"testing"

	"github.com/tychoish/grip/level"
	"github.com/tychoish/grip/message"
)

func TestMultiSenderRespectsLevel(t *testing.T) {
	// this is a limited test to prevent level filtering to behave
	// differently than expected

	mock, err := NewInternalLogger("mock", LevelInfo{Default: level.Critical, Threshold: level.Alert})
	if err != nil {
		t.Error(err)
	}
	s := MakeStdError()
	s.SetName("mock2")
	multi := MakeMulti(s, mock)

	if 0 != mock.Len() {
		t.Error("elements should be equal")
	}
	multi.Send(message.NewString(level.Info, "hello"))
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

	multi.Send(message.NewString(level.Alert, "hello"))
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

	multi.Send(message.NewString(level.Alert, "hello"))
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
