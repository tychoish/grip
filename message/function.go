// Message Futures and Functional Logging
//
// Grip can automatically converts functions which produce Composers
// or types that can be trivially converted to messages..
//
// The benefit of this logging model, is that the message is generated
// only when the message is above the logging threshold. In the case
// of conditional logging (i.e. When), if the conditional is false,
// then the function is never called. Similarly, if the message is a
// Debug, then the function is never logged.
//
// For sending architectures where, there's a buffer between the
// logger and the message being sent or persisted, the function call
// that resolves the message can be deferred.
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

	"github.com/tychoish/fun"
	"github.com/tychoish/grip/level"
)

// Marshaler allows arbitrary types to control how they're converted
// (in the default converter) to a message.Composer, without requiring
// that the arbitrary type itself implement Composer, for a related
// level of functionality.
type Marshaler interface {
	MarshalComposer() Composer
}

// MakeFuture constructs a compuser build around a fun.Future[T] which
// a function object that will resolve a message lazily. Use this to
// construct a function that will produce a message or a type that
// can be trivally converted to a message at call time.
//
// The future is resolved functions are only called when the outer
// composers Loggable, String, Raw, or Annotate methods are
// called. Changing the priority does not resolve the future. In
// practice, if the priority of the message is below the logging
// threshold, then the function will never be called.
func MakeFuture[T any](fp fun.Future[T]) Composer { return converterFuture(defaultConverter{}, fp) }

func converterFuture[T any](c Converter, fp fun.Future[T]) Composer {
	if fp == nil {
		fp = func() (o T) { return }
	}
	return &composerFutureMessage{cp: func() Composer { return c.Convert(fp()) }}
}

////////////////////////////////////////////////////////////////////////

type composerFutureMessage struct {
	cp      fun.Future[Composer]
	cached  Composer
	level   level.Priority
	exec    sync.Once
	lazyOps []fun.Observer[Composer]
}

func (cp *composerFutureMessage) resolve() {
	cp.exec.Do(func() {
		if cp.cp == nil {
			cp.cp = func() Composer { return MakeKV() }
		}

		cp.cached = cp.cp()
		for _, op := range cp.lazyOps {
			op(cp.cached)
		}
		cp.lazyOps = nil
	})
}

func (cp *composerFutureMessage) SetPriority(p level.Priority) {
	cp.level = p
	if cp.cached != nil {
		cp.cached.SetPriority(p)
	} else {
		cp.lazyOps = append(cp.lazyOps, func(c Composer) { c.SetPriority(cp.level) })
	}
}

func (cp *composerFutureMessage) Loggable() bool {
	if cp.cp == nil {
		return false
	}

	cp.resolve()
	return cp.cached.Loggable()
}

func (cp *composerFutureMessage) Priority() level.Priority { return cp.level }
func (cp *composerFutureMessage) Structured() bool         { cp.resolve(); return cp.cached.Structured() }
func (cp *composerFutureMessage) String() string           { cp.resolve(); return cp.cached.String() }
func (cp *composerFutureMessage) Raw() any                 { cp.resolve(); return cp.cached.Raw() }

func (cp *composerFutureMessage) Annotate(k string, v any) {
	if cp.cached != nil {
		cp.cached.Annotate(k, v)
	} else {
		cp.lazyOps = append(cp.lazyOps, func(c Composer) { c.Annotate(k, v) })
	}
}

func (cp *composerFutureMessage) SetOption(opts ...Option) {
	if cp.cached != nil {
		cp.cached.SetOption(opts...)
	} else {
		cp.lazyOps = append(cp.lazyOps, func(c Composer) { c.SetOption(opts...) })
	}

}
