package message

import (
	"context"
	"fmt"

	"strings"

	"github.com/tychoish/fun"
	"github.com/tychoish/fun/dt"
	"github.com/tychoish/fun/ft"
	"github.com/tychoish/grip/level"
)

// PairBuilder is a chainable interface for building a KV/dt.Pair
// message. These are very similar to Fields messages, however their
// keys are ordered, duplicate keys can be defined, and
type PairBuilder struct {
	kvs          dt.Pairs[string, any]
	cachedSize   int
	cachedOutput string
	hasMetadata  bool
	Base
}

// BuildPair creates a wrapper around a composer that allows for a
// chainable pair message building interface.
func BuildPair() *PairBuilder { return &PairBuilder{} }

// Composer returns the builder as a composer-type
func (p *PairBuilder) Composer() Composer                           { return p }
func (p *PairBuilder) Pair(key string, value any) *PairBuilder      { p.kvs.Add(key, value); return p }
func (p *PairBuilder) AddPair(in dt.Pair[string, any]) *PairBuilder { p.kvs.AddPair(in); return p }
func (p *PairBuilder) Option(f Option) *PairBuilder                 { p.SetOption(f); return p }
func (p *PairBuilder) Level(l level.Priority) *PairBuilder          { p.SetPriority(l); return p }
func (p *PairBuilder) Fields(f Fields) *PairBuilder                 { p.kvs.ConsumeMap(f); return p }

func (p *PairBuilder) Extend(in *dt.Pairs[string, any]) *PairBuilder  { p.kvs.Extend(in); return p }
func (p *PairBuilder) Append(in ...dt.Pair[string, any]) *PairBuilder { p.kvs.Append(in...); return p }
func (p *PairBuilder) PairWhen(cond bool, k string, v any) *PairBuilder {
	return ft.WhenDo(cond, func() *PairBuilder { return p.Pair(k, v) })
}

func (p *PairBuilder) Iterator(ctx context.Context, iter *fun.Iterator[dt.Pair[string, any]]) *PairBuilder {
	// TODO this is probably safe, but soon we can add this error
	// to the input iterator after fun v0.10
	_ = p.kvs.Consume(ctx, iter)
	return p
}

// MakeKV constructs a new Composer using KV (dt.Pair[string, any]).
func MakeKV(kvs ...dt.Pair[string, any]) Composer     { return BuildPair().Append(kvs...) }
func MakePairs(kvs *dt.Pairs[string, any]) Composer   { return &PairBuilder{kvs: *kvs} }
func KV(k string, v any) dt.Pair[string, any]         { return dt.MakePair(k, v) }
func (p *PairBuilder) Annotate(key string, value any) { p.kvs.Add(key, value) }
func (p *PairBuilder) Loggable() bool                 { return p.kvs.Len() > 0 }
func (p *PairBuilder) Structured() bool               { return true }
func (p *PairBuilder) Raw() any {
	p.Collect()

	if p.IncludeMetadata && !p.hasMetadata {
		p.kvs.Add("meta", &p.Base)
		p.hasMetadata = true
	}

	return p.kvs
}

func (p *PairBuilder) String() string {
	if p.cachedOutput != "" && p.kvs.Len() == p.cachedSize {
		return p.cachedOutput
	}

	p.Collect()

	if p.IncludeMetadata && !p.hasMetadata {
		p.kvs.Add("meta", &p.Base)
		p.hasMetadata = true
	}

	out := make([]string, 0, p.kvs.Len())
	var seenMetadata bool
	prod := p.kvs.Iterator().Producer()
	for {
		kv, ok := prod.CheckBlock()
		if !ok {
			break
		}
		if kv.Key == "meta" && (seenMetadata || !p.IncludeMetadata) {
			seenMetadata = true
			continue
		}

		switch val := kv.Value.(type) {
		case string, fmt.Stringer:
			out = append(out, fmt.Sprintf("%s='%s'", kv.Key, val))
		default:
			out = append(out, fmt.Sprintf("%s='%v'", kv.Key, kv.Value))
		}

		if kv.Key == "meta" {
			p.hasMetadata = true
			seenMetadata = true
		}
	}

	p.cachedOutput = strings.Join(out, " ")
	p.cachedSize = p.kvs.Len()

	return p.cachedOutput
}
