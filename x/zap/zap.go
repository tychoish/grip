package zap

import (
	"fmt"

	"github.com/tychoish/fun/dt"
	"github.com/tychoish/fun/risky"
	"github.com/tychoish/grip/level"
	"github.com/tychoish/grip/message"
	"github.com/tychoish/grip/send"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type shim struct {
	zap *zap.Logger
	send.Base
}

// NewSener provides a simple shim around zap that's compatible with
// other grip interfaces. Zap,like zerolog and other related
// structured loggers is a fast-path for JSON marshalling of
// structured log paths. The shim translates grip message types into
// appropriate Zerolog message building messages to preserve the fast
// path.
func MakeSender(zl *zap.Logger) send.Sender {
	s := &shim{zap: zl}

	return s
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

	if ce := s.zap.Check(convertLevel(m.Priority()), ""); ce != nil {
		if !m.Structured() {
			out, err := s.Format(m)
			if !s.HandleErrorOK(send.WrapError(err, m)) {
				return
			}
			ce.Message = out
			ce.Write()
			return
		}
		var fields []zap.Field

		payload := m.Raw()
		switch data := payload.(type) {
		case zap.Field:
			fields = append(fields, data)
		case []zap.Field:
			fields = append(fields, data...)
		case zapcore.ObjectMarshaler:
			fields = append(fields, zap.Inline(data))
		case error:
			fields = append(fields, zap.Error(data))
		case []error:
			fields = append(fields, zap.Errors("errors", data))
		case *dt.Pairs[string, any]:
			risky.Observe(data.Iterator(), func(kv dt.Pair[string, any]) {
				fields = append(fields, zap.Any(kv.Key, kv.Value))
			})
		case message.Fields:
			fields = append(fields, convertMapTypes(data)...)
		case map[string]any:
			fields = append(fields, convertMapTypes(data)...)
		default:
			fields = append(fields, zap.Any("payload", payload))
		}
		ce.Write(fields...)
	}
}

func convertMapTypes[K comparable, V any](in map[K]V) []zap.Field {
	out := make([]zap.Field, 0, len(in))
	for k, v := range in {
		out = append(out, zap.Any(fmt.Sprint(k), v))
	}
	return out
}

func convertLevel(in level.Priority) zapcore.Level {
	switch in {
	case level.Emergency:
		return zap.ErrorLevel
	case level.Alert:
		return zap.ErrorLevel
	case level.Critical:
		return zap.ErrorLevel
	case level.Error:
		return zap.ErrorLevel
	case level.Warning:
		return zap.WarnLevel
	case level.Notice:
		return zap.InfoLevel
	case level.Info:
		return zap.InfoLevel
	case level.Debug:
		return zap.DebugLevel
	case level.Trace:
		return zap.DebugLevel
	case level.Invalid:
		return zapcore.InvalidLevel
	default:
		return zapcore.InvalidLevel
	}
}
