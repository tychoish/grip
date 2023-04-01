// Functional Messages
//
// Grip can automatically convert three types of functions into
// messages:
//
//	func() Fields
//	func() Composer
//	func() error
//
// The benefit of these functions is that they're only called if
// the message is above the logging threshold. In the case of
// conditional logging (i.e. When), if the conditional is false, then
// the function is never called.
//
// in the case of all the buffered sending implementation, the
// function call can be deferred and run outside of the main thread,
// and so may be an easy way to defer message production outside in
// cases where messages may be complicated.
//
// Additionally, the message conversion in grip's logging method can
// take these function types and convert them to these messages, which
// can clean up some call-site operations, and makes it possible to
// use defer with io.Closer methods without wrapping the method in an
// additional function, as in:
//
//	defer grip.Error(file.Close)
//
// Although the WrapErrorFunc method, as in the following may permit
// useful annotation, as follows, which has the same "lazy" semantics.
//
//	defer grip.Error(message.WrapErrorFunc(file.Close, message.Fields{}))
package message

import (
	"sync"

	"github.com/tychoish/grip/level"
)

// KVProducer allows callers to delay generation of KV lists (as
// structured log payloads) until the log message needs to be sent
// (e.g. the .String() or .Raw() methods are called on the Composer
// interface.) While all implementations of composer provide this
// ability to do lazy evaluation of log messages, you can use this and
// other producer types to implement logging as functions rather than
// as implementations the Composer interface itself.
type KVProducer func() KVs

// FieldsProducer is a function that returns a structured message body
// as a way of writing simple Composer implementations in the form
// anonymous functions, as in:
//
//	grip.Info(func() message.Fields {return message.Fields{"message": "hello world!"}})
//
// Grip can automatically convert these functions when passed to a
// logging function.
//
// If the Fields object is nil or empty then no message is logged.
type FieldsProducer func() Fields

// ComposerProducer constructs a lazy composer, and makes it easy to
// implement new Composers as functions returning an existing composer
// type. Consider the following:
//
//	grip.Info(func() message.Composer { return WrapError(validateRequest(req), message.Fields{"op": "name"})})
//
// Grip can automatically convert these functions when passed to a
// logging function.
//
// If the Fields object is nil or empty then no message is ever logged.
type ComposerProducer func() Composer

// ErrorProducer is a function that returns an error, and is used for
// constructing message that lazily wraps the resulting function which
// is called when the message is dispatched.
//
// If you pass one of these functions to a logging method, the
// ConvertToComposer operation will construct a lazy Composer based on
// this function, as in:
//
//	grip.Error(func() error { return errors.New("error message") })
//
// It may be useful also to pass a "closer" function in this form, as
// in:
//
//	grip.Error(file.Close)
//
// As a special case the WrapErrorFunc method has the same semantics
// as other ErrorProducer methods, but makes it possible to annotate
// an error.
type ErrorProducer func() error

// MakeProduer constructs a lazy Producer message composer.
//
// Producer functions are only called before calling the Loggable,
// String, Raw, or Annotate methods. Changing the priority does not
// call the function. In practice, if the priority of the message is
// below the logging threshold, then the function will never be
// called.
func MakeProducer[T any, F ~func() T](fp F) Composer {
	if fp == nil {
		return MakeKV()
	}
	return &composerProducerMessage{cp: func() Composer { return Convert(fp()) }}
}

////////////////////////////////////////////////////////////////////////

type composerProducerMessage struct {
	cp     ComposerProducer
	cached Composer
	level  level.Priority
	exec   sync.Once
}

func (cp *composerProducerMessage) resolve() {
	cp.exec.Do(func() {
		if cp.cp == nil {
			cp.cp = func() Composer { return MakeKV() }
		}

		cp.cached = cp.cp()
		cp.cached.SetPriority(cp.level)
	})
}

func (cp *composerProducerMessage) Annotate(k string, v any) error {
	cp.resolve()
	return cp.cached.Annotate(k, v)
}

func (cp *composerProducerMessage) SetPriority(p level.Priority) {
	cp.level = p
	if cp.cached != nil {
		cp.cached.SetPriority(cp.level)
	}
}

func (cp *composerProducerMessage) Loggable() bool {
	if cp.cp == nil {
		return false
	}

	cp.resolve()
	return cp.cached.Loggable()
}

func (cp *composerProducerMessage) Priority() level.Priority { return cp.level }
func (cp *composerProducerMessage) Structured() bool         { cp.resolve(); return cp.cached.Structured() }
func (cp *composerProducerMessage) String() string           { cp.resolve(); return cp.cached.String() }
func (cp *composerProducerMessage) Raw() any                 { cp.resolve(); return cp.cached.Raw() }
