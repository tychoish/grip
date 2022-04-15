package grip

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/suite"
	"github.com/tychoish/grip/level"
	"github.com/tychoish/grip/message"
	"github.com/tychoish/grip/send"
)

type GripInternalSuite struct {
	grip Logger
	name string
	suite.Suite
}

func TestLoggerSuite(t *testing.T) {
	suite.Run(t, new(GripInternalSuite))
}

func (s *GripInternalSuite) SetupTest() {
	sender, err := send.NewStdOutput(s.name, send.LevelInfo{Default: level.Info, Threshold: level.Trace})
	s.NoError(err)
	s.Equal(sender.Name(), s.name)
}

func (s *GripInternalSuite) TestPanicSenderActuallyPanics() {
	// both of these are in anonymous functions so that the defers
	// cover the correct area.

	func() {
		// first make sure that the default send method doesn't panic
		defer func() {
			s.Nil(recover())
		}()

		s.grip.impl.Send(message.NewLines(level.Critical, "foo"))
	}()

	func() {
		// call a panic function with a recoverer set.
		defer func() {
			s.NotNil(recover())
		}()

		s.grip.sendPanic(level.Info, message.MakeLines("foo"))
	}()
}

func (s *GripInternalSuite) TestPanicSenderRespectsTThreshold() {
	s.True(level.Debug > s.grip.Sender().Level().Threshold)
	s.NoError(s.grip.impl.SetLevel(send.LevelInfo{Default: level.Info, Threshold: level.Notice}))
	s.True(level.Debug < s.grip.impl.Level().Threshold)

	// test that there is a no panic if the message isn't "logabble"
	defer func() {
		s.Nil(recover())
	}()

	s.grip.sendPanic(level.Debug, message.MakeLines("foo"))
}

func (s *GripInternalSuite) TestConditionalSend() {
	// because sink is an internal type (implementation of
	// sender,) and "GetMessage" isn't in the interface, though it
	// is exported, we can't pass the sink between functions.
	sink, err := send.NewInternalLogger("sink", s.grip.Sender().Level())
	s.NoError(err)
	s.grip.impl = sink

	msg := message.NewLines(level.Info, "foo")
	msgTwo := message.NewLines(level.Notice, "bar")

	// when the conditional argument is true, it should work
	s.grip.Log(msg.Priority(), message.When(true, msg))
	s.Equal(msg.Raw(), sink.GetMessage().Message.Raw())

	// when the conditional argument is true, it should work, and the channel is fifo
	s.grip.Log(msgTwo.Priority(), message.When(false, msgTwo))
	s.grip.Log(msg.Priority(), message.When(true, msg))
	result := sink.GetMessage().Message
	if result.Loggable() {
		s.Equal(msg.Raw(), result.Raw())
	} else {
		s.Equal(msgTwo.Raw(), result.Raw())
	}

	// change the order
	s.grip.Log(msg.Priority(), message.When(true, msg))
	s.grip.Log(msgTwo.Priority(), message.When(false, msgTwo))
	result = sink.GetMessage().Message

	if result.Loggable() {
		s.Equal(msg.Raw(), result.Raw())
	} else {
		s.Equal(msgTwo.Raw(), result.Raw())
	}
}

func (s *GripInternalSuite) TestCatchMethods() {
	sink, err := send.NewInternalLogger("sink", send.LevelInfo{Default: level.Trace, Threshold: level.Trace})
	s.NoError(err)
	s.grip = NewLogger(sink)

	cases := []interface{}{
		s.grip.Alert,
		s.grip.Critical,
		s.grip.Debug,
		s.grip.Emergency,
		s.grip.Error,
		s.grip.Info,
		s.grip.Notice,
		s.grip.Warning,

		s.grip.Alertf,
		s.grip.Criticalf,
		s.grip.Debugf,
		s.grip.Emergencyf,
		s.grip.Errorf,
		s.grip.Infof,
		s.grip.Noticef,
		s.grip.Warningf,

		s.grip.AlertWhen,
		s.grip.CriticalWhen,
		s.grip.DebugWhen,
		s.grip.EmergencyWhen,
		s.grip.ErrorWhen,
		s.grip.InfoWhen,
		s.grip.NoticeWhen,
		s.grip.WarningWhen,

		func(w bool, m interface{}) { s.grip.LogWhen(w, level.Info, m) },
		func(m interface{}) { s.grip.Log(level.Info, m) },
		func(m string, a ...interface{}) { s.grip.Logf(level.Info, m, a...) },
		func(m ...message.Composer) { s.grip.Log(level.Info, m) },
		func(m []message.Composer) { s.grip.Log(level.Info, m) },
		func(w bool, m ...message.Composer) { s.grip.LogWhen(w, level.Info, m) },
		func(w bool, m []message.Composer) { s.grip.LogWhen(w, level.Info, m) },
	}

	const msg = "hello world!"
	multiMessage := []message.Composer{
		message.ConvertWithPriority(0, nil),
		message.ConvertWithPriority(0, msg),
	}

	for idx, logger := range cases {
		s.Equal(0, sink.Len())
		s.False(sink.HasMessage())

		switch log := logger.(type) {
		case func(error):
			log(errors.New(msg))
		case func(interface{}):
			log(msg)
		case func(...interface{}):
			log(msg, "", nil)
		case func(string, ...interface{}):
			log("%s", msg)
		case func(bool, interface{}):
			log(false, msg)
			log(true, msg)
		case func(bool, ...interface{}):
			log(false, msg, "", nil)
			log(true, msg, "", nil)
		case func(bool, string, ...interface{}):
			log(false, "%s", msg)
			log(true, "%s", msg)
		case func(...message.Composer):
			log(multiMessage...)
		case func(bool, ...message.Composer):
			log(false, multiMessage...)
			log(true, multiMessage...)
		case func([]message.Composer):
			log(multiMessage)
		case func(bool, []message.Composer):
			log(false, multiMessage)
			log(true, multiMessage)
		default:
			panic(fmt.Sprintf("%T is not supported\n", log))
		}

		if sink.Len() > 1 {
			// this is the many case
			var numLogged int
			out := sink.GetMessage()
			for i := 0; i < sink.Len(); i++ {
				out = sink.GetMessage()
				if out.Logged {
					numLogged++
					s.Equal(out.Rendered, msg)
				}
			}

			s.True(numLogged == 1, fmt.Sprintf("[id=%d] %T: %d %s", idx, logger, numLogged, out.Priority))

			continue
		}

		s.True(sink.Len() == 1)
		s.True(sink.HasMessage())
		out := sink.GetMessage()
		s.Equal(out.Rendered, msg)
		s.True(out.Logged, fmt.Sprintf("[id=%d] %T %s", idx, logger, out.Priority))
	}
}

// This testing method uses the technique outlined in:
// http://stackoverflow.com/a/33404435 to test a function that exits
// since it's impossible to "catch" an os.Exit
func TestSendFatalExits(t *testing.T) {
	grip := NewLogger(send.MakeStdOutput())
	if os.Getenv("SHOULD_CRASH") == "1" {
		grip.sendFatal(level.Error, message.MakeLines("foo"))
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=TestSendFatalExits")
	cmd.Env = append(os.Environ(), "SHOULD_CRASH=1")
	err := cmd.Run()
	if err == nil {
		t.Errorf("sendFatal should have exited 0, instead: %+v", err)
	}
}
