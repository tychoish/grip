package zerolog

import (
	"fmt"

	"github.com/rs/zerolog"
	"github.com/tychoish/grip/level"
	"github.com/tychoish/grip/message"
	"github.com/tychoish/grip/send"
)

type shim struct {
	zl zerolog.Logger
	*send.Base
}

func NewSender(name string, l send.LevelInfo, zl zerolog.Logger) (send.Sender, error) {
	s := &shim{
		Base: send.NewBase(name),
	}

	if err := s.SetLevel(l); err != nil {
		return nil, fmt.Errorf("problem seeting level on new sender: %w", err)
	}

	return s, nil
}

func MakeSender(zl zerolog.Logger) send.Sender {
	s, _ := NewSender("", send.LevelInfo{Threshold: level.Trace, Default: level.Debug}, zl)
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
	if !s.Level().ShouldLog(m) {
		return
	}

	event := s.zl.WithLevel(convertLevel(m.Priority()))
	defer event.Send()
}
