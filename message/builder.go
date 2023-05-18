package message

import (
	"errors"

	"github.com/tychoish/fun"
	"github.com/tychoish/fun/erc"
	"github.com/tychoish/grip/level"
)

// Builder provides a chainable message building interface.
//
// Builders can produce multiple messages. If the SetGroup value is
// true (also controlled via the Group/Ungroup methods,) then the Send
// operation is called once for the group of messages, and otherwise
// the send operation is called once for every constituent message
// (which is the default.)
//
// Callers must call Send() at the end of the builder chain to send
// the message.
type Builder struct {
	send        func(Composer)
	converter   Converter
	composer    Composer
	level       *level.Priority
	catcher     erc.Collector
	sendAsGroup bool
	opts        []Option
}

// NewBuilder constructs the chainable builder type, and initializes
// the error tracking and establishes a connection to the sender.
func NewBuilder(send func(Composer), convert Converter) *Builder {
	return &Builder{send: send, converter: convert}
}

// Send finalizes the chain and delivers the message. Send resolves
// the built message using the Message method.
//
// If there are multiple messages captured in the builder, and the
// Group() is set to true, then the GroupComposer's default behavior
// is used, otherwise, each message is sent individually.
func (b *Builder) Send() {
	if b.send == nil {
		b.catcher.Add(errors.New("cannot send message to unconfigured builder"))
		return
	}

	m := b.Message()
	if b.sendAsGroup {
		if len(b.opts) > 0 {
			m.SetOption(b.opts...)
		}
		b.send(m)
		return
	}

	msgs := fun.Unwind(m)
	for _, msg := range msgs {
		if len(b.opts) > 0 {
			msg.SetOption(b.opts...)
		}
		b.send(msg)
	}
}

func (b *Builder) getMessage() Composer {
	if b.composer != nil {
		if b.catcher.HasErrors() {
			return WrapError(b.catcher.Resolve(), b.composer)
		}
		if b.level != nil {
			b.composer.SetPriority(*b.level)
		}

		return b.composer
	} else {
		out := MakeError(b.catcher.Resolve())
		return out
	}
}

// Message resolves the message built by the builder, flattening (if
// needed,) multiple messages into a single grouped message, and
// wrapping the message with an error if any were produced while
// building the message.
//
// If no message is built and no errors are registered, then Message
// resolves a non-loggable error message.
//
// If multiple messages are added to the logger they are stored in a
// wrapped form, so that modifications to the message (annotations,
// levels, etc.) affect the most recent message, and then later
// converted to a group.
func (b *Builder) Message() Composer {
	msg := b.getMessage()
	if len(b.opts) > 0 {
		msg.SetOption(b.opts...)
	}
	return msg
}

// Option sets options on the builder which are applied to the
// message(s) as they are sent with Send(), or exported with
// Message().
func (b *Builder) SetOption(opts ...Option) *Builder { b.opts = append(b.opts, opts...); return b }

// Level sets the priority of the message. Call this after creating a
// message via another method, otherwise an error is generated and
// added to the builder. Additionally an error is added to the builder
// if the level is not valid.
func (b *Builder) Level(l level.Priority) *Builder { b.level = &l; return b }

// When makes the message conditional. Pass a statement to this
// function, that when false will cause the rest of the message to be
// non-loggable. This may combine well with message types that are
// expensive to calculate, or the Fields/Composer/Error producer
// methods.
func (b *Builder) When(cond bool) *Builder                      { b.composer = When(cond, b.composer); return b }
func (b *Builder) SetGroup(sendAsGroup bool) *Builder           { b.sendAsGroup = sendAsGroup; return b }
func (b *Builder) Composer(c Composer) *Builder                 { return b.set(c) }
func (b *Builder) Any(msg any) *Builder                         { return b.set(b.converter.Convert(msg)) }
func (b *Builder) F(tmpl string, a ...any) *Builder             { return b.set(MakeFormat(tmpl, a...)) }
func (b *Builder) Ln(args ...any) *Builder                      { return b.set(MakeLines(args...)) }
func (b *Builder) Error(err error) *Builder                     { return b.set(MakeError(err)) }
func (b *Builder) String(str string) *Builder                   { return b.set(MakeString(str)) }
func (b *Builder) Strings(ss []string) *Builder                 { return b.set(newLinesFromStrings(ss)) }
func (b *Builder) Bytes(in []byte) *Builder                     { return b.set(MakeBytes(in)) }
func (b *Builder) ConvertProducer(f func() any) *Builder        { return AddProducerToBuilder(b, f) }
func (b *Builder) FieldsProducer(f func() Fields) *Builder      { return b.set(MakeProducer(f)) }
func (b *Builder) ComposerProducer(f ComposerProducer) *Builder { return b.set(MakeProducer(f)) }
func (b *Builder) ErrorProducer(f ErrorProducer) *Builder       { return b.set(MakeProducer(f)) }
func (b *Builder) KVProducer(f PairProducer) *Builder           { return b.set(MakeProducer(f)) }
func (b *Builder) AnyMap(f map[string]any) *Builder             { return b.Fields(f) }
func (b *Builder) StringMap(f map[string]string) *Builder       { return b.Fields(FieldsFromMap(f)) }
func (b *Builder) Annotate(key string, val any) *Builder        { return b.Pair(key, val) }
func (b *Builder) Pair(k string, v any) *Builder                { return b.Pairs(fun.MakePair(k, v)) }
func (b *Builder) Group() *Builder                              { return b.SetGroup(true) }
func (b *Builder) Ungroup() *Builder                            { return b.SetGroup(false) }

// PairBuilder creates a new PairBuilder message and adds it to the
// builder, returning the pair builder. The original message builder
// does need will need to be sent at some point.
func (b *Builder) PairBuilder() *PairBuilder { o := BuildPair(); b.set(o); return o }

func AddProducerToBuilder[T any, F ~func() T](b *Builder, fn F) *Builder {
	return b.Composer(converterProducer(b.converter, fn))
}

// Fields, creates a new fields message if no message has been
// defined, and otherwise annotates the existing message with the
// content of the input map. This is the same semantics as StringMap
// and AnyMap methods
func (b *Builder) Fields(f Fields) *Builder {
	if b.composer == nil {
		b.composer = MakeFields(f)
		return b
	}

	for k, v := range f {
		b.composer.Annotate(k, v)
	}

	return b
}

// Pairs, creates a new key-value message if no message has been
// defined, and otherwise annotates the existing message with the
// content of the input set. This is the same semantics as the Fields
// method.
func (b *Builder) Pairs(kvs ...fun.Pair[string, any]) *Builder {
	if b.composer == nil {
		b.composer = MakeKV(kvs...)
		return b
	}

	for _, kv := range kvs {
		b.composer.Annotate(kv.Key, kv.Value)
	}
	return b
}

// Annotate adds key-value pairs to the composer. Most message types
// add this to the underlying structured data that's part of messages
// payloads, and Fields-based messages handle append these annotations
// directly to the main body of their message. Some sender/message
// formating handlers and message types may not render annotations in
// all cases.

func (b *Builder) set(msg Composer) *Builder {
	if b.composer == nil {
		b.composer = msg
		return b
	}

	b.composer = Wrap(b.composer, msg)
	return b
}
