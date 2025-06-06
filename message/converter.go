package message

import (
	"fmt"

	"github.com/tychoish/fun/dt"
	"github.com/tychoish/fun/fn"
)

// Converter is an interface for converting arbitrary types to
// Composers. Like the http.Handler interface, the primary form of
// implementing the interface is by ConverterFunc itself, which
// implements Converter.
//
// The DefaultConverter function produces a wrapper around the Convert
// function which produces all logging messages.
type Converter interface{ Convert(any) Composer }

// ConverterFunc is a function that users can inject into
// their sender that the grip.Logger will use to convert arbitrary
// input types into message objects. If the second value is false, the
// output message will not be used and the logger will fall back to
// using message.Convert.
type ConverterFunc func(any) (Composer, bool)

func (cf ConverterFunc) Convert(m any) Composer {
	switch {
	case cf != nil:
		if out, ok := cf(m); ok {
			return out
		}
		fallthrough
	default:
		return Convert(m)
	}
}

// DefaultConverter is a Converter implementation around the Convert
// function.
func DefaultConverter() Converter { return defaultConverter{} }

type defaultConverter struct{}

func (defaultConverter) Convert(m any) Composer { return Convert(m) }

// Convert produces a composer interface for arbitrary input.
//
// Use this directly in your implementation of the Converter
// interface. The DefaultConverter implementation provides a wrapper
// around this implementation. The ConverterFunc-based implementations
// fall back to this implementation
func Convert(input any) Composer {
	switch message := input.(type) {
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
	case *dt.Pairs[string, any]:
		return MakePairs(message)
	case dt.Pair[string, any]:
		return MakeKV(message)
	case []dt.Pair[string, any]:
		return MakeKV(message...)
	case nil:
		return MakeKV()
	case map[string]any:
		return MakeFields(Fields(message))
	case []byte:
		return MakeBytes(message)
	case fn.Future[Fields]:
		return MakeFuture(message)
	case fn.Future[*dt.Pairs[string, any]]:
		return MakeFuture(message)
	case func() Fields:
		return MakeFuture(message)
	case fn.Future[Composer]:
		return MakeFuture(message)
	case func() Composer:
		return MakeFuture(message)
	case func() *dt.Pairs[string, any]:
		return MakeFuture(message)
	case func() map[string]any:
		return MakeFuture(message)
	case fn.Future[error]:
		return MakeFuture(message)
	case func() error:
		return MakeFuture(message)
	case Marshaler:
		return MakeFuture(message.MarshalComposer)
	case [][]string:
		return convertSlice(message)
	case [][]byte:
		return convertSlice(message)
	case []map[string]any:
		return convertSlice(message)
	case []Fields:
		return convertSlice(message)
	case []fn.Future[Fields]:
		return convertSlice(message)
	case []func() Fields:
		return convertSlice(message)
	case []func() map[string]any:
		return convertSlice(message)
	case []fn.Future[Composer]:
		return convertSlice(message)
	case []func() Composer:
		return convertSlice(message)
	case []fn.Future[error]:
		return convertSlice(message)
	case []func() error:
		return convertSlice(message)
	case [][]any:
		return convertSlice(message)
	case []*dt.Pairs[string, any]:
		return convertSlice(message)
	case []Marshaler:
		return convertSlice(message)
	// case interface{ IsZero() bool }:
	// 	if message.IsZero() {
	// 		return MakeKV()
	// 	}
	//
	// 	return MakeFormat("%+v", message)
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
		case Composer, fn.Future[Composer], fn.Future[error], fn.Future[Fields], Fields, dt.Pairs[string, any]:
			return convertSlice(vals)
		case []Composer, []fn.Future[Composer], []fn.Future[error], []fn.Future[Fields], []error, []Fields, []dt.Pairs[string, any]:
			return convertSlice(vals)
		default:
			return MakeLines(vals...)
		}
	}

	if len(vals)%2 != 0 {
		return MakeLines(vals...)
	}

	fields := &dt.Pairs[string, any]{}

	for i := 0; i < len(vals); i += 2 {
		fields.Add(fmt.Sprint(vals[i]), vals[i+1])
	}

	return MakePairs(fields)
}
