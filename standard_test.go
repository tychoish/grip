package grip

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/suite"
	"github.com/tychoish/grip/level"
	"github.com/tychoish/grip/message"
	"github.com/tychoish/grip/send"
)

const testMessage = "hello world"

type (
	basicMethod func(interface{})
	lnMethod    func(...interface{})
	fMethod     func(string, ...interface{})
	whenMethod  func(bool, interface{})
)

type LoggingMethodSuite struct {
	logger        Logger
	loggingSender *send.InternalSender
	stdSender     *send.InternalSender
	defaultSender send.Sender
	suite.Suite
}

func TestLoggingMethodSuite(t *testing.T) {
	suite.Run(t, new(LoggingMethodSuite))
}

func (s *LoggingMethodSuite) SetupTest() {
	s.defaultSender = GetGlobalLogger().GetSender()
	s.stdSender = send.MakeInternalLogger()
	SetGlobalLogger(NewLogger(s.stdSender))

	s.loggingSender = send.MakeInternalLogger()
	s.logger = NewLogger(s.loggingSender)
}

func (s *LoggingMethodSuite) TeardownTest() {
	SetGlobalLogger(NewLogger(s.defaultSender))
}

func (s *LoggingMethodSuite) TestWhenMethods() {
	cases := map[string][]whenMethod{
		"emergency": []whenMethod{EmergencyWhen, s.logger.EmergencyWhen},
		"alert":     []whenMethod{AlertWhen, s.logger.AlertWhen},
		"critical":  []whenMethod{CriticalWhen, s.logger.CriticalWhen},
		"error":     []whenMethod{ErrorWhen, s.logger.ErrorWhen},
		"warning":   []whenMethod{WarningWhen, s.logger.WarningWhen},
		"notice":    []whenMethod{NoticeWhen, s.logger.NoticeWhen},
		"info":      []whenMethod{InfoWhen, s.logger.InfoWhen},
		"debug":     []whenMethod{DebugWhen, s.logger.DebugWhen},
	}

	for kind, loggers := range cases {
		s.Len(loggers, 2)
		loggers[0](true, testMessage)
		loggers[1](true, testMessage)

		s.True(s.loggingSender.HasMessage())
		s.True(s.stdSender.HasMessage())
		lgrMsg := s.loggingSender.GetMessage()
		s.True(lgrMsg.Logged)
		stdMsg := s.stdSender.GetMessage()
		s.True(stdMsg.Logged)

		s.Equal(lgrMsg.Rendered, stdMsg.Rendered,
			fmt.Sprintf("%s: \n\tlogger: %+v \n\tstandard: %+v", kind, lgrMsg, stdMsg))

		loggers[0](false, testMessage)
		loggers[1](false, testMessage)

		lgrMsg = s.loggingSender.GetMessage()
		s.False(lgrMsg.Logged)
		stdMsg = s.stdSender.GetMessage()
		s.False(stdMsg.Logged)

	}
}

func (s *LoggingMethodSuite) TestBasicMethod() {
	cases := map[string][]basicMethod{
		"emergency": []basicMethod{Emergency, s.logger.Emergency},
		"alert":     []basicMethod{Alert, s.logger.Alert},
		"critical":  []basicMethod{Critical, s.logger.Critical},
		"error":     []basicMethod{Error, s.logger.Error},
		"warning":   []basicMethod{Warning, s.logger.Warning},
		"notice":    []basicMethod{Notice, s.logger.Notice},
		"info":      []basicMethod{Info, s.logger.Info},
		"debug":     []basicMethod{Debug, s.logger.Debug},
	}

	inputs := []interface{}{true, false, []string{"a", "b"}, message.Fields{"a": 1}, 1, "foo"}

	for kind, loggers := range cases {
		s.Len(loggers, 2)
		s.False(s.loggingSender.HasMessage())
		s.False(s.stdSender.HasMessage())

		for _, msg := range inputs {
			loggers[0](msg)
			loggers[1](msg)

			s.True(s.loggingSender.HasMessage())
			s.True(s.stdSender.HasMessage())
			lgrMsg := s.loggingSender.GetMessage()
			stdMsg := s.stdSender.GetMessage()
			s.Equal(lgrMsg.Rendered, stdMsg.Rendered,
				fmt.Sprintf("%s: \n\tlogger: %+v \n\tstandard: %+v", kind, lgrMsg, stdMsg))
		}
	}

}

