package message

type stringMessage struct {
	Message string `bson:"message" json:"message" yaml:"message"`
	Base    `bson:"metadata" json:"metadata" yaml:"metadata"`

	skipMetadata bool
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

func (s *stringMessage) String() string { return s.Message }
func (s *stringMessage) Loggable() bool { return s.Message != "" }
func (s *stringMessage) Raw() any {
	if !s.skipMetadata {
		s.Collect()
	}
	return s
}
