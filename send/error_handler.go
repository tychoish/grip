package send

import (
	"fmt"
	"io"
	"log"

	"github.com/tychoish/grip/level"
	"github.com/tychoish/grip/message"
)

func ErrorHandlerWriter(writer io.Writer) ErrorHandler {
	return func(err error, m message.Composer) {
		if err == nil {
			return
		}

		_, _ = io.WriteString(writer, fmt.Sprintln("logging error:", err.Error()))
		_, _ = writer.Write([]byte("\n"))
		_, _ = io.WriteString(writer, m.String())
	}
}

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
		em := message.WrapError(err, m)
		em.SetPriority(level.Error)
		s.Send(em)
	}
}