func (s *LoggingMethodSuite) TestfMethods() {
	cases := map[string][]fMethod{
		"emergency": []fMethod{Emergencyf, s.logger.Emergencyf},
		"alert":     []fMethod{Alertf, s.logger.Alertf},
		"critical":  []fMethod{Criticalf, s.logger.Criticalf},
		"error":     []fMethod{Errorf, s.logger.Errorf},
		"warning":   []fMethod{Warningf, s.logger.Warningf},
		"notice":    []fMethod{Noticef, s.logger.Noticef},
		"info":      []fMethod{Infof, s.logger.Infof},
		"debug":     []fMethod{Debugf, s.logger.Debugf},
	}

	for kind, loggers := range cases {
		s.Len(loggers, 2)
		s.False(s.loggingSender.HasMessage())
		s.False(s.stdSender.HasMessage())

		loggers[0]("%s: %d", testMessage, 3)
		loggers[1]("%s: %d", testMessage, 3)

		s.True(s.loggingSender.HasMessage())
		s.True(s.stdSender.HasMessage())
		lgrMsg := s.loggingSender.GetMessage()
		stdMsg := s.stdSender.GetMessage()
		s.Equal(lgrMsg.Rendered, stdMsg.Rendered,
			fmt.Sprintf("%s: \n\tlogger: %+v \n\tstandard: %+v", kind, lgrMsg, stdMsg))
	}
}

func (s *LoggingMethodSuite) TestProgramaticLevelMethods() {
	type (
		lgwhen   func(bool, level.Priority, interface{})
		lgwhenln func(bool, level.Priority, ...interface{})
		lgwhenf  func(bool, level.Priority, string, ...interface{})
		lg       func(level.Priority, interface{})
		lgln     func(level.Priority, ...interface{})
		lgf      func(level.Priority, string, ...interface{})
	)

	cases := map[string]interface{}{
		"when": []lgwhen{LogWhen, s.logger.LogWhen},
		"lg":   []lg{Log, s.logger.Log},
		"lgf":  []lgf{Logf, s.logger.Logf},
	}

	const l = level.Emergency

	for kind, loggers := range cases {
		s.Len(loggers, 2)
		s.False(s.loggingSender.HasMessage())
		s.False(s.stdSender.HasMessage())

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

		s.True(s.loggingSender.HasMessage())
		s.True(s.stdSender.HasMessage())
		lgrMsg := s.loggingSender.GetMessage()
		stdMsg := s.stdSender.GetMessage()
		s.Equal(lgrMsg.Rendered, stdMsg.Rendered,
			fmt.Sprintf("%s: \n\tlogger: %+v \n\tstandard: %+v", kind, lgrMsg, stdMsg))
	}
}

type GripSuite struct {
	grip Logger
	name string
	suite.Suite
}

func TestGripSuite(t *testing.T) {
	suite.Run(t, new(GripSuite))
}

func (s *GripSuite) SetupSuite() {
	s.grip = NewLogger(send.MakeNative())
	s.grip.GetSender().SetName(s.name)
	s.Equal(s.grip.GetSender().Name(), s.name)
}

func (s *GripSuite) SetupTest() {}

func (s *GripSuite) TestDefaultJournalerIsBootstrap() {
	firstName := s.grip.GetSender().Name()
	// the bootstrap sender is a bit special because you can't
	// change it's name, therefore:
	secondName := "something_else"
	s.grip.GetSender().SetName(secondName)

	s.Equal(s.grip.GetSender().Name(), secondName)
	s.NotEqual(s.grip.GetSender().Name(), firstName)
	s.NotEqual(firstName, secondName)
}

func (s *GripSuite) TestNameSetterAndGetter() {
	for _, name := range []string{"a", "a39df", "a@)(*E)"} {
		s.grip.GetSender().SetName(name)
		s.Equal(s.grip.GetSender().Name(), name)
	}
}
