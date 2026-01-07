package zerolog

import (
	"encoding/json"
	"iter"

	"github.com/rs/zerolog"
	"github.com/tychoish/fun/irt"
	"github.com/tychoish/grip/level"
	"github.com/tychoish/grip/message"
	"github.com/tychoish/grip/send"
)

type shim struct {
	zl zerolog.Logger
	send.Base
}

// MakeSender constructs a sender object as NewSender but without the
// error type or level configuration, consistent with other grip
// sender constructors.
func MakeSender(zl zerolog.Logger) send.Sender {
	s := &shim{zl: zl}
	s.SetPriority(level.Trace)
	return s
}

func convertLevel(in level.Priority) zerolog.Level {
	switch in {
	case level.Emergency:
		return zerolog.ErrorLevel
	case level.Alert:
		return zerolog.ErrorLevel
	case level.Critical:
		return zerolog.ErrorLevel
	case level.Error:
		return zerolog.ErrorLevel
	case level.Warning:
		return zerolog.WarnLevel
	case level.Notice:
		return zerolog.InfoLevel
	case level.Info:
		return zerolog.InfoLevel
	case level.Debug:
		return zerolog.DebugLevel
	case level.Trace:
		return zerolog.DebugLevel
	case level.Invalid:
		return zerolog.Disabled
	default:
		return zerolog.NoLevel
	}
}

func (s *shim) Send(m message.Composer) {
	if !send.ShouldLog(s, m) {
		return
	}
	// unwind group messages
	if grp, ok := m.(*message.GroupComposer); ok {
		for _, msg := range grp.Messages() {
			s.Send(msg)
		}
		return
	}

	event := s.zl.WithLevel(convertLevel(m.Priority()))
	if !m.Structured() {
		out, err := s.Format(m)
		if !s.HandleErrorOK(send.WrapError(err, m)) {
			return
		}
		event.Msg(out)
		return
	}

	addField := func(k string, v any) { event.Fields([]any{k, v}) }
	// handle payloads to take advantage of the fast paths
	payload := m.Raw()
	switch data := payload.(type) {
	case zerolog.LogObjectMarshaler:
		event.EmbedObject(data)
	case zerolog.LogArrayMarshaler:
		event.Array("payload", data)
	case iter.Seq2[string, any]:
		irt.Apply2(data, addField)
	case iter.Seq[irt.KV[string, any]]:
		irt.Apply2(irt.KVsplit(data), addField)
	case []irt.KV[string, any]:
		irt.Apply2(irt.KVsplit(irt.Slice(data)), addField)
	case message.Fields:
		irt.Apply2(irt.Map(data), addField)
	case map[string]any:
		irt.Apply2(irt.Map(data), addField)
	case json.Marshaler:
		// message.KVs are json.Marshalers so make sure this
		// clause stays last.
		r, err := data.MarshalJSON()
		if !s.HandleErrorOK(send.WrapError(err, m)) {
			return
		}
		event.RawJSON("payload", r)
	default:
		// in most cases this uses json.Marshler and
		// reflection, so this will end up being a slower path
		// than any of the above paths.
		event.Interface("payload", payload)
	}
	event.Send()
}
