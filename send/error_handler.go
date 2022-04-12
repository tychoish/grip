package send

import (
	"log"

	"github.com/tychoish/emt"
	"github.com/tychoish/grip/message"
)

// ErrorHandler is a function that you can use define how a sender
// handles errors sending messages. Implementations of this type
// should perform a noop if the err object is nil.
type ErrorHandler func(error, message.Composer)

func ErrorHandlerFromLogger(l *log.Logger) ErrorHandler {
	return func(err error, m message.Composer) {
		if err == nil {
			return
		}

		l.Println("logging error:", err.Error())
		l.Println(m.String())
	}
}

// ErrorHandlerFromSender wraps an existing Sender for sending error messages.
func ErrorHandlerFromSender(s Sender) ErrorHandler {
	return func(err error, m message.Composer) {
		if err == nil {
			return
		}

		s.Send(message.WrapError(err, m))
	}
}

// MakeCatcherErrorHandler produces an error handler useful for
// collecting errors from a sender using the supplied error
// catcher. At the very least, consider using a catcher that has a
// specified maxsize, and possibly timestamp annotating catcher as
// well.
func MakeCatcherErrorHandler(catcher emt.Catcher, fallback Sender) ErrorHandler {
	return func(err error, m message.Composer) {
		if err == nil {
			return
		}

		catcher.Add(err)

		if fallback != nil {
			fallback.Send(message.WrapError(err, m))
		}
	}
}
