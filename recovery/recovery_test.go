package recovery

import (
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/tychoish/grip"
	"github.com/tychoish/grip/message"
	"github.com/tychoish/grip/send"
)

func setupFixture(t *testing.T) *send.InternalSender {
	t.Helper()
	sender := grip.Sender()
	out := send.MakeInternal()
	if err := os.Setenv(killOverrideVarName, "true"); err != nil {
		t.Fatal(err)
	}

	grip.SetGlobalLogger(grip.NewLogger(out))
	t.Cleanup(func() {
		grip.SetGlobalLogger(grip.NewLogger(sender))
		if err := os.Setenv(killOverrideVarName, ""); err != nil {
			t.Error(err)
		}
	})
	return out

}

func TestWithoutPanicNoErrorsLoged(t *testing.T) {
	sender := setupFixture(t)

	if sender.HasMessage() {
		t.Error("should be false")
	}
	LogStackTraceAndContinue()
	if sender.HasMessage() {
		t.Error("should be false")
	}
	LogStackTraceAndExit()
	if sender.HasMessage() {
		t.Error("should be false")
	}
	if err := HandlePanicWithError(nil, nil); err != nil {
		t.Fatal(err)
	}
	if sender.HasMessage() {
		t.Error("should be false")
	}
}

func TestPanicCausesLogsWithContinueRecoverer(t *testing.T) {
	sender := setupFixture(t)
	if sender.HasMessage() {
		t.Error("should be false")
	}
	func() {
		// shouldn't panic
		defer LogStackTraceAndContinue()
		panic("sorry")
	}()
	if !sender.HasMessage() {
		t.Error("should be true")
	}
	msg, ok := sender.GetMessageSafe()
	if !ok {
		t.Error("should be true")
	}
	if !strings.Contains(msg.Rendered, "hit panic; recovering") {
		t.Error("string should contain substring")
	}
	if !strings.Contains(msg.Rendered, "sorry") {
		t.Error("string should contain substring")
	}
}

func TestPanicsCausesLogsWithExitHandler(t *testing.T) {
	sender := setupFixture(t)
	if sender.HasMessage() {
		t.Error("should be false")
	}
	grip.SetGlobalLogger(grip.NewLogger(sender))
	func() {
		defer LogStackTraceAndExit("exit op")
		panic("sorry buddy")
	}()
	if !sender.HasMessage() {
		t.Error("should be true")
	}
	msg, ok := sender.GetMessageSafe()
	if !ok {
		t.Error("should be true")
	}
	if !strings.Contains(msg.Rendered, "hit panic; exiting") {
		t.Error("should be true")
	}
	if !strings.Contains(msg.Rendered, "sorry buddy") {
		t.Error("should be true")
	}
	if !strings.Contains(msg.Rendered, "exit op") {
		t.Error("should be true")
	}
}

func TestPanicCausesLogsWithErrorHandler(t *testing.T) {
	sender := setupFixture(t)
	if sender.HasMessage() {
		t.Error("should be false")
	}
	grip.SetGlobalLogger(grip.NewLogger(sender))
	func() {
		// shouldn't panic

		err := func() (err error) {
			defer func() { err = HandlePanicWithError(recover(), nil) }()
			panic("get a grip")
		}()

		if err == nil {
			t.Fatal("error should not be nil")
		}
		if !strings.Contains(err.Error(), "get a grip") {
			t.Error("should be true")
		}
	}()
	if !sender.HasMessage() {
		t.Error("should be true")
	}
	msg, ok := sender.GetMessageSafe()
	if !ok {
		t.Error("should be true")
	}
	if !strings.Contains(msg.Rendered, "hit panic; adding error") {
		t.Error("should be true")
	}
	if !strings.Contains(msg.Rendered, "get a grip") {
		t.Error("should be true")
	}
}

