package message

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/tychoish/grip/level"
)

// KV represents an arbitrary key value pair for use in structured
// logging. Like the Fields type, but without the map type and it's
// restrictions (e.g. unique keys, random ordering in iteration,) and
// allows some Sender implementations to implement fast-path
// processing of these messages.
type KV struct {
	Key   string
	Value any
}

// KVs represents a collection of KV pairs, and is convertable to a
// Fields implementation, (e.g. a map). It implements MarshalJSON and
// UnmarshalJSON, via the map conversion.
type KVs []KV

func (kvs KVs) ToFields() Fields {
	out := make(Fields, len(kvs))
	for _, kv := range kvs {
		out[kv.Key] = kv.Value
	}
	return out
}

func (kvs KVs) MarshalJSON() ([]byte, error) { return json.Marshal(kvs.ToFields()) }
func (kvs *KVs) UnmarshalJSON(in []byte) error {
	f := Fields{}
	if err := json.Unmarshal(in, &f); err != nil {
		return err
	}
	new := make(KVs, 0, len(f))
	for k, v := range f {
		new = append(new, KV{Key: k, Value: v})
	}
	*kvs = new
	return nil
}

type kvMsg struct {
	fields       KVs
	skipMetadata bool
	cachedOutput string
	Base
}

// MakeKVs constructs a new Composer using KV pairs.
func MakeKVs(kvs KVs) Composer { return &kvMsg{fields: kvs} }

// MakeKV constructs a new Composer using KV pairs.
func MakeKV(kvs ...KV) Composer { return MakeKVs(kvs) }

// MakeSimpleKVs constructs a composer using KV pairs that does *not*
// populate the "base" structure (with time, hostname and pid information).
func MakeSimpleKVs(kvs KVs) Composer { return &kvMsg{fields: kvs, skipMetadata: true} }

// MakeSimpleKV constructs a composer using KV pairs that does *not*
// populate the "base" structure (with time, hostname and pid information).
func MakeSimpleKV(kvs ...KV) Composer { return MakeSimpleKVs(kvs) }

// NewKV constructs a new Composer using KV pairs wit the specified level.
func NewKV(p level.Priority, kvs ...KV) Composer { return NewKVs(p, kvs) }

// NewKVs constructs a new Composer using KV pairs wit the specified level.
func NewKVs(p level.Priority, kvs KVs) Composer {
	m := &kvMsg{fields: kvs, skipMetadata: true}
	_ = m.SetPriority(p)
	return m
}

// NewSimpleKV constructs a composer using KV pairs that does *not*
// populate the "base" structure (with time, hostname and pid
// information) with the specified level.
func NewSimpleKV(p level.Priority, kvs ...KV) Composer { return NewSimpleKVs(p, kvs) }

// NewSimpleKVs constructs a composer using KV pairs that does *not*
// populate the "base" structure (with time, hostname and pid
// information) with the specified level.
func NewSimpleKVs(p level.Priority, kvs KVs) Composer {
	m := &kvMsg{fields: kvs, skipMetadata: true}
	_ = m.SetPriority(p)
	return m
}

func (m *kvMsg) Annotate(key string, value any) error {
	m.fields = append(m.fields, KV{Key: key, Value: value})
	return nil
}

func (m *kvMsg) Loggable() bool   { return len(m.fields) > 0 }
func (m *kvMsg) Structured() bool { return true }
func (m *kvMsg) Raw() any         { return m.fields }
func (m *kvMsg) String() string {
	if !m.Loggable() {
		return ""
	}
	if m.cachedOutput != "" {
		return m.cachedOutput
	}
	if !m.skipMetadata {
		m.Collect()
		m.fields = append(m.fields, KV{Key: "metadata", Value: &m.Base})
	}

	out := make([]string, len(m.fields))
	for idx, kv := range m.fields {
		if str, ok := kv.Value.(fmt.Stringer); ok {
			out[idx] = fmt.Sprintf("%s='%s'", kv.Key, str.String())

		} else {
			out[idx] = fmt.Sprintf("%s='%v'", kv.Key, kv.Value)
		}
	}

	sort.Strings(out)
	m.cachedOutput = strings.Join(out, " ")

	return m.cachedOutput
}
