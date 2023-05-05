package message

import (
	"fmt"
	"sort"
	"strings"
)

// FieldsMsgName is the name of the default "message" field in the
// fields structure.
const FieldsMsgName = "message"

type fieldMessage struct {
	fields       Fields
	cachedOutput string
	skipMetadata bool
	Base
}

// Fields is a convince type that wraps map[string]any and is
// used for attaching structured metadata to a build request. For
// example:
//
//	message.Fields{"key0", <value>, "key1", <value>}
type Fields map[string]any

// MakeFields creates a composer interface from *just* a Fields instance.
func MakeFields(f Fields) Composer {
	m := &fieldMessage{fields: f}
	m.setup()
	return m
}

// MakeSimpleFields returns a structured Composer that does
// not attach basic logging metadata.
func MakeSimpleFields(f Fields) Composer {
	m := &fieldMessage{fields: f, skipMetadata: true}
	m.setup()
	return m
}

// GetDefaultFieldsMessage returns a "short" message form, to avoid
// needing to call .String() on the type, which produces a string form
// of the message. If the message has a short form (either in the map,
// or separate), it's returned, otherwise the "val" is returned.
//
// For composers not that don't wrap Fields, this function will always
// return the input value.
func GetDefaultFieldsMessage(msg Composer, val string) string {
	switch fields := msg.(type) {
	case *fieldMessage:
		if fields.fields == nil {
			return val
		}

		if str, ok := fields.fields[FieldsMsgName]; ok {
			return fmt.Sprintf("%s", str)
		}

		return val
	default:
		return val
	}
}

func FieldsFromMap[V any](in map[string]V) Fields {
	out := make(Fields, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

////////////////////////////////////////////////////////////////////////
//
// Implementation
//
////////////////////////////////////////////////////////////////////////

func (m *fieldMessage) setup() {
	if m.fields == nil {
		m.fields = Fields{}
	}

	if m.skipMetadata {
		return
	}

	m.Collect()

	if b, ok := m.fields["metadata"]; !ok {
		m.fields["metadata"] = &m.Base
	} else if _, ok = b.(*Base); ok {
		m.fields["metadata"] = &m.Base
	}
}

func (*fieldMessage) Structured() bool { return true }

func (m *fieldMessage) Loggable() bool {
	if len(m.fields) > 1 || (len(m.fields) == 1 && !m.fields.hasMetadatField()) {
		// it's not loggable if there's more than one field or
		// if there is only one field that isn't the metadata field added by m.setup().
		return true
	}
	return false
}

func (f Fields) hasMetadatField() bool { _, ok := f["metadata"]; return ok }

var skippedFields = map[string]struct{}{
	FieldsMsgName: {},
	"time":        {},
	"metadata":    {},
}

func (m *fieldMessage) String() string {
	if !m.Loggable() {
		return ""
	}

	if m.cachedOutput != "" {
		return m.cachedOutput
	}

	out := make([]string, 0, len(m.fields))
	if _, ok := m.fields[FieldsMsgName]; ok {
		out = append(out, "")
	}

	for k, v := range m.fields {
		if _, ok := skippedFields[k]; ok {
			continue
		}

		switch val := v.(type) {
		case fmt.Stringer, string:
			out = append(out, fmt.Sprintf("%s='%s'", k, val))
		default:
			out = append(out, fmt.Sprintf("%s='%v'", k, v))
		}
	}

	sort.Strings(out)
	if _, ok := m.fields[FieldsMsgName]; ok {
		out[0] = fmt.Sprintf("%s='%v'", FieldsMsgName, m.fields[FieldsMsgName])
	}
	m.cachedOutput = strings.Join(out, " ")

	return m.cachedOutput
}

func (m *fieldMessage) Raw() any                       { return m.fields }
func (m *fieldMessage) Annotate(key string, value any) { m.fields[key] = value }