func TestErrorHandlerPropogatesErrorAndPanicMessage(t *testing.T) {
	sender := setupFixture(t)
	func() {
		// shouldn't panic

		err := func() (err error) {
			defer func() { err = HandlePanicWithError(recover(), errors.New("bar"), "this op name") }()
			panic("got grip")
		}()

		if err == nil {
			t.Fatal("error should not be nil")
		}
		if !strings.Contains(err.Error(), "got grip") {
			t.Error("should be true")
		}
		if !strings.Contains(err.Error(), "bar") {
			t.Error("should be true")
		}
		if strings.Contains(err.Error(), "op name") {
			t.Error("should be false")
		}
	}()

	if !sender.HasMessage() {
		t.Error("should be true")
	}
	msg, ok := sender.GetMessageSafe()
	if !ok {
		t.Error("should be true")
	}
	if !strings.Contains(msg.Rendered, "this op name") {
		t.Error("string should contain substring")
	}
	if !strings.Contains(msg.Rendered, "got grip") {
		t.Error("string should contain substring")
	}
	if !strings.Contains(msg.Rendered, "bar") {
		t.Error("string should contain substring")
	}
}

func TestPanicHandlerWithErrorPropogatesErrorWithoutPanic(t *testing.T) {
	_ = setupFixture(t)
	err := HandlePanicWithError(nil, errors.New("foo"))
	if err == nil {
		t.Fatal("error should not be nil")
	}

	if !strings.Contains(err.Error(), "foo") {
		t.Error("should be true")
	}
}

func TestPanicHandlerPropogatesOperationName(t *testing.T) {
	sender := setupFixture(t)
	if sender.HasMessage() {
		t.Error("should be false")
	}
	func() {
		// shouldn't panic
		defer LogStackTraceAndContinue("test handler op")
		panic("sorry")
	}()
	if !sender.HasMessage() {
		t.Error("should be true")
	}
	msg, ok := sender.GetMessageSafe()
	if !ok {
		t.Error("should be true")
	}
	if !strings.Contains(msg.Rendered, "test handler op") {
		t.Error("should be true")
	}
}

func TestPanicHandlerPropogatesOperationNameWithArgs(t *testing.T) {
	sender := setupFixture(t)
	if sender.HasMessage() {
		t.Error("should be false")
	}
	func() {
		defer LogStackTraceAndContinue("test handler op", "for real")
		panic("sorry")
	}()
	if !sender.HasMessage() {
		t.Error("should be true")
	}
	msg, ok := sender.GetMessageSafe()
	if !ok {
		t.Error("should be true")
	}
	if !strings.Contains(msg.Rendered, "test handler op for real") {
		t.Error("should be true")
	}
}

func TestPanicHandlerAnnotationPropogagaesMessage(t *testing.T) {
	sender := setupFixture(t)
	if sender.HasMessage() {
		t.Error("should be false")
	}
	func() {
		// shouldn't panic
		defer AnnotateMessageWithStackTraceAndContinue(message.Fields{"foo": "test handler op1 for real"})
		panic("sorry")
	}()
	if !sender.HasMessage() {
		t.Error("should be true")
	}
	msg, ok := sender.GetMessageSafe()
	if !ok {
		t.Error("should be true")
	}
	if !strings.Contains(msg.Rendered, "test handler op1 for real") {
		t.Error("should be true")
	}

}

func TestPanicsCausesAnnotateLogsWithExitHandler(t *testing.T) {
	sender := setupFixture(t)
	if sender.HasMessage() {
		t.Error("should be false")
	}
	func() {
		// shouldn't panic
		defer AnnotateMessageWithStackTraceAndExit(message.Fields{"foo": "exit op1"})
		panic("sorry buddy")
	}()
	if !sender.HasMessage() {
		t.Error("should be true")
	}
	msg, ok := sender.GetMessageSafe()
	if !ok {
		t.Error("should be true")
	}
	if !strings.Contains(msg.Rendered, "hit panic; exiting") {
		t.Error("should be true")
	}
	if !strings.Contains(msg.Rendered, "sorry buddy") {
		t.Error("should be true")
	}
	if !strings.Contains(msg.Rendered, "exit op1") {
		t.Error("should be true")
	}
}

