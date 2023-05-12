package message

type stringMessage struct {
	Message string `bson:"msg" json:"msg" yaml:"msg"`
	Base    `bson:"meta,omitempty" json:"meta,omitempty" yaml:"meta,omitempty"`
	fm      *fieldMessage
}

// MakeString provides a basic message consisting of a single line.
func MakeString(m string) Composer {
	return &stringMessage{Message: m}
}

func (s *stringMessage) String() string {
	if s.fm != nil {
		return s.fm.String()
	}

	if len(s.Base.Context) > 0 {
		s.setupField()
		return s.fm.String()
	}

	return s.Message
}

func (s *stringMessage) setupField() {
	s.fm = &fieldMessage{
		fields:  s.Base.Context,
		Base:    s.Base,
		message: s.Message,
	}
}

func (s *stringMessage) Loggable() bool {
	return s.Message != "" || len(s.Base.Context) > 0 || (s.fm != nil && s.fm.Loggable())
}

func (s *stringMessage) Annotate(k string, v any) {
	if s.fm == nil {
		s.Base.Annotate(k, v)
		return
	}
	s.fm.Annotate(k, v)
}

func (s *stringMessage) SetOption(opts ...Option) {
	if s.fm == nil {
		s.Base.SetOption(opts...)
		return
	}
	s.fm.SetOption(opts...)
}

func (s *stringMessage) Raw() any {
	if s.fm != nil {
		return s.fm.Raw()
	}

	if len(s.Base.Context) > 0 {
		s.setupField()
		return s.fm.String()
	}

	if s.SkipMetadata {
		return stringMessage{Message: s.Message}
	}

	if !s.SkipCollection {
		s.Collect()
	}

	return s
}
