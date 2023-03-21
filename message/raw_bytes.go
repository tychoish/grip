// Bytes Messages
//
// The bytes types make it possible to send a byte slice as a message.
package message

type bytesMessage struct {
	data         []byte
	skipMetadata bool
	Base
}

// MakeBytes provides a basic message consisting of a single line.
func MakeBytes(b []byte) Composer {
	return &bytesMessage{data: b}
}

// MakeSimpleBytes produces a bytes-wrapping message but does not
// collect metadata.
func MakeSimpleBytes(b []byte) Composer {
	return &bytesMessage{data: b, skipMetadata: true}
}

func (s *bytesMessage) String() string { return string(s.data) }
func (s *bytesMessage) Loggable() bool { return len(s.data) > 0 }

func (s *bytesMessage) Raw() any {
	if !s.skipMetadata {
		s.Collect()
	}
	return struct {
		Metadata *Base  `bson:"metadata" json:"metadata" yaml:"metadata"`
		Message  string `bson:"message" json:"message" yaml:"message"`
	}{
		Metadata: &s.Base,
		Message:  string(s.data),
	}
}
