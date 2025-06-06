package slog_test

import (
	"context"
	"errors"
	"testing"

	"log/slog"

	"github.com/tychoish/grip/level"
	"github.com/tychoish/grip/message"
	slogx "github.com/tychoish/grip/x/slog"
)

type record struct {
	Level   slog.Level
	Message string
	Attrs   []slog.Attr
}

// ----------------------------------------------------------------------
// Existing tests (updated to std slog)
// ----------------------------------------------------------------------

func TestUnstructured(t *testing.T) {
	ctx := t.Context()
	h := &captureHandler{}
	sender := slogx.MakeSender(ctx, slog.New(h)) // new signature

	msg := message.MakeString("hello world")
	msg.SetPriority(level.Info)
	sender.Send(msg)

	if len(h.records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(h.records))
	}
	r := h.records[0]
	if r.Level != slog.LevelInfo {
		t.Errorf("expected level Info, got %v", r.Level)
	}
	if r.Message != "hello world" {
		t.Errorf("expected message %q, got %q", "hello world", r.Message)
	}
	if r.NumAttrs() != 0 {
		t.Errorf("expected no attributes, got %v", r.NumAttrs())
	}
}

func TestStructuredFields(t *testing.T) {
	ctx := t.Context()
	h := &captureHandler{}
	sender := slogx.MakeSender(ctx, slog.New(h))

	fields := message.Fields{"foo": "bar", "baz": 42}
	msg := message.MakeFields(fields)
	msg.SetPriority(level.Debug)
	sender.Send(msg)

	if len(h.records) != 1 {
		t.Log(h.records)
		t.Fatalf("expected 1 record, got %d", len(h.records))
	}
	r := h.records[0]
	if r.Level != slog.LevelDebug {
		t.Errorf("expected level Debug, got %v", r.Level)
	}
	if r.Message != "" {
		t.Errorf("expected empty message, got %q", r.Message)
	}
	got := map[string]any{}
	r.Attrs(func(a slog.Attr) bool {
		got[a.Key] = a.Value.Any()
		return true
	})

	if got["foo"] != "bar" || got["baz"] != int64(42) {
		t.Errorf("unexpected attrs: %v", got)
	}
}

func TestStructuredError(t *testing.T) {
	ctx := t.Context()
	h := &captureHandler{}
	sender := slogx.MakeSender(ctx, slog.New(h))

	err := errors.New("something bad")
	c := &errorComposer{err: err}
	c.SetPriority(level.Alert)
	sender.Send(c)

	if len(h.records) != 1 {
		t.Log(h.records)
		t.Fatalf("expected 1 record, got %d", len(h.records))
	}
	r := h.records[0]
	if r.Level != slog.LevelError {
		t.Errorf("expected level Error, got %v", r.Level)
	}
	if h.records[0].NumAttrs() != 1 {
		t.Fatalf("expected exactly 1 attribute, got %d", h.records[0].NumAttrs())
	}
	got := map[string]any{}
	r.Attrs(func(a slog.Attr) bool {
		got[a.Key] = a.Value.Any()
		return true
	})
	if got["error"] != err {
		t.Errorf("expected error attr %v, got %v", err, got["error"])
	}
}

// ----------------------------------------------------------------------
// New tests
// ----------------------------------------------------------------------

func TestPairBuilder(t *testing.T) {
	ctx := t.Context()
	h := &captureHandler{}
	s := slogx.MakeSender(ctx, slog.New(h))

	builder := &message.PairBuilder{}
	builder.Pair("alpha", 1).
		Pair("beta", true).
		Pair("gamma", 3.14).
		Level(level.Notice)

	s.Send(builder)

	if len(h.records) != 1 {
		t.Log(h.records)
		t.Fatalf("expected single record, got %d", len(h.records))
	}
	r := h.records[0]
	if r.Level != slog.LevelInfo { // level.Notice maps to Info
		t.Errorf("expected Info, got %v", r.Level)
	}
	got := make(map[string]any)
	r.Attrs(func(a slog.Attr) bool {
		got[a.Key] = a.Value.Any()
		return true
	})

	want := map[string]any{"alpha": int64(1), "beta": true, "gamma": 3.14}
	for k, v := range want {
		if got[k] != v {
			t.Logf("k=%s[%T] got[k]=%T, v=%T", k, k, got[k], v)
			t.Errorf("attr %s: expected %v, got %v", k, v, got[k])
		}
	}
}

