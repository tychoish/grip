// Package slog exposes a Grip sender that delegates log output to any
// std-library slog.Logger. It enables code written for Grip
// (github.com/tychoish/grip) to emit structured or unstructured records
// through slog without losing level fidelity or contextual data.
//
// # Filtering behaviour
//
// The Sender honours both its own grip-level Priority filter (evaluated by
// send.ShouldLog) and the slog.Logger's Handler-level filter
// (Handler().Enabled). The Grip filter always runs first; therefore
// decreasing the sender's Priority will not cause log events to bypass the
// slog.Handler, and increasing the Priority cannot force the Handler to
// accept records it would otherwise discard. In other words, the most
// restrictive filter between Grip and slog ultimately determines whether a
// record is emitted.
package slog

import (
	"context"
	"iter"
	"log/slog"
	"time"

	"github.com/tychoish/fun/irt"
	"github.com/tychoish/grip/level"
	"github.com/tychoish/grip/message"
	"github.com/tychoish/grip/send"
)

// MakeSender constructs a new send.Sender that wraps the provided
// slog.Logger and uses the supplied context for all log events.
func MakeSender(ctx context.Context, logger *slog.Logger) send.Sender {
	s := &sender{ctx: ctx, logger: logger}
	s.SetPriority(level.Trace) // capture all levels by default
	return s
}

type sender struct {
	ctx    context.Context
	logger *slog.Logger
	send.Base
}

// convertLevel maps grip's level.Priority to slog.Level.
func convertLevel(p level.Priority) slog.Level {
	switch {
	case p >= level.Error:
		return slog.LevelError
	case p >= level.Warning:
		return slog.LevelWarn
	case p >= level.Info:
		return slog.LevelInfo
	default:
		return slog.LevelDebug
	}
}

// Send implements the send.Sender interface.
//
// The method preserves Grip semantics while forwarding records to slog:
//   - honour sender-level filtering via send.ShouldLog
//   - short-circuit when the slog handler has disabled the requested level
//   - surface formatter and handler errors via HandleError / HandleErrorOK so
//     that upstream error counters and hooks remain consistent.
func (s *sender) Send(m message.Composer) {
	if !send.ShouldLog(s, m) {
		return
	}

	lvl := convertLevel(m.Priority())
	if !s.logger.Handler().Enabled(s.ctx, lvl) {
		// Early-out: if the underlying slog.Handler is disabled for this level,
		// skip all further processing to avoid unnecessary allocation and
		// formatting work.
		return
	}

	if m.Structured() {
		rec := slog.NewRecord(
			// TODO: derive timestamp from the input composer
			time.Now(),
			lvl,
			message.GetDefaultFieldsMessage(m, ""),
			0,
		)
		addAttrsFromPayload(s.ctx, &rec, m.Raw())

		if err := s.logger.Handler().Handle(s.ctx, rec); err != nil {
			s.HandleError(send.WrapError(err, m))
		}
		return
	}

	out, err := s.Format(m)
	if err != nil {
		s.HandleError(send.WrapError(err, m))
		return
	}

	rec := slog.NewRecord(time.Now(), lvl, out, 0)
	if err = s.logger.Handler().Handle(s.ctx, rec); err != nil {
		s.HandleError(send.WrapError(err, m))
	}
}

// ----------------------------------------------------------------------
// attribute helpers
// ----------------------------------------------------------------------

// addAttrsFromPayload enriches the slog.Record in-place with attributes that
// correspond to Grip’s structured payload formats.
func addAttrsFromPayload(ctx context.Context, rec *slog.Record, in any) {
	addField := makeAddAttr(rec)

	switch v := in.(type) {
	case nil:
		// Nothing to add for a nil payload.
		return
	case slog.Attr: // already an Attr
		rec.AddAttrs(v)
	case []slog.Attr:
		rec.AddAttrs(v...)
	case error:
		rec.Add(slog.Any("error", v))
	case []error:
		rec.Add(slog.Any("errors", v))
	case message.Fields: // alias of map[string]any
		irt.Apply(irt.Merge(irt.Map(v), slog.Any), addField)
	case map[string]any:
		irt.Apply(irt.Merge(irt.Map(v), slog.Any), addField)
	case iter.Seq2[string, any]:
		irt.Apply(irt.Merge(v, slog.Any), addField)
	case []irt.KV[string, any]:
		irt.Apply(irt.Merge(irt.KVsplit(irt.Slice(v)), slog.Any), addField)
	case iter.Seq[irt.KV[string, any]]:
		irt.Apply(irt.Merge(irt.KVsplit(v), slog.Any), addField)
	case *message.KV:
		addAttrsFromPayload(ctx, rec, v.Raw())
	case interface{ Iterator() iter.Seq2[string, any] }:
		irt.Apply(irt.Merge(v.Iterator(), slog.Any), addField)
	default:
		rec.Add(slog.Any("payload", in))
	}
}

func makeAddAttr(rec *slog.Record) func(slog.Attr) { return func(a slog.Attr) { addAttr(rec, a) } }
func addAttr(rec *slog.Record, attr slog.Attr)     { rec.Add(attr) }
