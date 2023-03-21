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
	if err := cs.SetLevel(LevelInfo{Default: level.Debug, Threshold: level.Notice}); err != nil {
		t.Error(err)
	}
	if !cs.Level().Valid() {
		t.Fail()
	}

	s := NewAsyncGroup(ctx, 2, cs)

	// if it's not valid to start with then we shouldn't make it
	// valid by setting it to avoid constituents being overridden,
	if s.Level().Valid() {
		t.Fail()
	}
	if err := s.SetLevel(LevelInfo{Default: level.Info, Threshold: level.Alert}); err != nil {
		t.Error(err)
	}
	if s.Level().Valid() {
		t.Fail()
	}
	if !cs.Level().Valid() {
		t.Fail()
	}

	impl, ok := s.(*asyncGroupSender)
	if !ok {
		t.Fail()
	}
	newLevel := LevelInfo{Default: level.Info, Threshold: level.Alert}
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

	if err := s.Flush(testt.ContextWithTimeout(t, time.Second)); err != nil {
		t.Error(err)
	}
	if err := s.Close(); err != nil {
		t.Error(err)
	}
}