// Level mapping table for convenience in fidelity test.
var levelTable = []struct {
	grip level.Priority
	slog slog.Level
}{
	{level.Trace, slog.LevelDebug},
	{level.Debug, slog.LevelDebug},
	{level.Info, slog.LevelInfo},
	{level.Notice, slog.LevelInfo},
	{level.Warning, slog.LevelWarn},
	{level.Error, slog.LevelError},
	{level.Alert, slog.LevelError},
	{level.Emergency, slog.LevelError},
}

func TestLevelFidelity(t *testing.T) {
	ctx := t.Context()
	h := &captureHandler{}
	s := slogx.MakeSender(ctx, slog.New(h))

	for _, pair := range levelTable {
		msg := message.MakeString(pair.grip.String())
		msg.SetPriority(pair.grip)
		s.Send(msg)
	}

	if len(h.records) != len(levelTable) {
		t.Fatalf("expected %d records, got %d", len(levelTable), len(h.records))
	}
	for i, pair := range levelTable {
		if got := h.records[i].Level; got != pair.slog {
			t.Errorf("entry %d: grip=%s expected slog=%v, got %v", i, pair.grip, pair.slog, got)
		}
	}
}

func TestLevelFiltering(t *testing.T) {
	ctx := t.Context()
	h := &captureHandler{}
	s := slogx.MakeSender(ctx, slog.New(h))
	s.SetPriority(level.Info) // filter out Debug and Trace

	traceMsg := message.MakeString("ignore me")
	traceMsg.SetPriority(level.Trace)
	s.Send(traceMsg)

	infoMsg := message.MakeString("process me")
	infoMsg.SetPriority(level.Info)
	s.Send(infoMsg)

	if len(h.records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(h.records))
	}
	if h.records[0].Message != "process me" {
		t.Errorf("expected %q, got %q", "process me", h.records[0].Message)
	}
}

// ----------------------------------------------------------------------
// helpers
// ----------------------------------------------------------------------

// errorComposer implements message.Composer for raw error payloads.
type errorComposer struct {
	message.Base
	err error
}

func (c *errorComposer) Structured() bool { return true }
func (c *errorComposer) Raw() any         { return c.err }
func (c *errorComposer) Loggable() bool   { return true }
func (c *errorComposer) String() string   { return "" }

// TestLazyResolution ensures that a composer that is only resolved lazily does
// not incur the cost of resolution when the slog handler has filtered it out.
func TestLazyResolution(t *testing.T) {
	ctx := t.Context()
	disabledHandler := slog.New(&noopHandler{}) // never Enabled
	s := slogx.MakeSender(ctx, disabledHandler)

	lazy := &countingComposer{}
	lazy.SetPriority(level.Debug)

	s.Send(lazy)

	if lazy.resolved {
		t.Fatal("composer was resolved even though handler dropped it")
	}
}

// ----------------------------------------------------------------------
// auxiliary helpers for lazy-resolution test
// ----------------------------------------------------------------------

type noopHandler struct{}

func (h *noopHandler) Enabled(_ context.Context, _ slog.Level) bool  { return false }
func (h *noopHandler) Handle(_ context.Context, _ slog.Record) error { return nil }
func (h *noopHandler) WithAttrs(_ []slog.Attr) slog.Handler          { return h }
func (h *noopHandler) WithGroup(_ string) slog.Handler               { return h }

type countingComposer struct {
	message.Base
	resolved bool
}

func (c *countingComposer) Loggable() bool   { return true }
func (c *countingComposer) Structured() bool { return false }
func (c *countingComposer) Raw() any {
	// Mark the composer as resolved when Raw is invoked to ensure
	// resolution accounting remains accurate regardless of the
	// code-path used by the sender.
	c.resolved = true
	return nil
}
func (c *countingComposer) String() string {
	c.resolved = true
	return "expensive"
}
