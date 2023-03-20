package message

import (
	"github.com/tychoish/grip/level"
)

type condComposer struct {
	cond bool
	msg  Composer
}

// When returns a conditional message that is only logged if the
// condition is bool. Converts the second argument to a composer, if
// needed, using the same rules that the logging methods use.
func When(cond bool, m any) Composer { return &condComposer{cond: cond, msg: Convert(m)} }

// Whenf returns a conditional message that is only logged if the
// condition is bool, and creates a sprintf-style message, which will
// itself only log if the base expression is not the empty string.
func Whenf(cond bool, m string, args ...any) Composer {
	return &condComposer{cond: cond, msg: MakeFormat(m, args...)}
}

// Whenln returns a conditional message that is only logged if the
// condition is bool, and creates a sprintf-style message, which will
// itself only log if the base expression is not the empty string.
func Whenln(cond bool, args ...any) Composer {
	return &condComposer{cond: cond, msg: MakeLines(args...)}
}

// WhenMsg returns a conditional message that is only logged if the
// condition is bool, and creates a string message that will only log
// when the message content is not the empty string. Use this for a
// more strongly-typed conditional logging message.
func WhenMsg(cond bool, m string) Composer { return &condComposer{cond: cond, msg: MakeString(m)} }

func (c *condComposer) String() string                     { return c.msg.String() }
func (c *condComposer) Raw() any                           { return c.msg.Raw() }
func (c *condComposer) Priority() level.Priority           { return c.msg.Priority() }
func (c *condComposer) SetPriority(p level.Priority) error { return c.msg.SetPriority(p) }
func (c *condComposer) Annotate(k string, v any) error     { return c.msg.Annotate(k, v) }
func (c *condComposer) Structured() bool                   { return c.msg.Structured() }
func (c *condComposer) Loggable() bool                     { return c.cond && c.msg.Loggable() }
