package message

import (
	"encoding/json"
	"fmt"
	"strings"
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
	hasMetadata  bool
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

func (m *kvMsg) Annotate(key string, value any) error {
	m.cachedOutput = ""
	m.fields = append(m.fields, KV{Key: key, Value: value})
	return nil
}

func (m *kvMsg) Loggable() bool   { return len(m.fields) > 0 }
func (m *kvMsg) Structured() bool { return true }
func (m *kvMsg) Raw() any {
	if !m.skipMetadata && !m.hasMetadata {
		m.Collect()
		m.fields = append(m.fields, KV{Key: "metadata", Value: &m.Base})
		m.hasMetadata = true
		m.cachedOutput = ""
	}

	return m.fields
}
func (m *kvMsg) String() string {
	if !m.Loggable() {
		return ""
	}
	if m.cachedOutput != "" {
		return m.cachedOutput
	}

	out := make([]string, len(m.fields))
	for idx, kv := range m.fields {
		switch val := kv.Value.(type) {
		case string, fmt.Stringer:
			out[idx] = fmt.Sprintf("%s='%s'", kv.Key, val)
		default:
			out[idx] = fmt.Sprintf("%s='%v'", kv.Key, kv.Value)
		}
		if kv.Key == "metadata" {
			m.hasMetadata = true
		}
	}

	m.cachedOutput = strings.Join(out, " ")

	return m.cachedOutput
}
