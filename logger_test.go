package grip

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"testing"

	"github.com/tychoish/grip/level"
	"github.com/tychoish/grip/message"
	"github.com/tychoish/grip/send"
)

func TestContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if Context(ctx) != std {
		t.Fatal("context does not default to standard")
	}

	logger := NewLogger(send.MakeStdOutput())
	ctx = WithLogger(ctx, logger)

	if Context(ctx) == std {
		t.Fatal("context logger should not return standard if set")
	}

	if Context(ctx) != logger {
		t.Fatal("context should return expected value")
	}
}

func TestLogger(t *testing.T) {
	const name = "gripTest"
	testSender := func(t *testing.T) send.Sender {
		t.Helper()
		sender, err := send.NewStdOutput(name, send.LevelInfo{Default: level.Info, Threshold: level.Trace})
		if err != nil {
			t.Fatal(err)
		}
		if sender.Name() != name {
			t.Errorf("sender is named %q not %q", sender.Name(), name)
		}
		return sender
	}

	t.Run("PanicSenderPanics", func(t *testing.T) {
		// both of these are in anonymous functions so that the defers
		// cover the correct area.

		func() {
			// first make sure that the default send method doesn't panic
			defer func() {
				if r := recover(); r != nil {
					t.Fatal(r)

				}
			}()

			gripImpl := NewLogger(testSender(t))

			gripImpl.Log(level.Critical, message.MakeLines("foo"))
		}()

		func() {
			// call a panic function with a recoverer set.
			defer func() {
				if r := recover(); r == nil {
					t.Fatal("did not panic in expected situation")
				}
			}()

			gripImpl := NewLogger(testSender(t))
			gripImpl.sendPanic(level.Info, message.MakeLines("foo"))
		}()
	})
	t.Run("PanicRespectsThreshold", func(t *testing.T) {
		grip := NewLogger(testSender(t))

		if level.Debug < grip.Sender().Level().Threshold {
			t.Fatal("level ordering is not correct")
		}
		if err := grip.Sender().SetLevel(send.LevelInfo{Default: level.Info, Threshold: level.Notice}); err != nil {
			t.Fatal(err)
		}
		if level.Debug > grip.Sender().Level().Threshold {
			t.Fatal("level ordering is not correct")
		}

		// test that there is a no panic if the message isn't "logabble"
		defer func() {
			if r := recover(); r != nil {
				t.Fatal("panic doesn't respect level")

			}
		}()

		grip.sendPanic(level.Debug, message.MakeLines("foo"))
	})
	t.Run("ConditionalSend", func(t *testing.T) {
		// because sink is an internal type (implementation of
		// sender,) and "GetMessage" isn't in the interface, though it
		// is exported, we can't pass the sink between functions.
		sink, err := send.NewInternalLogger("sink", send.LevelInfo{Default: level.Debug, Threshold: level.Info})
		if err != nil {
			t.Fatal(err)
		}
		grip := NewLogger(sink)

		msg := message.NewLines(level.Info, "foo")
		msgTwo := message.NewLines(level.Notice, "bar")

		// when the conditional argument is true, it should work
		grip.Log(msg.Priority(), message.When(true, msg))
		if msg.Raw() != sink.GetMessage().Message.Raw() {
			t.Fatal("messages is not propagated")
		}

		// when the conditional argument is true, it should work, and the channel is fifo
		grip.Log(msgTwo.Priority(), message.When(false, msgTwo))
		grip.Log(msg.Priority(), message.When(true, msg))
		result := sink.GetMessage().Message
		if result.Loggable() {
			if msg.Raw() != result.Raw() {
				t.Fatal("message is not propagated")
			}
		} else {
			if msgTwo.Raw() != result.Raw() {
				t.Fatal("message is not propagated")
			}
		}

		// change the order
		grip.Log(msg.Priority(), message.When(true, msg))
		grip.Log(msgTwo.Priority(), message.When(false, msgTwo))
		result = sink.GetMessage().Message

		if result.Loggable() {
			if msg.Raw() != result.Raw() {
				t.Fatal("message is not propagated")
			}
		} else {
			if msgTwo.Raw() != result.Raw() {
				t.Fatal("message is not propagated")
			}
		}
	})
	t.Run("CatchMethods", func(t *testing.T) {
		sink, err := send.NewInternalLogger("sink", send.LevelInfo{Default: level.Debug, Threshold: level.Trace})
		if err != nil {
			t.Fatal(err)
		}
		grip := NewLogger(sink)

		cases := []interface{}{
			grip.Alert,
			grip.Critical,
			grip.Debug,
			grip.Emergency,
			grip.Error,
			grip.Info,
			grip.Notice,
			grip.Warning,

			grip.Alertf,
			grip.Criticalf,
			grip.Debugf,
			grip.Emergencyf,
			grip.Errorf,
			grip.Infof,
			grip.Noticef,
			grip.Warningf,

			grip.AlertWhen,
			grip.CriticalWhen,
			grip.DebugWhen,
			grip.EmergencyWhen,
			grip.ErrorWhen,
			grip.InfoWhen,
			grip.NoticeWhen,
			grip.WarningWhen,

			func(w bool, m interface{}) { grip.LogWhen(w, level.Info, m) },
			func(m interface{}) { grip.Log(level.Info, m) },
			func(m string, a ...interface{}) { grip.Logf(level.Info, m, a...) },
			func(m ...message.Composer) { grip.Log(level.Info, m) },
			func(m []message.Composer) { grip.Log(level.Info, m) },
			func(w bool, m ...message.Composer) { grip.LogWhen(w, level.Info, m) },
			func(w bool, m []message.Composer) { grip.LogWhen(w, level.Info, m) },
		}

		const msg = "hello world!"
		multiMessage := []message.Composer{
			message.ConvertWithPriority(0, nil),
			message.ConvertWithPriority(0, msg),
		}

		for idx, logger := range cases {
			if sink.Len() != 0 {
				t.Fatalf("sink has %d", sink.Len())
			}

			if sink.HasMessage() {
				t.Fatal("messages exist in sink before test")
			}

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
						if out.Rendered != msg {
							t.Fatal("message rendered incorrectly")
						}
					}
				}
				if numLogged != 1 {
					t.Fatalf("[id=%d] %T: %d %s", idx, logger, numLogged, out.Priority)
				}

				continue
			}

			if sink.Len() != 1 {
				t.Fatal("sink has incorrect number of messages", sink.Len())
			}

			if !sink.HasMessage() {
				t.Fatal("sink does not have any messages")
			}

			out := sink.GetMessage()
			if out.Rendered != msg {
				t.Fatal("message rendered incorrectly")
			}

			if !out.Logged {
				t.Fatalf("[id=%d] %T %s", idx, logger, out.Priority)
			}
		}

	})
	t.Run("DefaultJournalerIsBootstrap", func(t *testing.T) {
		grip := NewLogger(testSender(t))
		firstName := grip.Sender().Name()
		// the bootstrap sender is a bit special because you can't
		// change it's name, therefore:
		const secondName = "something_else"
		grip.Sender().SetName(secondName)

		if grip.Sender().Name() != secondName {
			t.Fatal("name incorrect")
		}
		if grip.Sender().Name() == firstName {
			t.Fatal("name incorrect")
		}
		if firstName == secondName {
			t.Fatal("names should not be equal")
		}
	})
	t.Run("NameControler", func(t *testing.T) {
		grip := NewLogger(testSender(t))
		for _, name := range []string{"a", "a39df", "a@)(*E)"} {
			grip.Sender().SetName(name)
			if grip.Sender().Name() != name {
				t.Fatal("name was not correctly set")
			}
		}

	})

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
