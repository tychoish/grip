package message

import (
	"iter"

	"github.com/tychoish/fun/erc"
	"github.com/tychoish/fun/fn"
	"github.com/tychoish/fun/irt"
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
	level       fn.Future[level.Priority]
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
		b.catcher.New("cannot send message to unconfigured builder")
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

	msgs := Unwind(m)
	for _, msg := range msgs {
		if len(b.opts) > 0 {
			msg.SetOption(b.opts...)
		}
		b.send(msg)
	}
}

func (b *Builder) getMessage() Composer {
	if b.composer != nil {
		if !b.catcher.Ok() {
			return WrapError(b.catcher.Resolve(), b.composer)
		}
		if b.level != nil {
			b.composer.SetPriority(b.level())
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
func (b *Builder) WithOptions(opts ...Option) *Builder { b.opts = append(b.opts, opts...); return b }
func (b *Builder) init() *Builder                      { b.setDefault(makeComposer); return b }

// Level sets the priority of the message. Call this after creating a
// message via another method, otherwise an error is generated and
// added to the builder. Additionally an error is added to the builder
// if the level is not valid.
func (b *Builder) Level(l level.Priority) *Builder               { b.level = fn.AsFuture(l); return b }
func (b *Builder) Leveler(fp fn.Future[level.Priority]) *Builder { b.level = fp.Once(); return b }
func (b *Builder) Loggable() bool                                { return b.composer != nil && b.composer.Loggable() }
func (b *Builder) Extend(seq iter.Seq2[string, any]) *Builder    { return b.init().iter(seq) }
func (b *Builder) Annotate(k string, v any)                      { b.init().push(k, v) }
func (b *Builder) SetPriority(l level.Priority)                  { b.Level(l) }
func (b *Builder) String() string                                { return b.init().composer.String() }
func (b *Builder) Structured() bool                              { return b.init().composer.Structured() }
func (b *Builder) Priority() level.Priority                      { return b.init().composer.Priority() }
func (b *Builder) Raw() any                                      { return b.init().composer.Raw() }
func (b *Builder) SetOption(opts ...Option)                      { b.WithOptions(opts...) }
func (b *Builder) with(k string, v any) *Builder                 { b.push(k, v); return b }
func (b *Builder) push(k string, v any)                          { b.composer.Annotate(k, v) }
func (b *Builder) iter(s iter.Seq2[string, any]) *Builder        { irt.Apply2(s, b.push); return b }
func (b *Builder) set(msg Composer) *Builder                     { b.wrap(msg); return b }
func (b *Builder) wrap(msg Composer)                             { b.composer = Wrap(b.composer, msg) }

// When makes the message conditional. Pass a statement to this
// function, that when false will cause the rest of the message to be
// non-loggable. This may combine well with message types that are
// expensive to calculate, or the Fields/Composer/Error producer
// methods.
func (b *Builder) When(cond bool) *Builder                { b.composer = When(cond, b.composer); return b }
func (b *Builder) SetGroup(sendAsGroup bool) *Builder     { b.sendAsGroup = sendAsGroup; return b }
func (b *Builder) Group() *Builder                        { return b.SetGroup(true) }
func (b *Builder) Ungroup() *Builder                      { return b.SetGroup(false) }
func (b *Builder) Composer(c Composer) *Builder           { return b.set(c) }
func (b *Builder) Any(msg any) *Builder                   { return b.set(b.converter.Convert(msg)) }
func (b *Builder) F(tmpl string, a ...any) *Builder       { return b.set(MakeFormat(tmpl, a...)) }
func (b *Builder) Ln(str string) *Builder                 { return b.set(MakeString(str)) }
func (b *Builder) Lns(args ...any) *Builder               { return b.set(MakeLines(args...)) }
func (b *Builder) Strings(ss []string) *Builder           { return b.set(newLinesFromStrings(ss)) }
func (b *Builder) Error(err error) *Builder               { return b.set(MakeError(err)) }
func (b *Builder) Bytes(in []byte) *Builder               { return b.set(MakeBytes(in)) }
func (b *Builder) AnyMap(f map[string]any) *Builder       { return b.Fields(f) }
func (b *Builder) StringMap(f map[string]string) *Builder { return b.Fields(FieldsFromMap(f)) }
func (b *Builder) KV(k string, v any) *Builder            { return b.init().with(k, v) }
func (b *Builder) Future() *BuilderFuture                 { return &BuilderFuture{uilder: b} }

func (b *BuilderFuture) Send()                                    { b.uilder.Send() }
func (b *BuilderFuture) Convert(f fn.Future[any]) *Builder        { return WithFuture(b.uilder, f) }
func (b *BuilderFuture) Fields(f fn.Future[Fields]) *Builder      { return WithFuture(b.uilder, f) }
func (b *BuilderFuture) Map(f fn.Future[map[string]any]) *Builder { return WithFuture(b.uilder, f) }
func (b *BuilderFuture) Composer(f fn.Future[Composer]) *Builder  { return WithFuture(b.uilder, f) }
func (b *BuilderFuture) Error(f fn.Future[error]) *Builder        { return WithFuture(b.uilder, f) }
func (b *BuilderFuture) String(f fn.Future[string]) *Builder      { return WithFuture(b.uilder, f) }
func (b *BuilderFuture) KV(f fn.Future[iter.Seq2[string, any]]) *Builder {
	return WithFuture(b.uilder, f)
}

type BuilderFuture struct{ uilder *Builder }

func WithFuture[T any](b *Builder, fp fn.Future[T]) *Builder {
	return b.Composer(converterFuture(b.converter, fp))
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

	irt.Apply2(irt.Map(f), b.composer.Annotate)

	return b
}

func (b *Builder) setDefault(f fn.Future[Composer]) {
	if b.composer == nil {
		b.composer = f()
	}
}
