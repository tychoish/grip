package graphite

import (
	"io"

	"github.com/tychoish/fun"
	"github.com/tychoish/fun/dt"
	"github.com/tychoish/fun/ft"
	"github.com/tychoish/fun/risky"
	"github.com/tychoish/grip/message"
)

// MetricMessage is a collection of events and a message.Composer
// object that can be used as a message.Composer but that also
// contains some number of events.
type MetricMessage struct {
	message.Composer
	Events []*Event
}

// WithMetrics inspects a value that might have *Event, (or related
// types, including functions that produce events and slices of
// events) embedded in them.
func WithMetrics(c any, events ...*Event) message.Composer {
	msg := extractMetrics(c)
	msg.Events = append(msg.Events, events...)
	return msg
}

////////////////////////////////////////////////////////////////////////
//
// Machinery...

type extractableMessageTypes interface {
	any | []any | *dt.Pairs[string, any] | ~map[string]any | []dt.Pair[string, any] | []*dt.Pair[string, any]
}

type eventObjects interface {
	Event | ~*Event | ~[]*Event | ~[]Event
}

type extractableMessageFutures[T extractableMessageTypes | eventObjects] interface{ ~func() T }

func extractMetrics[T eventObjects | extractableMessageTypes | extractableMessageFutures[T]](msg T) *MetricMessage {
	if out, ok := ft.Cast[*MetricMessage](msg); ok {
		return out
	}

	if ft.IgnoreSecond(isEventTyped(msg)) {
		return &MetricMessage{Events: getEvents(msg)}

	}
	if !hasMetrics(msg) {
		return &MetricMessage{Composer: message.Convert(msg)}
	}

	return ft.Ptr(resolveEvents(msg))
}

func isEventTyped(in any) (bool, error) {
	switch in.(type) {
	case Event, *Event, []Event, []*Event:
		return true, io.EOF
	case func() Event, func() *Event, func() []Event, func() []*Event:
		return true, io.EOF
	default:
		return false, nil
	}
}

func getEvents(in any) []*Event {
	switch ev := in.(type) {
	case Event:
		return []*Event{&ev}
	case *Event:
		return []*Event{ev}
	case []Event:
		out := make([]*Event, 0, len(ev))
		for idx := range ev {
			out[idx] = ft.Ptr(ev[idx])
		}
		return out
	case []*Event:
		return ev
	case func() Event:
		return getEvents(ev())
	case func() *Event:
		return getEvents(ev())
	case func() []Event:
		return getEvents(ev())
	case func() []*Event:
		return getEvents(ev())
	default:
		return nil
	}
}

func hasMetrics[T extractableMessageTypes](in T) (isMetric bool) {
	switch ev := any(in).(type) {
	case Event, *Event, []Event, []*Event:
		return true
	case func() Event, func() *Event, func() []Event, func() []*Event:
		return true
	case fun.Future[Event], fun.Future[*Event], fun.Future[[]Event], fun.Future[[]*Event]:
		return true
	case map[string]any: // also mesage.Fields
		dt.Mapify(ev).Values().Process(fun.MakeProcessor(func(in any) (err error) {
			isMetric, err = isEventTyped(in)
			return
		})).Ignore().Block()
	case *dt.Pairs[string, any]:
		ev.Values().Process(fun.MakeProcessor(func(in any) (err error) {
			isMetric, err = isEventTyped(in)
			return
		})).Ignore().Block()
	case []dt.Pair[string, any]:
		dt.Sliceify(ev).Iterator().Process(fun.MakeProcessor(func(in dt.Pair[string, any]) (err error) {
			isMetric, err = isEventTyped(in.Value)
			return
		})).Ignore().Block()
	case []*dt.Pair[string, any]:
		dt.Sliceify(ev).Iterator().Process(fun.MakeProcessor(func(in *dt.Pair[string, any]) (err error) {
			isMetric, err = isEventTyped(in.Value)
			return
		})).Ignore().Block()
	case []any:
		dt.Sliceify(ev).Iterator().Process(fun.MakeProcessor(func(in any) (err error) {
			isMetric, err = isEventTyped(in)
			return
		})).Ignore().Block()
	case any:
		isMetric = ft.IgnoreSecond(isEventTyped(ev))
	}
	return
}

func resolveEvents(in any) (out MetricMessage) {
	switch msg := in.(type) {
	case Event, *Event, []Event, []*Event:
		out.Events = getEvents(msg)
	case func() Event, func() *Event, func() []Event, func() []*Event:
		out.Events = getEvents(msg)
	case map[string]any:
		m := make(map[string]any, len(msg))
		for k, v := range msg {
			if ft.IgnoreSecond(isEventTyped(v)) {
				out.Events = append(out.Events, getEvents(v)...)
				continue
			}
			m[k] = v
		}
		out.Composer = message.Convert(m)
	case *dt.Pairs[string, any]:
		p := &dt.Pairs[string, any]{}
		risky.Observe(msg.Iterator(), func(item dt.Pair[string, any]) {
			if ft.IgnoreSecond(isEventTyped(item.Value)) {
				out.Events = append(out.Events, getEvents(item.Value)...)
				return
			}
			p.AddPair(item)
		})
		out.Composer = message.Convert(p)
	case []dt.Pair[string, any]:
		p := &dt.Pairs[string, any]{}
		risky.Observe(fun.SliceIterator(msg), func(item dt.Pair[string, any]) {
			if ft.IgnoreSecond(isEventTyped(item.Value)) {
				out.Events = append(out.Events, getEvents(item.Value)...)
				return
			}
			p.AddPair(item)
		})
		out.Composer = message.Convert(p)
	case []*dt.Pair[string, any]:
		p := &dt.Pairs[string, any]{}
		risky.Observe(fun.SliceIterator(msg), func(item *dt.Pair[string, any]) {
			if ft.IgnoreSecond(isEventTyped(item.Value)) {
				out.Events = append(out.Events, getEvents(item.Value)...)
				return
			}
			p.AddPair(*item)
		})
		out.Composer = message.Convert(p)
	case []any:
		mm := make([]any, 0, len(msg))

		risky.Observe(fun.SliceIterator(msg), func(in any) {
			if ft.IgnoreSecond(isEventTyped(in)) {
				out.Events = append(out.Events, getEvents(in)...)
				return
			}
			mm = append(mm, in)
		})
		out.Composer = message.Convert(mm)
	}
	return
}
