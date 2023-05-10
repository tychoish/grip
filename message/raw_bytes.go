// Bytes Messages
//
// The bytes types make it possible to send a byte slice as a message.
package message

type bytesMessage struct {
	data []byte
	Base
}

// MakeBytes provides a basic message consisting of a single line.
func MakeBytes(b []byte) Composer {
	return &bytesMessage{data: b}
}

func (s *bytesMessage) String() string { return string(s.data) }
func (s *bytesMessage) Loggable() bool { return len(s.data) > 0 }

func (s *bytesMessage) Raw() any {
	if !s.SkipCollection {
		s.Collect()
	}

	out := struct {
		Meta    *Base  `bson:"meta,omitempty" json:"meta,omitempty" yaml:"meta,omitempty"`
		Message string `bson:"msg" json:"msg" yaml:"msg"`
	}{
		Message: string(s.data),
	}

	if !s.SkipMetadata {
		out.Meta = &s.Base
	}

	return out
}
