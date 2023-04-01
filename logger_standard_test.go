package grip

import (
	"testing"

	"github.com/tychoish/fun/assert"
	"github.com/tychoish/grip/level"
	"github.com/tychoish/grip/message"
	"github.com/tychoish/grip/send"
)

const testMessage = "hello world"

type (
	basicMethod func(any)
	lnMethod    func(...any)
	fMethod     func(string, ...any)
	whenMethod  func(bool, any)
)

type LoggingMethodSuite struct {
	logger        Logger
	loggingSender *send.InternalSender
	stdSender     *send.InternalSender
	defaultSender send.Sender
}

func setupFixtures(t *testing.T) *LoggingMethodSuite {
	t.Helper()
	lvl := level.Trace

	s := &LoggingMethodSuite{
		defaultSender: Sender(),
		stdSender:     send.MakeInternalLogger(),
		loggingSender: send.MakeInternalLogger(),
	}
	s.stdSender.SetPriority(lvl)
	s.loggingSender.SetPriority(lvl)

	SetGlobalLogger(NewLogger(s.stdSender))
	s.logger = NewLogger(s.loggingSender)
	t.Cleanup(func() {
		SetGlobalLogger(NewLogger(s.defaultSender))
	})

	return s
}

func TestWhenMethods(t *testing.T) {
	s := setupFixtures(t)

	cases := map[string][]whenMethod{
		"emergency": {EmergencyWhen, s.logger.EmergencyWhen},
		"alert":     {AlertWhen, s.logger.AlertWhen},
		"critical":  {CriticalWhen, s.logger.CriticalWhen},
		"error":     {ErrorWhen, s.logger.ErrorWhen},
		"warning":   {WarningWhen, s.logger.WarningWhen},
		"notice":    {NoticeWhen, s.logger.NoticeWhen},
		"info":      {InfoWhen, s.logger.InfoWhen},
		"debug":     {DebugWhen, s.logger.DebugWhen},
		"trace":     {TraceWhen, s.logger.TraceWhen},
	}

	for kind, loggers := range cases {
		t.Run(kind, func(t *testing.T) {
			if l := len(loggers); l != 2 {
				t.Errorf("length is %d not %d", l, 2)
			}
			loggers[0](true, testMessage)
			loggers[1](true, testMessage)

			if !s.loggingSender.HasMessage() {
				t.Error("value should be true")
			}
			if !s.stdSender.HasMessage() {
				t.Error("value should be true")
			}
			lgrMsg := s.loggingSender.GetMessage()
			if !lgrMsg.Logged {
				t.Error("value should be true")
			}
			stdMsg := s.stdSender.GetMessage()
			if !stdMsg.Logged {
				t.Error("value should be true")
			}

			if lgrMsg.Rendered != stdMsg.Rendered {
				t.Error("values should be equal")
			}

			loggers[0](false, testMessage)
			loggers[1](false, testMessage)

			lgrMsg = s.loggingSender.GetMessage()
			if lgrMsg.Logged {
				t.Error("value should be false")
			}
			stdMsg = s.stdSender.GetMessage()
			if stdMsg.Logged {
				t.Error("value should be false")
			}
		})
	}
}

func TestBasicMethod(t *testing.T) {
	s := setupFixtures(t)

	cases := map[string][]basicMethod{
		"emergency": {Emergency, s.logger.Emergency},
		"alert":     {Alert, s.logger.Alert},
		"critical":  {Critical, s.logger.Critical},
		"error":     {Error, s.logger.Error},
		"warning":   {Warning, s.logger.Warning},
		"notice":    {Notice, s.logger.Notice},
		"info":      {Info, s.logger.Info},
		"debug":     {Debug, s.logger.Debug},
		"trace":     {Trace, s.logger.Trace},
	}

	inputs := []any{true, false, []string{"a", "b"}, message.Fields{"a": 1}, 1, "foo"}

	for kind, loggers := range cases {
		t.Run(kind, func(t *testing.T) {
			if l := len(loggers); l != 2 {
				t.Errorf("length is %d not %d", l, 2)
			}
			if s.loggingSender.HasMessage() {
				t.Error("value should be false")
			}
			if s.stdSender.HasMessage() {
				t.Error("value should be false")
			}

			for _, msg := range inputs {
				loggers[0](msg)
				loggers[1](msg)

				if !s.loggingSender.HasMessage() {
					t.Error("value should be true")
				}
				if !s.stdSender.HasMessage() {
					t.Error("value should be true")
				}
				lgrMsg := s.loggingSender.GetMessage()
				stdMsg := s.stdSender.GetMessage()
				if lgrMsg.Rendered != stdMsg.Rendered {
					t.Log("rendered", lgrMsg.Rendered)
					t.Log("standard", stdMsg.Rendered)
					t.Error("values should be equal")
				}
			}
		})
	}

}

