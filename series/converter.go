package series

import (
	"iter"
	"maps"
	"strings"

	"github.com/tychoish/fun/adt"
	"github.com/tychoish/fun/dt"
	"github.com/tychoish/fun/fn"
	"github.com/tychoish/fun/irt"
	"github.com/tychoish/fun/stw"
	"github.com/tychoish/grip/message"
)

// EventExtractor is a type that is implementable by arbitrary types
// to create events.
type EventExtractor interface {
	Events() []*Event
}

// MetricMessage is a collection of events and a message.Composer
// object that can be used as a message.Composer but that also
// contains some number of events.
type MetricMessage struct {
	message.Composer
	Events []*Event
}

func (e *Event) Export() Record {
	labels := &dt.OrderedMap[string, string]{}
	for k, v := range e.m.labelCache() {
		labels.Store(k, v)
	}
	return Record{
		ID:     e.m.ID,
		Value:  e.value,
		Labels: labels,
	}
}

type Record struct {
	ID     string                         `bson:"metric" json:"metric" yaml:"metric"`
	Value  int64                          `bson:"Value" json:"Value" yaml:"Value"`
	Labels *dt.OrderedMap[string, string] `bson:"labels" json:"labels" yaml:"labels"`
}

func (m *MetricMessage) Structured() bool { return true }

func (m *MetricMessage) String() string {
	out := make([]string, 0, len(m.Events)+1)
	out = append(out, m.Composer.String())
	for _, ev := range m.Events {
		out = append(out, ev.String())
	}
	return strings.Join(out, "; ")
}

func (m *MetricMessage) Raw() any {
	return struct {
		Message any      `bson:"message" json:"message" yaml:"message"`
		Events  []Record `bson:"events,omitempty" json:"events,omitempty" yaml:"events,omitempty"`
	}{
		Message: m.Composer.Raw(),
		Events: func() (out []Record) {
			for _, ev := range m.Events {
				out = append(out, ev.Export())
			}
			return
		}(),
	}
}

// Message is a simple constructor around *MetricMessage (which
// implements message.Composer) and includes a slice of event
// pointers.
func Message(m message.Composer, events ...*Event) *MetricMessage {
	return &MetricMessage{
		Composer: m,
		Events:   events,
	}
}

// Extract takes an arbitrary object and attempts to introspect it to
// find events.
func Extract(c any) []*Event { return extractMetrics(c, metricMessageWithOnlyEvents).Events }

// WithMetrics inspects a value that might have *Event, (or related
// types, including functions that produce events and slices of
// events) embedded in them.
func WithMetrics(c any, events ...*Event) message.Composer {
	msg := extractMetrics(c, metricMessageWithComposer)
	msg.Events = append(msg.Events, events...)
	return msg
}

////////////////////////////////////////////////////////////////////////
//
// Machinery...

type extractableMessageTypes interface {
	any | []any | ~map[string]any | irt.KV[string, any] |
		iter.Seq2[string, any] | iter.Seq[irt.KV[string, any]] |
		*adt.Map[string, any] | *adt.OrderedMap[string, any] |
		*dt.Set[irt.KV[string, any]] | *dt.OrderedSet[irt.KV[string, any]] |
		*adt.OrderedSet[irt.KV[string, any]] | *adt.Set[irt.KV[string, any]]
}

type eventObjects interface {
	Event | ~*Event | ~[]*Event | ~[]Event
}

type extractableMessageFutures[T extractableMessageTypes | eventObjects] interface{ ~func() T }

type metricMessageExtractOption bool

const (
	metricMessageWithComposer   metricMessageExtractOption = false
	metricMessageWithOnlyEvents metricMessageExtractOption = true
)

func extractMetrics[T eventObjects | extractableMessageTypes | extractableMessageFutures[T]](
	msg T,
	buildMessage metricMessageExtractOption,
) *MetricMessage {
	if out, ok := any(msg).(*MetricMessage); ok {
		return out
	}

	if isEventTyped(msg) {
		return &MetricMessage{Events: getEvents(msg)}
	}

	if !hasMetrics(msg) {
		return &MetricMessage{Composer: message.Convert(msg)}
	}
	val := resolveEvents(msg, buildMessage)
	return &val
}

