package send

import (
	"context"
	"testing"
	"time"

	"github.com/tychoish/fun"
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
	if err := cs.SetLevel(LevelInfo{Default: level.Debug, Threshold: level.Notice}); err != nil {
		t.Error(err)
	}

	s := MakeAsyncGroup(ctx, 2, cs)
	impl, ok := s.(*asyncGroupSender)
	if !ok {
		t.Fatalf("%T", s)
	}

	newLevel := LevelInfo{Default: level.Notice, Threshold: level.Alert}
	if newLevel == s.Level() {
		t.Error("elements should not be equal")
	}
	impl.level.Set(newLevel)
	if newLevel != s.Level() {
		t.Error("elements shold be equal")
	}

	s.Send(NewString(level.Debug, "hello"))
	newLevel = LevelInfo{Default: level.Debug, Threshold: level.Alert}
	if err := impl.SetLevel(newLevel); err != nil {
		t.Error(err)
	}
	if newLevel != s.Level() {
		t.Error("elements shold be equal")
	}

	if err := s.SetLevel(LevelInfo{}); err == nil {
		t.Error("should error")
	} else {
		errs := fun.Unwind(err)
		if len(errs) != 1 {
			t.Error(len(errs), errs)
		}
	}

	if err := s.Flush(testt.ContextWithTimeout(t, time.Second)); err != nil {
		t.Error(err)
	}
	if err := s.Close(); err != nil {
		t.Error(err)
	}
}
