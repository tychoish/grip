package message

import (
	"github.com/tychoish/grip/level"
)

type conditional struct {
	cond        bool
	resolved    Composer
	constructor func() Composer
	lazyOpts    []func(c Composer)
}

// When returns a conditional message that is only logged if the
// condition is bool. Converts the second argument to a composer, if
// needed, using the same rules that the logging methods use.
func When(cond bool, m any) Composer {
	return &conditional{cond: cond, constructor: func() Composer { return Convert(m) }}
}

// Whenf returns a conditional message that is only logged if the
// condition is bool, and creates a sprintf-style message, which will
// itself only log if the base expression is not the empty string.
func Whenf(cond bool, m string, args ...any) Composer {
	return &conditional{cond: cond, constructor: func() Composer { return MakeFormat(m, args...) }}
}

// Whenln returns a conditional message that is only logged if the
// condition is bool, and creates a sprintf-style message, which will
// itself only log if the base expression is not the empty string.
func Whenln(cond bool, args ...any) Composer {
	return &conditional{cond: cond, constructor: func() Composer { return MakeLines(args...) }}
}

// WhenMsg returns a conditional message that is only logged if the
// condition is bool, and creates a string message that will only log
// when the message content is not the empty string. Use this for a
// more strongly-typed conditional logging message.
func WhenMsg(cond bool, m string) Composer {
	return &conditional{cond: cond, constructor: func() Composer { return MakeString(m) }}
}

func (c *conditional) resolve() Composer {
	switch {
	case c.constructor != nil:
		c.resolved = c.constructor()
		c.constructor = nil
	case c.resolved != nil:
		return c.resolved
	}

	for _, op := range c.lazyOpts {
		op(c.resolved)
	}
	c.lazyOpts = nil

	return c.resolved
}

func (c *conditional) String() string {
	return safeDo(c.resolve(), func(c Composer) string { return c.String() })
}

func (c *conditional) Raw() any {
	return safeDo(c.resolve(), func(c Composer) any { return c.Raw() })
}

func (c *conditional) Annotate(k string, v any) {
	if c.resolved == nil {
		c.lazyOpts = append(c.lazyOpts, func(c Composer) { c.Annotate(k, v) })
	} else {
		c.resolved.Annotate(k, v)
	}
}

func (c *conditional) Structured() bool {
	return safeDo(c.resolve(), func(c Composer) bool { return c.Structured() })
}

func (c *conditional) Loggable() bool {
	return safeDo(c.resolve(), func(c Composer) bool { return c.Loggable() })
}

func (c *conditional) Priority() level.Priority {
	return safeDo(c.resolve(), func(c Composer) level.Priority { return c.Priority() })
}

func (c *conditional) SetPriority(p level.Priority) {
	if c.resolved == nil {
		c.lazyOpts = append(c.lazyOpts, func(c Composer) { c.SetPriority(p) })
	} else {
		c.resolved.SetPriority(p)
	}
}

func (c *conditional) SetOption(opts ...Option) {
	c.lazyOpts = append(c.lazyOpts, func(cp Composer) { cp.SetOption(opts...) })
}

func safeDo[O any](c Composer, fn func(Composer) O) O {
	if c != nil {
		return fn(c)
	}

	var out O
	return out
}
