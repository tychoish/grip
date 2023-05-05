package message

import (
	"fmt"

	"github.com/tychoish/grip/level"
)

// Composer defines an interface with a "String()" method that
// returns the message in string format. Objects that implement this
// interface, in combination to the Compose[*] operations, the
// String() method is only caled if the priority of the method is
// greater than the threshold priority. This makes it possible to
// defer building log messages (that may be somewhat expensive to
// generate) until it's certain that we're going to be outputting the
// message.
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
}

// Convert produces a composer interface for arbitrary input.
func Convert[T any](input T) Composer {
	switch message := any(input).(type) {
	case Composer:
		return message
	case []Composer:
		return MakeGroupComposer(message)
	case string:
		return MakeString(message)
	case []string:
		return newLinesFromStrings(message)
	case []any:
		return buildFromSlice(message)
	case error:
		return MakeError(message)
	case Fields:
		return MakeFields(message)
	case KVs:
		return MakeKVs(message)
	case []KV:
		return MakeKVs(message)
	case nil:
		return MakeKV()
	case map[string]any:
		return MakeFields(Fields(message))
	case []byte:
		return MakeBytes(message)
	case FieldsProducer:
		return MakeProducer(message)
	case func() Fields:
		return MakeProducer(message)
	case ComposerProducer:
		return MakeProducer(message)
	case func() Composer:
		return MakeProducer(message)
	case func() map[string]any:
		return MakeProducer(message)
	case ErrorProducer:
		return MakeProducer(message)
	case func() error:
		return MakeProducer(message)
	case Marshaler:
		return MakeProducer(message.MarshalComposer)
	case [][]string:
		return convertSlice(message)
	case [][]byte:
		return convertSlice(message)
	case []map[string]any:
		return convertSlice(message)
	case []Fields:
		return convertSlice(message)
	case []FieldsProducer:
		return convertSlice(message)
	case []func() Fields:
		return convertSlice(message)
	case []func() map[string]any:
		return convertSlice(message)
	case []ComposerProducer:
		return convertSlice(message)
	case []func() Composer:
		return convertSlice(message)
	case []ErrorProducer:
		return convertSlice(message)
	case []func() error:
		return convertSlice(message)
	case [][]any:
		return convertSlice(message)
	case []KVs:
		return convertSlice(message)
	case []Marshaler:
		return convertSlice(message)
	default:
		return MakeFormat("%+v", message)
	}
}

func convertSlice[T any](in []T) Composer {
	switch len(in) {
	case 0:
		return MakeKV()
	case 1:
		return Convert(in[0])
	default:
		out := make([]Composer, len(in))
		for idx := range in {
			out[idx] = Convert(in[idx])
		}
		return MakeGroupComposer(out)
	}
}

func buildFromSlice(vals []any) Composer {
	if len(vals) == 0 {
		return MakeKV()
	}

	// check to see that the even numbered items are strings, if
	// they're something else, convert them as a slice to a group
	// of something.
	for i := 0; i < len(vals); i += 2 {
		switch vals[i].(type) {
		case string:
			continue
		case Composer, ComposerProducer, ErrorProducer, Fields, KVs, []KV:
			return convertSlice(vals)
		case []Composer, []ComposerProducer, []ErrorProducer, []Fields:
			return convertSlice(vals)
		default:
			return MakeLines(vals...)
		}
	}

	if len(vals)%2 != 0 {
		return MakeLines(vals...)
	}

	fields := make(KVs, 0, len(vals)/2)
	for i := 0; i < len(vals); i += 2 {
		fields = append(fields, KV{
			Key:   fmt.Sprint(vals[i]),
			Value: vals[i+1],
		})
	}

	return MakeKVs(fields)
}
