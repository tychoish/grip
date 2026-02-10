package message

import (
	"fmt"
	"iter"

	"github.com/tychoish/fun/adt"
	"github.com/tychoish/fun/dt"
	"github.com/tychoish/fun/fn"
	"github.com/tychoish/fun/irt"
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
func Convert[T any](input T) Composer {
	switch message := any(input).(type) {
	case nil:
		return Noop()
	case Composer:
		return message
	case string:
		return MakeString(message)
	case error:
		return MakeError(message)
	case Fields:
		return MakeFields(message)
	case Marshaler:
		return MakeFuture(message.MarshalComposer)

	case []string:
		return NewLines(message)
	case []Composer:
		return MakeGroupComposer(message)
	case []Fields:
		return convertSlice(message)
	case []any:
		return buildFromSlice(message)
	case []error:
		return JoinErrors(irt.Slice(message))
	case []irt.KV[string, any]:
		return MakeKV(irt.KVsplit(irt.Slice(message)))
	case []irt.KV[string, string]:
		return MakeKV(irt.KVsplit(irt.Slice(message)))
	case []byte:
		return MakeBytes(message)
	case *dt.List[string]:
		return CreateGroupComposer(irt.Convert(message.IteratorFront(), Convert))
	case dt.List[string]:
		return CreateGroupComposer(irt.Convert(message.IteratorFront(), Convert))
	case *dt.OrderedSet[string]:
		return CreateGroupComposer(irt.Convert(message.Iterator(), Convert))
	case iter.Seq[string]:
		return CreateGroupComposer(irt.Convert(message, Convert))
	case iter.Seq[any]:
		return CreateGroupComposer(irt.Convert(message, Convert))
	case iter.Seq[*KV]:
		return CreateGroupComposer(irt.Convert(message, Convert))

	case map[string]any:
		return MakeFields(Fields(message))
	case map[string]string:
		return MakeKV(irt.Map(message))
	case adt.OrderedMap[string, any]:
		return MakeKV(message.Iterator())
	case adt.OrderedMap[string, string]:
		return MakeKV(message.Iterator())
	case dt.OrderedMap[string, any]:
		return MakeKV(message.Iterator())
	case dt.OrderedMap[string, string]:
		return MakeKV(message.Iterator())
	case *adt.OrderedMap[string, any]:
		return MakeKV(message.Iterator())
	case *adt.OrderedMap[string, string]:
		return MakeKV(message.Iterator())
	case *dt.OrderedMap[string, any]:
		return MakeKV(message.Iterator())
	case *dt.OrderedMap[string, string]:
		return MakeKV(message.Iterator())

	case []*dt.List[string]:
		return convertSlice(message)
	case []dt.List[string]:
		return convertSlice(message)
	case []*dt.OrderedSet[string]:
		return convertSlice(message)
	case []iter.Seq[string]:
		return convertSlice(message)
	case []iter.Seq[any]:
		return convertSlice(message)
	case []*KV:
		return convertSlice(message)
	case []iter.Seq[*KV]:
		return convertSlice(message)
	case func() dt.List[string]:
		return MakeFuture(message)
	case func() *dt.List[string]:
		return MakeFuture(message)
	case func() *dt.OrderedSet[string]:
		return MakeFuture(message)
	case func() iter.Seq[string]:
		return MakeFuture(message)
	case func() iter.Seq[*KV]:
		return MakeFuture(message)
	case func() []*dt.List[string]:
		return MakeFuture(message)
	case func() []dt.List[string]:
		return MakeFuture(message)
	case func() []*dt.OrderedSet[string]:
		return MakeFuture(message)
	case func() []iter.Seq[string]:
		return MakeFuture(message)
	case func() []string:
		return MakeFuture(message)
	case func() []error:
		return MakeFuture(message)
	case func() []*KV:
		return MakeFuture(message)
	case fn.Future[dt.List[string]]:
		return MakeFuture(message)
	case fn.Future[*dt.List[string]]:
		return MakeFuture(message)
	case fn.Future[*dt.OrderedSet[string]]:
		return MakeFuture(message)
	case fn.Future[iter.Seq[string]]:
		return MakeFuture(message)
	case fn.Future[iter.Seq[*KV]]:
		return MakeFuture(message)
	case fn.Future[[]string]:
		return MakeFuture(message)
	case fn.Future[[]error]:
		return MakeFuture(message)
	case fn.Future[[]*KV]:
		return MakeFuture(message)
	case fn.Future[[]dt.List[string]]:
		return MakeFuture(message)
	case fn.Future[[]*dt.List[string]]:
		return MakeFuture(message)
	case fn.Future[[]*dt.OrderedSet[string]]:
		return MakeFuture(message)
	case []func() dt.List[string]:
		return convertSlice(message)
	case []func() *dt.List[string]:
		return convertSlice(message)
	case []func() *dt.OrderedSet[string]:
		return convertSlice(message)
	case []func() iter.Seq[string]:
		return convertSlice(message)
	case []func() iter.Seq[*KV]:
		return convertSlice(message)
	case []func() []error:
		return convertSlice(message)
	case []func() []string:
		return convertSlice(message)
	case []func() []*KV:
		return convertSlice(message)
	case []func() []*dt.List[string]:
		return convertSlice(message)
	case []func() []dt.List[string]:
		return convertSlice(message)
	case []func() []*dt.OrderedSet[string]:
		return convertSlice(message)
	case []func() []iter.Seq[string]:
		return convertSlice(message)
	case []fn.Future[dt.List[string]]:
		return convertSlice(message)
	case []fn.Future[*dt.List[string]]:
		return convertSlice(message)
	case []fn.Future[*dt.OrderedSet[string]]:
		return convertSlice(message)
	case []fn.Future[iter.Seq[*KV]]:
		return convertSlice(message)
	case []fn.Future[iter.Seq[string]]:
		return convertSlice(message)
	case []fn.Future[[]*dt.List[string]]:
		return convertSlice(message)
	case []fn.Future[[]dt.List[string]]:
		return convertSlice(message)
	case []fn.Future[[]*dt.OrderedSet[string]]:
		return convertSlice(message)
	case []fn.Future[[]string]:
		return convertSlice(message)
	case []fn.Future[[]error]:
		return convertSlice(message)
	case []fn.Future[[]*KV]:
		return convertSlice(message)

	case iter.Seq[error]:
		return CreateGroupComposer(irt.Convert(message, MakeError))
	case iter.Seq2[string, any]:
		return MakeKV(message)
	case iter.Seq2[string, string]:
		return MakeKV(message)
	case iter.Seq[irt.KV[string, any]]:
		return MakeKV(irt.KVsplit(message))
	case iter.Seq[irt.KV[string, string]]:
		return MakeKV(irt.KVsplit(message))

	case fn.Future[[]irt.KV[string, any]]:
		return MakeFuture(message)
	case fn.Future[[]irt.KV[string, string]]:
		return MakeFuture(message)

	case fn.Future[Fields]:
		return MakeFuture(message)
	case fn.Future[*KV]:
		return MakeFuture(message)
	case fn.Future[Composer]:
		return MakeFuture(message)
	case fn.Future[error]:
		return MakeFuture(message)
	case fn.Future[map[string]any]:
		return MakeFuture(message)
	case fn.Future[map[string]string]:
		return MakeFuture(message)

	case fn.Future[adt.OrderedMap[string, any]]:
		return MakeFuture(message)
	case fn.Future[adt.OrderedMap[string, string]]:
		return MakeFuture(message)
	case fn.Future[dt.OrderedMap[string, string]]:
		return MakeFuture(message)
	case fn.Future[dt.OrderedMap[string, any]]:
		return MakeFuture(message)

	case fn.Future[iter.Seq2[string, any]]:
		return MakeFuture(message)
	case fn.Future[iter.Seq2[string, string]]:
		return MakeFuture(message)
	case fn.Future[iter.Seq[irt.KV[string, any]]]:
		return MakeFuture(message)
	case fn.Future[iter.Seq[irt.KV[string, string]]]:
		return MakeFuture(message)
	case fn.Future[iter.Seq[error]]:
		return MakeFuture(message)

	case func() Fields:
		return MakeFuture(message)
	case func() Composer:
		return MakeFuture(message)
	case func() map[string]any:
		return MakeFuture(message)
	case func() map[string]string:
		return MakeFuture(message)
	case func() *KV:
		return MakeFuture(message)
	case func() error:
		return MakeFuture(message)
	case func() iter.Seq2[string, any]:
		return MakeFuture(message)
	case func() iter.Seq2[string, string]:
		return MakeFuture(message)
	case func() iter.Seq[irt.KV[string, any]]:
		return MakeFuture(message)
	case func() iter.Seq[irt.KV[string, string]]:
		return MakeFuture(message)
	case func() iter.Seq[error]:
		return MakeFuture(message)
	case func() adt.OrderedMap[string, any]:
		return MakeFuture(message)
	case func() adt.OrderedMap[string, string]:
		return MakeFuture(message)
	case func() dt.OrderedMap[string, string]:
		return MakeFuture(message)
	case func() dt.OrderedMap[string, any]:
		return MakeFuture(message)
	case func() []irt.KV[string, any]:
		return MakeFuture(message)
	case func() []irt.KV[string, string]:
		return MakeFuture(message)

	case []adt.OrderedMap[string, any]:
		return convertSlice(message)
	case []dt.OrderedMap[string, any]:
		return convertSlice(message)
	case []dt.OrderedMap[string, string]:
		return convertSlice(message)
	case []adt.OrderedMap[string, string]:
		return convertSlice(message)
	case []*adt.OrderedMap[string, any]:
		return convertSlice(message)
	case []*dt.OrderedMap[string, any]:
		return convertSlice(message)
	case []*dt.OrderedMap[string, string]:
		return convertSlice(message)
	case []*adt.OrderedMap[string, string]:
		return convertSlice(message)
	case []interface{ Iterator() iter.Seq2[string, any] }:
		return convertSlice(message)

	case [][]string:
		return convertSlice(message)
	case [][]byte:
		return convertSlice(message)
	case []map[string]any:
		return convertSlice(message)
	case []map[string]string:
		return convertSlice(message)
	case [][]any:
		return convertSlice(message)
	case []Marshaler:
		return convertSlice(message)

	case []fn.Future[Fields]:
		return convertSlice(message)
	case []fn.Future[Composer]:
		return convertSlice(message)
	case []fn.Future[*KV]:
		return convertSlice(message)
	case []fn.Future[error]:
		return convertSlice(message)
	case []fn.Future[map[string]any]:
		return convertSlice(message)
	case []fn.Future[map[string]string]:
		return convertSlice(message)

	case []fn.Future[iter.Seq[error]]:
		return convertSlice(message)
	case []fn.Future[iter.Seq2[string, any]]:
		return convertSlice(message)
	case []fn.Future[iter.Seq2[string, string]]:
		return convertSlice(message)
	case []fn.Future[iter.Seq[irt.KV[string, any]]]:
		return convertSlice(message)
	case []fn.Future[iter.Seq[irt.KV[string, string]]]:
		return convertSlice(message)

	case []fn.Future[adt.OrderedMap[string, any]]:
		return convertSlice(message)
	case []fn.Future[adt.OrderedMap[string, string]]:
		return convertSlice(message)
	case []fn.Future[dt.OrderedMap[string, string]]:
		return convertSlice(message)
	case []fn.Future[dt.OrderedMap[string, any]]:
		return convertSlice(message)

	case []func() Fields:
		return convertSlice(message)
	case []func() Composer:
		return convertSlice(message)
	case []func() *KV:
		return convertSlice(message)
	case []func() error:
		return convertSlice(message)
	case []func() iter.Seq[error]:
		return convertSlice(message)
	case []func() map[string]any:
		return convertSlice(message)
	case []func() map[string]string:
		return convertSlice(message)
	case []func() iter.Seq2[string, any]:
		return convertSlice(message)
	case []func() iter.Seq2[string, string]:
		return convertSlice(message)
	case []func() iter.Seq[irt.KV[string, any]]:
		return convertSlice(message)
	case []func() iter.Seq[irt.KV[string, string]]:
		return convertSlice(message)
	case []func() adt.OrderedMap[string, any]:
		return convertSlice(message)
	case []func() adt.OrderedMap[string, string]:
		return convertSlice(message)
	case []func() dt.OrderedMap[string, string]:
		return convertSlice(message)
	case []func() dt.OrderedMap[string, any]:
		return convertSlice(message)
	// interfaces
	case interface{ Iterator() iter.Seq[string] }:
		return Convert(message.Iterator)
	case interface {
		Iterator() iter.Seq2[string, string]
	}:
		return Convert(message.Iterator)
	case interface{ Iterator() iter.Seq[any] }:
		return Convert(message.Iterator)
	case fmt.Stringer:
		return MakeFuture(message.String)
	case interface{ Iterator() iter.Seq2[string, any] }:
		return MakeFuture(func() Composer { return NewKV().Extend(message.Iterator()) })
	case interface{ IsZero() bool }:
		if message.IsZero() {
			return Noop()
		}
		return MakeFormat("%+v", message)
	default:
		return MakeFormat("%+v", message)
	}
}

func convertSlice[T any](in []T) Composer {
	switch len(in) {
	case 0:
		return Noop()
	case 1:
		return Convert(in[0])
	default:
		return convertSequence(irt.Any(irt.Slice(in)))
	}
}

func convertSequence[T any](seq iter.Seq[T]) Composer {
	return CreateGroupComposer(irt.Convert(seq, Convert))
}

func buildFromSlice(vals []any) Composer {
	if len(vals) == 0 {
		return Noop()
	}

	// check to see that the even numbered items are strings, if
	// they're something else, convert them as a slice to a group
	// of something.
	for i := 0; i < len(vals); i += 2 {
		switch vals[i].(type) {
		case Composer, fn.Future[Composer], fn.Future[error], fn.Future[Fields], Fields, iter.Seq2[string, any], *KV:
			return convertSlice(vals)
		case []Composer, []fn.Future[Composer], []fn.Future[error], []fn.Future[Fields], []error, []Fields, []iter.Seq2[string, any], []*KV, []KV:
			return convertSlice(vals)
		case string, fmt.Stringer:
			continue
		default:
			return MakeLines(vals...)
		}
	}

	if len(vals)%2 != 0 {
		return MakeLines(vals...)
	}

	return MakeKV(func(yield func(string, any) bool) {
		for i := 0; i < len(vals); i += 2 {
			if !yield(fmt.Sprint(vals[i]), vals[i+1]) {
				return
			}
		}
	})
}