func isEventTyped(in any) bool {
	switch in.(type) {
	case Event, *Event, []Event, []*Event:
		return true
	case func() Event, func() *Event, func() []Event, func() []*Event:
		return true
	case EventExtractor, MetricMessage, *MetricMessage:
		return true
	default:
		return false
	}
}

func getEvents(in any) []*Event {
	switch ev := in.(type) {
	case []*Event:
		return ev
	case *Event:
		return []*Event{ev}
	case Event:
		return []*Event{&ev}
	case []Event:
		out := make([]*Event, 0, len(ev))
		for idx := range ev {
			out[idx] = stw.Ptr(ev[idx])
		}
		return out
	case EventExtractor:
		return ev.Events()
	case *MetricMessage:
		return ev.Events
	case MetricMessage:
		return ev.Events
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
	case Event, *Event, []Event, []*Event, EventExtractor:
		return true
	case MetricMessage, *MetricMessage:
		return true
	case func() Event, func() *Event, func() []Event, func() []*Event:
		return true
	case fn.Future[Event], fn.Future[*Event], fn.Future[[]Event], fn.Future[[]*Event]:
		return true
	case map[string]any: // also mesage.Fields
		for v := range maps.Values(ev) {
			if isEventTyped(v) {
				return true
			}
		}
	case iter.Seq2[string, any]:
		for v := range ev {
			if isEventTyped(v) {
				return true
			}
		}
	case []irt.KV[string, any]:
		for v := range irt.Second(irt.KVsplit(irt.Slice(ev))) {
			if isEventTyped(v) {
				return true
			}
		}
	case []*irt.KV[string, any]:
		for v := range irt.Second(irt.KVsplit(irt.DerefWithZeros(irt.Slice(ev)))) {
			if isEventTyped(v) {
				return true
			}
		}
	case []any:
		for v := range irt.Slice(ev) {
			if isEventTyped(v) {
				return true
			}
		}
	case any:
		return isEventTyped(ev)
	}
	return false
}

func resolveEvents(in any, buildMessage metricMessageExtractOption) (out MetricMessage) {
	var p []irt.KV[string, any]

	switch msg := in.(type) {
	case Event, *Event, []Event, []*Event:
		out.Events = getEvents(msg)
	case func() Event, func() *Event, func() []Event, func() []*Event:
		out.Events = getEvents(msg)
	case map[string]any:
		if buildMessage {
			p = []irt.KV[string, any]{}
		}

		for k, v := range msg {
			if isEventTyped(v) {
				out.Events = append(out.Events, getEvents(v)...)
				continue
			}
			if buildMessage {
				p = append(p, irt.KV[string, any]{Key: k, Value: v})
			}
		}
	case iter.Seq2[string, any]:
		if buildMessage {
			p = []irt.KV[string, any]{}
		}

		for k, v := range msg {
			if isEventTyped(v) {
				out.Events = append(out.Events, getEvents(v)...)
				return
			}
			if buildMessage {
				p = append(p, irt.KV[string, any]{Key: k, Value: v})
			}
		}
	case []irt.KV[string, any]:
		if buildMessage {
			p = []irt.KV[string, any]{}
		}

		for _, item := range msg {
			if isEventTyped(item.Value) {
				out.Events = append(out.Events, getEvents(item.Value)...)
				return
			}
			if buildMessage {
				p = append(p, item)
			}
		}
	case []*irt.KV[string, any]:
		if buildMessage {
			p = []irt.KV[string, any]{}
		}

		for _, item := range msg {
			if isEventTyped(item.Value) {
				out.Events = append(out.Events, getEvents(item.Value)...)
				return
			}
			if buildMessage {
				p = append(p, *item)
			}
		}
	case []any:
		var mm []any
		if buildMessage {
			mm = make([]any, 0, len(msg))
		}

		for _, in := range msg {
			if isEventTyped(in) {
				out.Events = append(out.Events, getEvents(in)...)
				return
			}
			if buildMessage {
				mm = append(mm, in)
			}
		}
		if buildMessage {
			out.Composer = message.Convert(mm)
		}
	}
	if buildMessage && p != nil && out.Composer == nil {
		out.Composer = message.Convert(p)
	}

	return
}
