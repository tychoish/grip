package message

import (
	"os"
	"path/filepath"
	"time"

	"github.com/tychoish/fun/adt"
	"github.com/tychoish/grip/level"
)

var (
	hostnameCache *adt.Once[string]
	pidCache      *adt.Once[int]
	procCache     *adt.Once[string]
)

func init() {
	hostnameCache = &adt.Once[string]{}
	pidCache = &adt.Once[int]{}
	procCache = &adt.Once[string]{}
}

// Base provides a simple embedable implementation of some common
// aspects of a message.Composer. Additionally the Collect() method
// collects some simple metadata, that may be useful for some more
// structured logging applications.
type Base struct {
	Level            level.Priority `bson:"level,omitempty" json:"level,omitempty" yaml:"level,omitempty"`
	Pid              int            `bson:"pid,omitempty" json:"pid,omitempty" yaml:"pid,omitempty"`
	Process          string         `bson:"proc,omitempty" json:"proc,omitempty" yaml:"proc,omitempty"`
	Host             string         `bson:"host,omitempty" json:"host,omitempty" yaml:"host,omitempty"`
	Time             time.Time      `bson:"ts,omitempty" json:"ts,omitempty" yaml:"ts,omitempty"`
	Context          Fields         `bson:"data,omitempty" json:"data,omitempty" yaml:"data,omitempty"`
	SkipCollection   bool           `bson:"-" json:"-" yaml:"-"`
	SkipMetadata     bool           `bson:"-" json:"-" yaml:"-"`
	MessageIsSpecial bool           `bson:"-" json:"-" yaml:"-"`
}

func (b *Base) SetOption(opts ...Option) {
	for _, opt := range opts {
		switch opt {
		case OptionSkipAllMetadata:
			b.SkipMetadata = true
		case OptionSkipCollect:
			b.SkipCollection = true
		case OptionDoBaseCollect:
			b.SkipCollection = false
		case OptionIncludeAllMetadata:
			b.SkipMetadata = false
		case OptionMessageIsNotStructuredField:
			b.MessageIsSpecial = true
		}
	}
}

// IsZero returns true when Base is nil or it is non-nil and none of
// its fields are set.
func (b *Base) IsZero() bool {
	return b == nil || b.Level == level.Invalid && b.Host == "" && b.Time.IsZero() && b.Process == "" && b.Pid == 0 && b.Context == nil
}

// Collect records the time, process name, and hostname. Useful in the
// context of a Raw() method.
func (b *Base) Collect() {
	if b.Pid > 0 || b.SkipCollection {
		return
	}

	b.Host = hostnameCache.Do(func() string { out, _ := os.Hostname(); return out })
	b.Process = procCache.Do(func() string { return filepath.Base(os.Args[0]) })
	b.Pid = pidCache.Do(func() int { return os.Getpid() })
	b.Time = time.Now()
}

// Priority returns the configured priority of the message.
func (b *Base) Priority() level.Priority { return b.Level }

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
