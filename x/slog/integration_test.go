package slog_test

import (
	"context"
	"log/slog"
	"testing"

	"github.com/tychoish/grip"
	"github.com/tychoish/grip/level"
	"github.com/tychoish/grip/message"
	"github.com/tychoish/grip/send"
	slogx "github.com/tychoish/grip/x/slog"
)

// captureHandler is a minimal slog.Handler used only in tests. It captures
// every slog.Record it receives so the test can assert on the emitted level,
// message, and number of log events.
type captureHandler struct{ records []slog.Record }

func (h *captureHandler) Enabled(_ context.Context, _ slog.Level) bool { return true }
func (h *captureHandler) Handle(_ context.Context, r slog.Record) error {
	h.records = append(h.records, r)
	return nil
}
func (h *captureHandler) WithAttrs(_ []slog.Attr) slog.Handler { return h }
func (h *captureHandler) WithGroup(_ string) slog.Handler      { return h }

func TestGripIntegration(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		Name         string
		Composer     message.Composer
		ExpectLevel  slog.Level
		ExpectString string
	}{
		{
			Name:         "Info/Unstructured",
			Composer:     func() message.Composer { c := message.MakeString("hi"); c.SetPriority(level.Info); return c }(),
			ExpectLevel:  slog.LevelInfo,
			ExpectString: "hi",
		},
		{
			Name: "Warning/Structured",
			Composer: func() message.Composer {
				c := message.MakeFields(map[string]any{"x": 1})
				c.SetPriority(level.Warning)
				return c
			}(),
			ExpectLevel:  slog.LevelWarn,
			ExpectString: "",
		},
	} {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			ctx := t.Context()

			h := &captureHandler{}
			logger := slog.New(h)
			s := slogx.MakeSender(ctx, logger)

			l := grip.NewLogger(send.Sender(s))

			l.Log(tc.Composer.Priority(), tc.Composer)

			if len(h.records) != 1 {
				t.Log(h.records)
				t.Fatalf("expected 1 record, got %d", len(h.records))
			}
			rec := h.records[0]
			if rec.Level != tc.ExpectLevel {
				t.Errorf("expected level %v, got %v", tc.ExpectLevel, rec.Level)
			}
			if rec.Message != tc.ExpectString {
				t.Errorf("expected message %q, got %q", tc.ExpectString, rec.Message)
			}
		})
	}
}
