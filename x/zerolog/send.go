package zerolog

import (
	"encoding/json"
	"iter"
	"time"

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

	if event := s.zl.WithLevel(convertLevel(m.Priority())); event != nil {
		if !m.Structured() {
			out, err := s.Format(m)
			if !s.HandleErrorOK(send.WrapError(err, m)) {
				return
			}
			event.Msg(out)
			return
		}

		// handle payloads to take advantage of the fast paths
		switch data := m.Raw().(type) {
		case zerolog.LogObjectMarshaler:
			event.EmbedObject(data)
		case zerolog.LogArrayMarshaler:
			event.Array("payload", data)
		case string:
			event.Str("message", data)
		case iter.Seq2[string, any]:
			irt.Apply2(data, addFieldOp(event))
		case iter.Seq[irt.KV[string, any]]:
			irt.Apply2(irt.KVsplit(data), addFieldOp(event))
		case []irt.KV[string, any]:
			irt.Apply2(irt.KVsplit(irt.Slice(data)), addFieldOp(event))
		case message.Fields:
			irt.Apply2(irt.Map(data), addFieldOp(event))
		case map[string]any:
			irt.Apply2(irt.Map(data), addFieldOp(event))
		case interface{ Iterator() iter.Seq2[string, any] }:
			irt.Apply2(data.Iterator(), addFieldOp(event))
		case error:
			event.Err(data)
		case []error:
			event.Errs("errors", data)
		case json.Marshaler:
			event.RawJSON("payload", data)
		default:
			// in most cases this uses json.Marshler and
			// reflection, so this will end up being a slower path
			// than any of the above paths.
			addToEvent(event, "payload", data)
		}
		event.Send()
	}
}

func addFieldOp(event *zerolog.Event) func(string, any) {
	return func(key string, value any) { addToEvent(event, key, value) }
}

func addToEvent(event *zerolog.Event, key string, value any) {
	switch data := value.(type) {
	case zerolog.LogObjectMarshaler:
		event.EmbedObject(data)
	case zerolog.LogArrayMarshaler:
		event.Array(key, data)
	case iter.Seq2[string, any]:
		irt.Apply2(data, addFieldOp(event))
	case iter.Seq[irt.KV[string, any]]:
		irt.Apply2(irt.KVsplit(data), addFieldOp(event))
	case []irt.KV[string, any]:
		irt.Apply2(irt.KVsplit(irt.Slice(data)), addFieldOp(event))
	case message.Fields:
		irt.Apply2(irt.Map(data), addFieldOp(event))
	case map[string]any:
		irt.Apply2(irt.Map(data), addFieldOp(event))
	case string:
		event.Str(key, data)
	case error:
		event.Err(data)
	case []error:
		event.Errs(key, data)
	case int:
		event.Int(key, data)
	case int8:
		event.Int8(key, data)
	case int16:
		event.Int16(key, data)
	case int32:
		event.Int32(key, data)
	case int64:
		event.Int64(key, data)
	case []int:
		event.Ints(key, data)
	case []int8:
		event.Ints8(key, data)
	case []int16:
		event.Ints16(key, data)
	case []int32:
		event.Ints32(key, data)
	case []int64:
		event.Ints64(key, data)
	case uint:
		event.Uint(key, data)
	case uint8:
		event.Uint8(key, data)
	case uint16:
		event.Uint16(key, data)
	case uint32:
		event.Uint32(key, data)
	case uint64:
		event.Uint64(key, data)
	case []uint:
		event.Uints(key, data)
	case []uint8:
		event.Uints8(key, data)
	case []uint16:
		event.Uints16(key, data)
	case []uint32:
		event.Uints32(key, data)
	case []uint64:
		event.Uints64(key, data)
	case float32:
		event.Float32(key, data)
	case float64:
		event.Float64(key, data)
	case []float32:
		event.Floats32(key, data)
	case []float64:
		event.Floats64(key, data)
	case time.Time:
		event.Time(key, data)
	case []time.Time:
		event.Times(key, data)
	case time.Duration:
		event.Dur(key, data)
	case []time.Duration:
		event.Durs(key, data)
	case interface{ Iterator() iter.Seq2[string, any] }:
		irt.Apply2(data.Iterator(), addFieldOp(event))
	case json.Marshaler:
		encoded, err := json.Marshal(data)
		if err != nil {
			event.Err(err)
			event.Interface(key, data)
		} else {
			event.RawJSON(key, encoded)
		}
	default:
		event.Interface(key, data)
	}
}
