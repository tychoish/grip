package message

import (
	"context"
	"fmt"
	"strings"

	"github.com/tychoish/fun"
	"github.com/tychoish/grip/level"
)

// PairBuilder is a chainable interface for building a KV/fun.Pair
// message. These are very similar to Fields messages, however their
// keys are ordered, duplicate keys can be defined, and
type PairBuilder struct {
	kvs          fun.Pairs[string, any]
	cachedSize   int
	cachedOutput string
	hasMetadata  bool
	Base
}

// BuildPair creates a wrapper around a composer that allows for a
// chainable pair message building interface.
func BuildPair() *PairBuilder { return &PairBuilder{} }

// Composer returns the builder as a composer-type
func (p *PairBuilder) Composer() Composer                            { return p }
func (p *PairBuilder) Pair(key string, value any) *PairBuilder       { p.kvs.Add(key, value); return p }
func (p *PairBuilder) AddPair(in fun.Pair[string, any]) *PairBuilder { p.kvs.AddPair(in); return p }
func (p *PairBuilder) Option(f Option) *PairBuilder                  { p.SetOption(f); return p }
func (p *PairBuilder) Level(l level.Priority) *PairBuilder           { p.SetPriority(l); return p }
func (p *PairBuilder) Fields(f Fields) *PairBuilder                  { p.kvs.ConsumeMap(f); return p }

func (p *PairBuilder) extender(in fun.Pairs[string, any])              { p.kvs = p.kvs.Append(in...) }
func (p *PairBuilder) Extend(in fun.Pairs[string, any]) *PairBuilder   { p.extender(in); return p }
func (p *PairBuilder) Append(in ...fun.Pair[string, any]) *PairBuilder { return p.Extend(in) }

func (p *PairBuilder) Iterator(ctx context.Context, iter fun.Iterator[fun.Pair[string, any]]) *PairBuilder {
	p.kvs.Consume(ctx, iter)
	return p
}

// MakeKV constructs a new Composer using KV (fun.Pair[string, any]).
func MakeKV(kvs ...fun.Pair[string, any]) Composer { return BuildPair().Append(kvs...) }
func KV(k string, v any) fun.Pair[string, any]     { return fun.MakePair(k, v) }

func (p *PairBuilder) Annotate(key string, value any) {
	p.cachedOutput = ""
	p.kvs = append(p.kvs, fun.MakePair(key, value))
}

func (p *PairBuilder) Loggable() bool   { return len(p.kvs) > 0 }
func (p *PairBuilder) Structured() bool { return true }
func (p *PairBuilder) Raw() any {
	p.Collect()

	if p.IncludeMetadata && !p.hasMetadata {
		p.kvs = append(p.kvs, fun.MakePair[string, any]("meta", &p.Base))
		p.hasMetadata = true
	}

	return p.kvs
}
func (p *PairBuilder) String() string {
	if p.cachedOutput != "" && len(p.kvs) != p.cachedSize {
		return p.cachedOutput
	}

	p.Collect()

	if p.IncludeMetadata && !p.hasMetadata {
		p.kvs = append(p.kvs, fun.MakePair[string, any]("meta", &p.Base))
		p.hasMetadata = true
	}

	out := make([]string, len(p.kvs))
	var seenMetadata bool
	for idx, kv := range p.kvs {
		if kv.Key == "meta" && (seenMetadata || !p.IncludeMetadata) {
			seenMetadata = true
			continue
		}

		switch val := kv.Value.(type) {
		case string, fmt.Stringer:
			out[idx] = fmt.Sprintf("%s='%s'", kv.Key, val)
		default:
			out[idx] = fmt.Sprintf("%s='%v'", kv.Key, kv.Value)
		}

		if kv.Key == "meta" {
			p.hasMetadata = true
			seenMetadata = true
		}
	}

	p.cachedOutput = strings.Join(out, " ")
	p.cachedSize = len(p.kvs)

	return p.cachedOutput
}
