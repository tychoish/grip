// Bytes Messages
//
// The bytes types make it possible to send a byte slice as a message.
package message

import "github.com/tychoish/grip/level"

type bytesMessage struct {
	data         []byte
	skipMetadata bool
	Base
}

// NewBytes provides a Composer interface around a byte slice,
// which are always logable unless the string is empty.
func NewBytes(p level.Priority, b []byte) Composer {
	m := &bytesMessage{data: b}
	_ = m.SetPriority(p)
	return m
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

// NewSimpleBytes produces a bytes-wrapping message with the
// specified priority but does not collect metadata.
func NewSimpleBytes(p level.Priority, b []byte) Composer {
	m := &bytesMessage{data: b, skipMetadata: true}
	_ = m.SetPriority(p)

	return m
}

func (s *bytesMessage) String() string { return string(s.data) }
func (s *bytesMessage) Loggable() bool { return len(s.data) > 0 }

func (s *bytesMessage) Raw() interface{} {
	if !s.skipMetadata {
		_ = s.Collect()
	}
	return struct {
		Metadata *Base  `bson:"metadata" json:"metadata" yaml:"metadata"`
		Message  string `bson:"message" json:"message" yaml:"message"`
	}{
		Metadata: &s.Base,
		Message:  string(s.data),
	}
}
