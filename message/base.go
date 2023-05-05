package message

import (
	"os"
	"time"

	"github.com/tychoish/grip/level"
)

// Base provides a simple embedable implementation of some common
// aspects of a message.Composer. Additionally the Collect() method
// collects some simple metadata, that may be useful for some more
// structured logging applications.
type Base struct {
	Level    level.Priority `bson:"level,omitempty" json:"level,omitempty" yaml:"level,omitempty"`
	Hostname string         `bson:"hostname,omitempty" json:"hostname,omitempty" yaml:"hostname,omitempty"`
	Time     time.Time      `bson:"time,omitempty" json:"time,omitempty" yaml:"time,omitempty"`
	Process  string         `bson:"process,omitempty" json:"process,omitempty" yaml:"process,omitempty"`
	Pid      int            `bson:"pid,omitempty" json:"pid,omitempty" yaml:"pid,omitempty"`
	Context  Fields         `bson:"context,omitempty" json:"context,omitempty" yaml:"context,omitempty"`
}

// IsZero returns true when Base is nil or it is non-nil and none of
// its fields are set.
func (b *Base) IsZero() bool {
	return b == nil || b.Level == level.Invalid && b.Hostname == "" && b.Time.IsZero() && b.Process == "" && b.Pid == 0 && b.Context == nil
}

// Collect records the time, process name, and hostname. Useful in the
// context of a Raw() method.
func (b *Base) Collect() {
	if b.Pid > 0 {
		return
	}

	b.Hostname, _ = os.Hostname()

	b.Time = time.Now()
	b.Process = os.Args[0]
	b.Pid = os.Getpid()
}

// Priority returns the configured priority of the message.
func (b *Base) Priority() level.Priority {
	return b.Level
}

// Structured returns true if there are any annotations. Otherwise
// false. Most Composer implementations should override.
func (b *Base) Structured() bool { return len(b.Context) >= 1 }

// SetPriority allows you to configure the priority of the
// message. Returns an error if the priority is not valid.
func (b *Base) SetPriority(l level.Priority) { b.Level = l }

// Annotate makes it possible for callers and senders to add
// structured data to a message. This may be overridden for some
// implementations.
func (b *Base) Annotate(key string, value any) {
	if b.Context == nil {
		b.Context = Fields{}
	}

	b.Context[key] = value

	return
}
