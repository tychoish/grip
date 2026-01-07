package zap

import (
	"fmt"
	"iter"

	"github.com/tychoish/fun/irt"
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
			fields = data
		case zapcore.ObjectMarshaler:
			fields = append(fields, zap.Inline(data))
		case error:
			fields = append(fields, zap.Error(data))
		case []error:
			fields = append(fields, zap.Errors("errors", data))
		case iter.Seq2[string, any]:
			fields = convertMapTypes(data, 8)
		case []irt.KV[string, any]:
			fields = convertMapTypes(irt.KVsplit(irt.Slice(data)), len(data))
		case iter.Seq[irt.KV[string, any]]:
			fields = convertMapTypes(irt.KVsplit(data), 8)
		case message.Fields:
			fields = convertMapTypes(irt.Map(data), len(data))
		case map[string]any:
			fields = convertMapTypes(irt.Map(data), len(data))
		default:
			fields = append(fields, zap.Any("payload", payload))
		}
		ce.Write(fields...)
	}
}

func toField[K comparable, V any](k K, v V) zap.Field { return zap.Any(fmt.Sprint(k), v) }

func convertMapTypes[K comparable, V any](seq iter.Seq2[K, V], hint ...int) []zap.Field {
	return irt.Collect(irt.Merge(seq, toField))
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
