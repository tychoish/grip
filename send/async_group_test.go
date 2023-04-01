package send

import (
	"context"
	"testing"
	"time"

	"github.com/tychoish/fun/testt"
	"github.com/tychoish/grip/level"
	"github.com/tychoish/grip/message"
)

func NewString(l level.Priority, in string) message.Composer {
	m := message.MakeString(in)
	m.SetPriority(l)
	return m
}
func NewSimpleString(l level.Priority, in string) message.Composer {
	m := message.MakeSimpleString(in)
	m.SetPriority(l)
	return m
}

func TestAsyncGroupSender(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cs := MakeStdError()
	cs.SetPriority(level.Notice)

	s := MakeAsyncGroup(ctx, 2, cs)
	impl, ok := s.(*asyncGroupSender)
	if !ok {
		t.Fatalf("%T", s)
	}

	newLevel := level.Alert
	if newLevel == s.Priority() {
		t.Error("elements should not be equal")
	}
	impl.priority.Set(newLevel)
	if newLevel != s.Priority() {
		t.Error("elements shold be equal")
	}

	s.Send(NewString(level.Debug, "hello"))
	newLevel = level.Alert
	s.SetPriority(newLevel)
	if newLevel != s.Priority() {
		t.Error("elements shold be equal")
	}

	if err := s.Flush(testt.ContextWithTimeout(t, time.Second)); err != nil {
		t.Error(err)
	}
	if err := s.Close(); err != nil {
		t.Error(err)
	}
}
