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

	// Annotate makes it possible for Senders and Journalers to
	// add structured data to a log message. May return an error
	// when the key alrady exists.
	Annotate(string, any) error

	// Priority returns the priority of the message.
	Priority() level.Priority
	SetPriority(level.Priority) error
}

// ConvertWithPriority can coerce unknown objects into Composer
// instances, as possible. This method will override the priority of
// composers set to it.
func ConvertWithPriority(p level.Priority, message any) Composer {
	if cmp, ok := message.(Composer); ok {
		if pri := cmp.Priority(); pri != level.Invalid {
			p = pri
		}
	}

	out := Convert(message)
	_ = out.SetPriority(p)

	return out
}

// Convert produces a composer interface for arbitrary input.
func Convert(message any) Composer {
	switch message := message.(type) {
	case Composer:
		return message
	case []Composer:
		return MakeGroupComposer(message)
	case string:
		return MakeString(message)
	case error:
		return MakeError(message)
	case FieldsProducer:
		return MakeFieldsProducer(message)
	case func() Fields:
		return MakeFieldsProducer(message)
	case ComposerProducer:
		return MakeProducer(message)
	case func() Composer:
		return MakeProducer(message)
	case func() map[string]any:
		return MakeConvertedFieldsProducer(message)
	case ErrorProducer:
		return MakeErrorProducer(message)
	case func() error:
		return MakeErrorProducer(message)
	case []string:
		return newLinesFromStrings(message)
	case []any:
		return buildFromSlice(message)
	case []byte:
		return MakeBytes(message)
	case Fields:
		return MakeFields(message)
	case KVs:
		return MakeKVs(message)
	case map[string]any:
		return MakeFields(Fields(message))
	case [][]string:
		grp := make([]Composer, len(message))
		for idx := range message {
			grp[idx] = newLinesFromStrings(message[idx])
		}
		return MakeGroupComposer(grp)
	case [][]byte:
		grp := make([]Composer, len(message))
		for idx := range message {
			grp[idx] = MakeBytes(message[idx])
		}
		return MakeGroupComposer(grp)
	case []map[string]any:
		grp := make([]Composer, len(message))
		for idx := range message {
			grp[idx] = MakeFields(message[idx])
		}
		return MakeGroupComposer(grp)
	case []Fields:
		grp := make([]Composer, len(message))
		for idx := range message {
			grp[idx] = MakeFields(message[idx])
		}
		return MakeGroupComposer(grp)
	case []FieldsProducer:
		grp := make([]Composer, len(message))
		for idx := range message {
			grp[idx] = MakeFieldsProducer(message[idx])
		}
		return MakeGroupComposer(grp)
	case []func() Fields:
		grp := make([]Composer, len(message))
		for idx := range message {
			grp[idx] = MakeFieldsProducer(message[idx])
		}
		return MakeGroupComposer(grp)
	case []func() map[string]any:
		grp := make([]Composer, len(message))
		for idx := range message {
			grp[idx] = MakeConvertedFieldsProducer(message[idx])
		}
		return MakeGroupComposer(grp)
	case []ComposerProducer:
		grp := make([]Composer, len(message))
		for idx := range message {
			grp[idx] = MakeProducer(message[idx])
		}
		return MakeGroupComposer(grp)
	case []func() Composer:
		grp := make([]Composer, len(message))
		for idx := range message {
			grp[idx] = MakeProducer(message[idx])
		}
		return MakeGroupComposer(grp)
	case []ErrorProducer:
		grp := make([]Composer, len(message))
		for idx := range message {
			grp[idx] = MakeErrorProducer(message[idx])
		}
		return MakeGroupComposer(grp)
	case []func() error:
		grp := make([]Composer, len(message))
		for idx := range message {
			grp[idx] = MakeErrorProducer(message[idx])
		}
		return MakeGroupComposer(grp)
	case [][]any:
		grp := make([]Composer, len(message))
		for idx := range message {
			grp[idx] = buildFromSlice(message[idx])
		}
		return MakeGroupComposer(grp)
	case []KVs:
		grp := make([]Composer, len(message))
		for idx := range message {
			grp[idx] = MakeKVs(message[idx])
		}
		return MakeGroupComposer(grp)
	case nil:
		return MakeKV()
	default:
		return MakeFormat("%+v", message)
	}
}

func buildFromSlice(vals []any) Composer {
	if len(vals)%2 != 0 {
		return MakeLines(vals...)
	}
	if len(vals) == 0 {
		return MakeKV()
	}

	for i := 0; i < len(vals); i += 2 {
		switch val := vals[i].(type) {
		case string:
			continue
		case fmt.Stringer:
			vals[i] = val.String()
		default:
			return MakeLines(vals...)
		}
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
