// Bytes Messages
//
// The bytes types make it possible to send a byte slice as a message.
package message

type bytesMessage struct {
	Message []byte `bson:"msg" json:"msg" yaml:"msg"`
	Base    `bson:"meta,omitempty" json:"meta,omitempty" yaml:"meta,omitempty"`
}

// MakeBytes provides a basic message consisting of a single line.
func MakeBytes(b []byte) Composer {
	return &bytesMessage{Message: b}
}

func (s *bytesMessage) String() string { return string(s.Message) }
func (s *bytesMessage) Loggable() bool { return len(s.Message) > 0 }
func (s *bytesMessage) Raw() any {
	if s.IncludeMetadata {
		s.Collect()
		return s
	}

	return struct {
		Message []byte `bson:"msg" json:"msg" yaml:"msg"`
	}{
		Message: s.Message,
	}
}
