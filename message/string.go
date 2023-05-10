package message

type stringMessage struct {
	Message string `bson:"msg" json:"msg" yaml:"msg"`
	Base    `bson:"meta,omitempty" json:"meta,omitempty" yaml:"meta,omitempty"`
}

// MakeString provides a basic message consisting of a single line.
func MakeString(m string) Composer {
	return &stringMessage{Message: m}
}

func (s *stringMessage) String() string { return s.Message }
func (s *stringMessage) Loggable() bool { return s.Message != "" }
func (s *stringMessage) Raw() any {
	if !s.SkipCollection {
		s.Collect()
	}
	if s.SkipMetadata {
		return stringMessage{Message: s.Message}
	}
	return s
}
