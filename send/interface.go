// Package send provides an interface for defining "senders" for
// different logging backends, as well as basic implementations for
// common logging approaches to use with the Grip logging
// interface. Backends currently include: syslog, systemd's journal,
// standard output, and file baased methods.
package send

import (
	"context"
	"log"

	"github.com/tychoish/grip/level"
	"github.com/tychoish/grip/message"
)

// The Sender interface describes a lower level message sending
// interface used by the Logger to send messages. Implementations in
// the send package, in addition to a number of senders implemented in
// the x/ hierarchy, allow Loggers to target consumers in a number of
// different forms directly.
//
// The send.Base implementation provides implementations for all
// methods in the interface except Send. Most implementations will
// only implement Send, and sometimes Flush, in addition to
// exposing/composing Base.
type Sender interface {
	// Name returns the name of the logging system. Typically this
	// corresponds directly with the underlying logging capture system.
	Name() string
	//SetName sets the name of the logging system.
	SetName(string)

	// Method that actually sends messages (the string) to the logging
	// capture system. The Send() method filters out logged messages based
	// based on priority, typically using the generic
	// MessageInfo.ShouldLog() function.
	Send(message.Composer)

	// Flush flushes any potential buffered messages to the logging capture
	// system. If the Sender is not buffered, this function should noop and
	// return nil.
	Flush(context.Context) error

	// SetPriority sets the threshold of the sender. Typically,
	// loggers will not send messages that have a priority less
	// than this level.
	SetPriority(level.Priority)

	// Level returns the currently configured level for the sender.
	Priority() level.Priority

	// SetErrorHandler provides a method to inject error handling behavior
	// to a sender. Not all sender implementations use the error handler,
	// although some, use a default handler to write logging errors to
	// standard output.
	SetErrorHandler(ErrorHandler)
	ErrorHandler() ErrorHandler

	// SetFormatter allows users to inject formatting functions to modify
	// the output of the log sender by providing a function that takes a
	// message and returns string and error.
	SetFormatter(MessageFormatter)
	Formatter() MessageFormatter

	// SetConverter allows users to inject a custom converter into
	// the sender to be used by the logging infrastructure to make
	// message.Composer objects from arbitrary input types.
	SetConverter(CustomMessageConverter)
	Converter() MessageConverter

	// If the logging sender holds any resources that require desecration
	// they should be cleaned up in the Close() method. Close() is called
	// by the SetSender() method before changing loggers. Sender implementations
	// that wrap other Senders may or may not close their underlying Senders.
	Close() error
}

// ErrorHandler is a function that you can use define how a sender
// handles errors sending messages. Implementations of this type
// should perform a noop if the err object is nil.
type ErrorHandler func(error, message.Composer)

// MessageFormatter is a function type used by senders to construct the
// entire string returned as part of the output. This makes it
// possible to modify the logging format without needing to implement
// new Sender interfaces.
type MessageFormatter func(message.Composer) (string, error)

// CustomMessageConverter is a function that users can inject into
// their sender that the grip.Logger will use to convert arbitrary
// input types into message objects. If the second value is false, the
// output message will not be used and the logger will fall back to
// using `message.Convert`. Implementing a custom converter is optional.
type CustomMessageConverter func(any) (message.Composer, bool)

// MessageConverter defines the converter provided by the sender to
// higher level interfaces (e.g. grip.Logger) that will always produce
// a valid message.Composer from an arbitrary input.
type MessageConverter func(any) message.Composer

// MakeStandard produces a standard library logging instance that
// write to the underlying sender.
func MakeStandard(s Sender) *log.Logger { return log.New(MakeWriter(s), "", 0) }

// FromStandard prodeces a sender implementation from the standard
// library logger.
func FromStandard(logger *log.Logger) Sender { return WrapWriter(logger.Writer()) }

func ShouldLog(s Sender, m message.Composer) bool {
	if m == nil || !m.Loggable() || s == nil {
		return false
	}
	mp := m.Priority()
	return mp != level.Invalid && mp >= s.Priority()
}
