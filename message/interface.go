package message

import (
	"github.com/tychoish/grip/level"
)

// Composer defines an interface with a String() method that returns
// the message in string format, as well as a Raw() method that may
// provide a structured form of the message. Objects that implement
// this interface, the String() method is only caled if the priority
// of the method is greater than the threshold priority. This makes it
// possible to defer building log messages (that may be somewhat
// expensive to generate) until it's certain that they will be
// consumed.
//
// Most implementations will only implement String() and Raw() and
// rely on the message.Base type which can be composed and provides
// basic implementations for types.
type Composer interface {
	// Returns the content of the message as a string for use in
	// line-printing logging engines.
	String() string

	// A "raw" format of the logging output for use by some Sender
	// implementations that write logged items to interfaces that
	// accept JSON or another structured format.
	Raw() any

	// Returns "true" when the message has content and should be
	// logged, and false otherwise. When false, the sender can
	// (and should!) ignore messages even if they are otherwise
	// above the logging threshold.
	Loggable() bool

	// Returns "true" when the underlying message type has
	// substantial structured data and should be handled by the
	// sender as structured data.
	Structured() bool

	// Annotate makes it possible for users (including internally)
	// to add structured data to a log message. Implementations may
	// choose to override key/value pairs that already exist.
	Annotate(string, any)

	// Priority returns the priority of the message.
	Priority() level.Priority

	// SetPriority sets the messaages' log level. The high level
	// logging interfaces set this before sending the
	// message. If you send a message to a sender directly without
	// setting the level, or set the level to an invalid level,
	// the message is not loggable.
	SetPriority(level.Priority)

	// AttachMetadata is used by send.Sender implementations and
	// send.Formater implementations to control the output format
	// and processing of messages. These options may be defined in
	// other packages, and implementations are under no obligation
	// to respect them. In the case where two options that
	// contradict eachother, the last one should win.
	SetOption(...Option)
}

// Options control the behavior and output of a message, specifically
// the String and Raw methods. Implementations are responsible for
// compliance with the options. The `Base` type provides basic support
// for setting and exposing these options to implementations.
type Option string

const (
	// OptionIncludeMetadata tells the message to annotate itself
	// basic metadata to a message.
	OptionIncludeMetadata Option = "include-metadata"
	// OptionSkipMetadata disables the inclusion of metadata
	// in the output messaage. This is typically the default in
	// most implementations.
	OptionSkipMetadata Option = "skip-metadata"
	// OptionCollectInfo enables collecting extra data
	// (implemented by the Base type) including hostname, process
	// name, and time. While this information is cached and not
	// difficult to collect, it may increase the message payload
	// with unneeded data.
	OptionCollectInfo Option = "collect-info"
	// OptionSkipCollect tells the message, typically for Raw()
	// calls to *not* call the message/Base.Collect method which
	// annotates fields about the host system and level. This is
	// typically the default.
	OptionSkipCollectInfo Option = "skip-collect-info"
	// OptionMessageIsNotStructuredField indicates to the
	// implementor that the Message field name in Fields-typed
	// messages (defined by the message.FieldsMsgName constant)
	// should *not* be handdled specially.
	OptionMessageIsNotStructuredField Option = "message-is-not-structured"
)
