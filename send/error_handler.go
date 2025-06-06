package send

import (
	"fmt"
	"io"
	"log"

	"github.com/tychoish/fun/erc"
	"github.com/tychoish/fun/ers"
	"github.com/tychoish/fun/fn"
	"github.com/tychoish/grip/level"
	"github.com/tychoish/grip/message"
)

const ErrGripMessageSendError ers.Error = "unable to send grip log message"

// WrapError wraps an error with the message composer and produces a
// combined error that includes the string form of the message (as an
// error,) ErrGripMessageSendError, and the underlying error. When the
// input error is nil, the error is nil.
func WrapError(err error, m message.Composer) error {
	if ers.IsOk(err) {
		return nil
	}

	return erc.Join(ErrGripMessageSendError, err, ers.Error(m.String()))
}

// ErrorHandlerWriter returns a fun.Handler that writes the error to the
// provided io.Writer.
func ErrorHandlerWriter(writer io.Writer) ErrorHandler {
	return ErrorHandler(fn.NewHandler(func(err error) {
		if ers.IsOk(err) {
			return
		}

		_, _ = io.WriteString(writer, fmt.Sprintln("logging error:", err.Error()))
		_, _ = writer.Write([]byte("\n"))
	}))
}

// ErrorHandlerFromLogger returns a fun.Handler that logs the error with the
// provided standard library *log.Logger.
func ErrorHandlerFromLogger(l *log.Logger) ErrorHandler {
	return ErrorHandler(fn.NewHandler(func(err error) {
		if ers.IsOk(err) {
			return
		}

		l.Println("logging error:", err.Error())
	}))
}

// ErrorHandlerFromSender wraps an existing Sender for sending error messages and
// exposes it as a fun.Handler.
func ErrorHandlerFromSender(s Sender) ErrorHandler {
	return ErrorHandler(fn.NewHandler(func(err error) {
		if ers.IsOk(err) {
			return
		}

		em := message.MakeError(err)
		em.SetPriority(level.Error)
		s.Send(em)
	}))
}
