package message

import "github.com/tychoish/grip/level"

type stringMessage struct {
	Message string `bson:"message" json:"message" yaml:"message"`
	Base    `bson:"metadata" json:"metadata" yaml:"metadata"`

	skipMetadata bool
}

// NewString provides a Composer interface around a single
// string, which are always logable unless the string is empty.
func NewString(p level.Priority, message string) Composer {
	m := &stringMessage{Message: message}
	_ = m.SetPriority(p)
	return m
}

// MakeString provides a basic message consisting of a single line.
func MakeString(m string) Composer {
	return &stringMessage{Message: m}
}

// MakeSimpleString produces a string message that does not attach
// process metadata.
func MakeSimpleString(m string) Composer {
	return &stringMessage{Message: m, skipMetadata: true}
}

// NewSimpleString produces a string message with a priority
// that does not attach process metadata.
func NewSimpleString(p level.Priority, message string) Composer {
	m := &stringMessage{Message: message, skipMetadata: true}
	_ = m.SetPriority(p)
	return m
}

func (s *stringMessage) String() string { return s.Message }
func (s *stringMessage) Loggable() bool { return s.Message != "" }
func (s *stringMessage) Raw() interface{} {
	if !s.skipMetadata {
		_ = s.Collect()
	}
	return s
}