func TestFormatMethods(t *testing.T) {
	s := setupFixtures(t)

	cases := map[string][]fMethod{
		"emergency": {Emergencyf, s.logger.Emergencyf},
		"alert":     {Alertf, s.logger.Alertf},
		"critical":  {Criticalf, s.logger.Criticalf},
		"error":     {Errorf, s.logger.Errorf},
		"warning":   {Warningf, s.logger.Warningf},
		"notice":    {Noticef, s.logger.Noticef},
		"info":      {Infof, s.logger.Infof},
		"debug":     {Debugf, s.logger.Debugf},
		"trace":     {Tracef, s.logger.Tracef},
	}

	for kind, loggers := range cases {
		t.Run(kind, func(t *testing.T) {
			if l := len(loggers); l != 2 {
				t.Errorf("length is %d not %d", l, 2)
			}
			if s.loggingSender.HasMessage() {
				t.Error("value should be false")
			}
			if s.stdSender.HasMessage() {
				t.Error("value should be false")
			}

			loggers[0]("%s: %d", testMessage, 3)
			loggers[1]("%s: %d", testMessage, 3)

			if !s.loggingSender.HasMessage() {
				t.Error("value should be true")
			}
			if !s.stdSender.HasMessage() {
				t.Error("value should be true")
			}
			lgrMsg := s.loggingSender.GetMessage()
			stdMsg := s.stdSender.GetMessage()
			if lgrMsg.Rendered != stdMsg.Rendered {
				t.Error("values should be equal")
			}
		})
	}
}

func TestLineMethods(t *testing.T) {
	s := setupFixtures(t)

	cases := map[string][]lnMethod{
		"emergency": {Emergencyln, s.logger.Emergencyln},
		"alert":     {Alertln, s.logger.Alertln},
		"critical":  {Criticalln, s.logger.Criticalln},
		"error":     {Errorln, s.logger.Errorln},
		"warning":   {Warningln, s.logger.Warningln},
		"notice":    {Noticeln, s.logger.Noticeln},
		"info":      {Infoln, s.logger.Infoln},
		"debug":     {Debugln, s.logger.Debugln},
		"trace":     {Traceln, s.logger.Traceln},
	}

	for kind, loggers := range cases {
		t.Run(kind, func(t *testing.T) {

			if l := len(loggers); l != 2 {
				t.Errorf("length is %d not %d", l, 2)
			}
			if s.loggingSender.HasMessage() {
				t.Error("value should be false")
			}
			if s.stdSender.HasMessage() {
				t.Error("value should be false")
			}

			loggers[0](testMessage, 3)
			loggers[1](testMessage, 3)

			if !s.loggingSender.HasMessage() {
				t.Error("value should be true")
			}
			if !s.stdSender.HasMessage() {
				t.Error("value should be true")
			}
			lgrMsg := s.loggingSender.GetMessage()
			stdMsg := s.stdSender.GetMessage()
			if lgrMsg.Rendered != stdMsg.Rendered {
				t.Error("values should be equal")
			}
		})
	}
}

func TestProgrgramaticLevelMethods(t *testing.T) {
	s := setupFixtures(t)

	type (
		lgwhen   func(bool, level.Priority, any)
		lgwhenln func(bool, level.Priority, ...any)
		lgwhenf  func(bool, level.Priority, string, ...any)
		lg       func(level.Priority, any)
		lgln     func(level.Priority, ...any)
		lgf      func(level.Priority, string, ...any)
	)

	cases := map[string]any{
		"when": []lgwhen{LogWhen, s.logger.LogWhen},
		"lg":   []lg{Log, s.logger.Log},
		"lgln": []lgln{Logln, s.logger.Logln},
		"lgf":  []lgf{Logf, s.logger.Logf},
	}

	const l = level.Emergency

	for kind, loggers := range cases {
		t.Run(kind, func(t *testing.T) {
			if s.loggingSender.HasMessage() {
				t.Error("value should be false")
			}
			if s.stdSender.HasMessage() {
				t.Error("value should be false")
			}

			switch log := loggers.(type) {
			case []lgwhen:
				log[0](true, l, testMessage)
				log[1](true, l, testMessage)
			case []lgwhenln:
				log[0](true, l, testMessage, "->", testMessage)
				log[1](true, l, testMessage, "->", testMessage)
			case []lgwhenf:
				log[0](true, l, "%T: (%s) %s", log, kind, testMessage)
				log[1](true, l, "%T: (%s) %s", log, kind, testMessage)
			case []lg:
				log[0](l, testMessage)
				log[1](l, testMessage)
			case []lgln:
				log[0](l, testMessage, "->", testMessage)
				log[1](l, testMessage, "->", testMessage)
			case []lgf:
				log[0](l, "%T: (%s) %s", log, kind, testMessage)
				log[1](l, "%T: (%s) %s", log, kind, testMessage)
			default:
				panic("testing error")
			}

			if !s.loggingSender.HasMessage() {
				t.Error("value should be true")
			}
			if !s.stdSender.HasMessage() {
				t.Error("value should be true")
			}
			lgrMsg := s.loggingSender.GetMessage()
			stdMsg := s.stdSender.GetMessage()
			if lgrMsg.Rendered != stdMsg.Rendered {
				t.Error("values should be equal")
			}
		})
	}
}

func TestBuilder(t *testing.T) {
	s := setupFixtures(t)

	Build().Level(level.Info).String("hello").Send()
	s.logger.Build().Level(level.Info).String("hello").Send()

	assert.Equal(t,
		s.loggingSender.GetMessage().Rendered,
		s.stdSender.GetMessage().Rendered,
	)
}