func TestPanicAnnotatesLogsWithErrorHandler(t *testing.T) {
	sender := setupFixture(t)
	if sender.HasMessage() {
		t.Error("should be false")
	}
	func() {
		// shouldn't panic
		err := func() (err error) {
			defer func() { err = AnnotateMessageWithPanicError(recover(), nil, message.Fields{"foo": "bar"}) }()
			panic("get a grip")
		}()

		if err == nil {
			t.Fatal("error should not be nil")
		}
		if !strings.Contains(err.Error(), "get a grip") {
			t.Error("should be true")
		}
	}()
	if !sender.HasMessage() {
		t.Error("should be true")
	}
	msg, ok := sender.GetMessageSafe()
	if !ok {
		t.Error("should be true")
	}
	if !strings.Contains(msg.Rendered, "hit panic; adding error") {
		t.Error("should be true")
	}
	if !strings.Contains(msg.Rendered, "get a grip") {
		t.Error("should be true")
	}
	if !strings.Contains(msg.Rendered, "foo='bar'") {
		t.Error("should be true")
	}
}

func TestPanicHandlerSendJournalerPropogagaesMessage(t *testing.T) {
	sender := setupFixture(t)
	if sender.HasMessage() {
		t.Error("should be false")
	}
	func() {
		// shouldn't panic
		logger := grip.NewLogger(sender)
		defer SendStackTraceAndContinue(logger, message.Fields{"foo": "test handler op2 for real"})

		panic("sorry")
	}()
	if !sender.HasMessage() {
		t.Error("should be true")
	}
	msg, ok := sender.GetMessageSafe()
	if !ok {
		t.Error("should be true")
	}
	if !strings.Contains(msg.Rendered, "test handler op2 for real") {
		t.Error("should be true")
	}

}

func TestPanicsCausesSendJournalerLogsWithExitHandler(t *testing.T) {
	sender := setupFixture(t)
	if sender.HasMessage() {
		t.Error("should be false")
	}
	func() {
		// shouldn't panic
		logger := grip.NewLogger(sender)
		defer SendStackTraceMessageAndExit(logger, message.Fields{"foo": "exit op2"})
		panic("sorry buddy")
	}()
	if !sender.HasMessage() {
		t.Error("should be true")
	}
	msg, ok := sender.GetMessageSafe()
	if !ok {
		t.Error("should be true")
	}
	if !strings.Contains(msg.Rendered, "hit panic; exiting") {
		t.Error("should be true")
	}
	if !strings.Contains(msg.Rendered, "sorry buddy") {
		t.Error("should be true")
	}
	if !strings.Contains(msg.Rendered, "exit op2") {
		t.Error("should be true")
	}
}

func TestPanicSendJournalerLogsWithErrorHandler(t *testing.T) {
	sender := setupFixture(t)
	if sender.HasMessage() {
		t.Error("should be false")
	}

	func() {
		// shouldn't panic
		err := func() (err error) {
			logger := grip.NewLogger(sender)
			defer func() { err = SendMessageWithPanicError(recover(), nil, logger, message.Fields{"foo": "bar1"}) }()
			panic("get a grip")
		}()

		if err == nil {
			t.Fatal("error shouldn't be nil")
		}
		if !strings.Contains(err.Error(), "get a grip") {
			t.Error("should be true")
		}
	}()
	if !sender.HasMessage() {
		t.Error("should be true")
	}
	msg, ok := sender.GetMessageSafe()
	if !ok {
		t.Error("should be true")
	}
	if !strings.Contains(msg.Rendered, "hit panic; adding error") {
		t.Error("should be true")
	}
	if !strings.Contains(msg.Rendered, "get a grip") {
		t.Error("should be true")
	}
	if !strings.Contains(msg.Rendered, "foo='bar1'") {
		t.Error("should be true")
	}
}
