package message

import (
	"fmt"
	"iter"
	"maps"
	"slices"
	"strings"

	"github.com/tychoish/fun/dt"
	"github.com/tychoish/fun/irt"
	"github.com/tychoish/grip/level"
)

// KV is a chainable interface for building a KV/dt.Pair
// message. These are very similar to Fields messages, however their
// keys are ordered. Satisfies the message.Composer interface.
type KV struct {
	kvs          dt.OrderedMap[string, any]
	cachedSize   int
	cachedOutput string
	hasMetadata  bool
	suppress     bool
	core         Base
}

// NewKV creates a wrapper around a composer that allows for a
// chainable pair message building interface.
func NewKV() *KV             { return &KV{} }
func makeComposer() Composer { return &KV{} }

func MakeKV[V any](seq iter.Seq2[string, V]) Composer {
	return MakeFuture(func() Composer { return NewKV().Extend(vtoany(seq)) })
}

// Composer returns the builder as a composer-type
func (p *KV) Composer() Composer                   { return p }
func (p *KV) KV(key string, value any) *KV         { p.kvs.Set(key, value); return p }
func (p *KV) WithOptions(f ...Option) *KV          { p.core.SetOption(f...); return p }
func (p *KV) Level(l level.Priority) *KV           { p.SetPriority(l); return p }
func (p *KV) Extend(in iter.Seq2[string, any]) *KV { p.kvs.Extend(in); return p }
func (p *KV) Fields(f Fields) *KV                  { return p.Extend(maps.All(f)) }
func (p *KV) KVs(e ...irt.KV[string, any]) *KV     { return p.Extend(irt.KVsplit(irt.Slice(e))) }
func (p *KV) WithError(err error) *KV              { return p.WhenKV(err != nil, "error", err) }
func (p *KV) When(cond bool) *KV                   { p.suppress = !cond; return p }
func (p *KV) WhenKV(cond bool, k string, v any) *KV {
	if cond {
		p.kvs.Set(k, v)
	}
	return p
}

func vtoany[V any](seq iter.Seq2[string, V]) iter.Seq2[string, any] {
	return irt.Convert2(seq, func(key string, value V) (string, any) { return key, value })
}

func (p *KV) Annotate(key string, value any) { p.kvs.Set(key, value) }
func (p *KV) Loggable() bool                 { return !p.suppress && p.kvs.Len() > 0 }
func (p *KV) SetOption(opts ...Option)       { p.core.SetOption(opts...) }
func (p *KV) Priority() level.Priority       { return p.core.Priority() }
func (p *KV) SetPriority(l level.Priority)   { p.core.SetPriority(l) }
func (p *KV) Structured() bool               { return true }
func (p *KV) Raw() any {
	p.core.Collect()

	if p.core.IncludeMetadata && !p.hasMetadata {
		p.kvs.Set("meta", &p.core)
		p.hasMetadata = true
	}

	return &p.kvs
}

func (p *KV) String() string {
	if p.kvs.Len() == p.cachedSize && (p.cachedOutput != "" || !p.core.IncludeMetadata) {
		return p.cachedOutput
	}

	p.core.Collect()

	if p.core.IncludeMetadata && !p.hasMetadata {
		p.kvs.Set("meta", &p.core)
		p.hasMetadata = true
	}

	out := makeSimpleFieldsString(p.kvs.Iterator(), p.core.IncludeMetadata, p.kvs.Len())
	if p.core.SortComponents {
		slices.Sort(out)
	}
	p.cachedOutput = strings.Join(out, " ")
	p.cachedSize = p.kvs.Len()

	return p.cachedOutput
}

var skippedFields = map[string]struct{}{"meta": {}}

func renderField(k string, v any) string {
	if _, ok := skippedFields[k]; ok {
		return ""
	}
	switch val := v.(type) {
	case fmt.Stringer, string:
		return fmt.Sprintf("%s='%s'", k, val)
	default:
		return fmt.Sprintf("%s='%v'", k, v)
	}
}

func makeSimpleFieldsString(f iter.Seq2[string, any], doSkips bool, hint int) []string {
	return irt.Collect(irt.RemoveZeros(irt.Merge(f, renderField)))
}
