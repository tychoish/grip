package message

import (
	"fmt"
	"iter"
	"maps"
	"strings"

	"github.com/tychoish/fun/dt"
	"github.com/tychoish/fun/irt"
	"github.com/tychoish/grip/level"
)

// BuilderKV is a chainable interface for building a KV/dt.Pair
// message. These are very similar to Fields messages, however their
// keys are ordered. Satisfies the message.Composer interface.
type BuilderKV struct {
	kvs          dt.OrderedMap[string, any]
	cachedSize   int
	cachedOutput string
	hasMetadata  bool
	Base
}

// BuildKV creates a wrapper around a composer that allows for a
// chainable pair message building interface.
func BuildKV() *BuilderKV    { return &BuilderKV{} }
func makeComposer() Composer { return &BuilderKV{} }

// Composer returns the builder as a composer-type
func (p *BuilderKV) Composer() Composer                          { return p }
func (p *BuilderKV) KV(key string, value any) *BuilderKV         { p.kvs.Set(key, value); return p }
func (p *BuilderKV) Option(f Option) *BuilderKV                  { p.SetOption(f); return p }
func (p *BuilderKV) Level(l level.Priority) *BuilderKV           { p.SetPriority(l); return p }
func (p *BuilderKV) Fields(f Fields) *BuilderKV                  { p.kvs.Extend(maps.All(f)); return p }
func (p *BuilderKV) Extend(in iter.Seq2[string, any]) *BuilderKV { p.kvs.Extend(in); return p }

func (p *BuilderKV) WhenKV(cond bool, k string, v any) *BuilderKV {
	if cond {
		p.kvs.Set(k, v)
	}
	return p
}

func vtoany[V any](seq iter.Seq2[string, V]) iter.Seq2[string, any] {
	return irt.Convert2(seq, func(key string, value V) (string, any) { return key, value })
}

// MakeKV constructs a new Composer from a dt.OrderedMap).
func MakeKV[V any](seq iter.Seq2[string, V]) Composer { return BuildKV().Extend(vtoany(seq)) }
func KV(k string, v any) iter.Seq2[string, any]       { return irt.Two(k, v) }
func (p *BuilderKV) Annotate(key string, value any)   { p.kvs.Set(key, value) }
func (p *BuilderKV) Loggable() bool                   { return p.kvs.Len() > 0 }
func (p *BuilderKV) Structured() bool                 { return true }
func (p *BuilderKV) Raw() any {
	p.Collect()

	if p.IncludeMetadata && !p.hasMetadata {
		p.kvs.Set("meta", &p.Base)
		p.hasMetadata = true
	}

	return &p.kvs
}

func (p *BuilderKV) String() string {
	if p.cachedOutput != "" && p.kvs.Len() == p.cachedSize {
		return p.cachedOutput
	}

	p.Collect()

	if p.IncludeMetadata && !p.hasMetadata {
		p.kvs.Set("meta", &p.Base)
		p.hasMetadata = true
	}

	out := make([]string, 0, p.kvs.Len())
	var seenMetadata bool

	for key, value := range p.kvs.Iterator() {
		if key == "meta" && (seenMetadata || !p.IncludeMetadata) {
			seenMetadata = true
			continue
		}

		switch val := value.(type) {
		case string, fmt.Stringer:
			out = append(out, fmt.Sprintf("%s='%s'", key, val))
		default:
			out = append(out, fmt.Sprintf("%s='%v'", key, value))
		}

		if key == "meta" {
			p.hasMetadata = true
		}
	}
	p.cachedOutput = strings.Join(out, " ")
	p.cachedSize = p.kvs.Len()

	return p.cachedOutput
}
