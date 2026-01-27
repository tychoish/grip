package message

import (
	"fmt"

	"github.com/tychoish/fun/stw"
)

// FieldsMsgName is the name of the default "message" field in the
// fields structure.
const FieldsMsgName = "msg"

// Fields is a convince type that wraps map[string]any and is
// used for attaching structured metadata to a build request. For
// example:
//
//	message.Fields{"key0", <value>, "key1", <value>}
type Fields stw.Map[string, any]

// MakeFields creates a composer interface from *just* a Fields instance.
func MakeFields(f Fields) Composer {
	return MakeFuture(func() Composer {
		if !f.isLoggable() {
			return Noop()
		}
		return NewKV().Fields(f)
	})
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
	case *KV:
		out, ok := fields.kvs.Load(FieldsMsgName)
		if ok {
			return fmt.Sprint(out)
		}
		return val
	default:
		return val
	}
}

func (f Fields) isLoggable() bool      { return len(f) > 1 || (len(f) == 1 && !f.hasMetadatField()) }
func (f Fields) hasMetadatField() bool { _, ok := f["meta"]; return ok }
