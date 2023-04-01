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

// The Sender interface describes how the Journaler type's method in primary
// "grip" package's methods interact with a logging output method. The
// Journaler type provides Sender() and SetSender() methods that allow client
// code to swap logging backend implementations dependency-injection style.
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

	// If the logging sender holds any resources that require desecration
	// they should be cleaned up in the Close() method. Close() is called
	// by the SetSender() method before changing loggers. Sender implementations
	// that wrap other Senders may or may not close their underlying Senders.
	Close() error
}

// MakeStandard produces a standard library logging instance that
// write to the underlying sender.
func MakeStandard(s Sender) *log.Logger { return log.New(MakeWriter(s), "", 0) }

// FromStandard prodeces a sender implementation from the standard
// library logger.
func FromStandard(logger *log.Logger) Sender { return WrapWriter(logger.Writer()) }

func ShouldLog(s Sender, m message.Composer) bool {
	if !m.Loggable() {
		return false
	}
	mp := m.Priority()
	return mp != level.Invalid && mp >= s.Priority()
}
