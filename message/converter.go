package message

import "fmt"

// ConverterFunc is a function that users can inject into
// their sender that the grip.Logger will use to convert arbitrary
// input types into message objects. If the second value is false, the
// output message will not be used and the logger will fall back to
// using message.Convert.
type ConverterFunc func(any) (Composer, bool)

// Convert produces a composer interface for arbitrary input.
//
// The result is almost never (typed nil values may pass thorugh)
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
		m := MakeKV()
		m.SetOption(OptionSkipAllMetadata)
		return m
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
	// case interface{ IsZero() bool }:
	// 	if message.IsZero() {
	// 		return MakeKV()
	// 	}

	// 	return MakeFormat("%+v", message)
	default:
		return MakeFormat("%+v", message)
	}
}

func convertSlice[T any](in []T) Composer {
	switch len(in) {
	case 0:
		m := MakeKV()
		m.SetOption(OptionSkipAllMetadata)
		return m
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
		m := MakeKV()
		m.SetOption(OptionSkipAllMetadata)
		return m
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
