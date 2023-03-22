package message

import (
	"github.com/tychoish/fun"
	"github.com/tychoish/grip/level"
)

type conditional struct {
	cond bool
	msg  Composer
}

// When returns a conditional message that is only logged if the
// condition is bool. Converts the second argument to a composer, if
// needed, using the same rules that the logging methods use.
func When(cond bool, m any) Composer { return &conditional{cond: cond, msg: Convert(m)} }

// Whenf returns a conditional message that is only logged if the
// condition is bool, and creates a sprintf-style message, which will
// itself only log if the base expression is not the empty string.
func Whenf(cond bool, m string, args ...any) Composer {
	return &conditional{cond: cond, msg: MakeFormat(m, args...)}
}

// Whenln returns a conditional message that is only logged if the
// condition is bool, and creates a sprintf-style message, which will
// itself only log if the base expression is not the empty string.
func Whenln(cond bool, args ...any) Composer {
	return &conditional{cond: cond, msg: MakeLines(args...)}
}

// WhenMsg returns a conditional message that is only logged if the
// condition is bool, and creates a string message that will only log
// when the message content is not the empty string. Use this for a
// more strongly-typed conditional logging message.
func WhenMsg(cond bool, m string) Composer { return &conditional{cond: cond, msg: MakeString(m)} }

func (c *conditional) String() string {
	return safeDo(c.msg, func(c Composer) string { return c.String() })
}

func (c *conditional) Raw() any {
	return safeDo(c.msg, func(c Composer) any { return c.Raw() })
}

func (c *conditional) Annotate(k string, v any) error {
	return safeDo(c.msg, func(c Composer) error { return c.Annotate(k, v) })
}

func (c *conditional) Structured() bool {
	return safeDo(c.msg, func(c Composer) bool { return c.Structured() })
}

func (c *conditional) Loggable() bool {
	return c.cond && safeDo(c.msg, func(c Composer) bool { return c.Loggable() })
}

func (c *conditional) Priority() level.Priority {
	return safeDo(c.msg, func(c Composer) level.Priority { return c.Priority() })
}

func (c *conditional) SetPriority(p level.Priority) {
	safeOp(c.msg, func(c Composer) { c.SetPriority(p) })
}

func safeOp(c Composer, fn func(Composer)) {
	if c != nil {
		fn(c)
	}
}

func safeDo[O any](c Composer, fn func(Composer) O) O {
	if c != nil {
		return fn(c)
	}
	return fun.ZeroOf[O]()
}
