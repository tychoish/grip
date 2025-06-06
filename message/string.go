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

func (s *stringMessage) setupField() {
	s.Collect()
	s.fm = &fieldMessage{
		fields:  s.Context,
		Base:    s.Base,
		message: s.Message,
	}
}

func (s *stringMessage) Loggable() bool {
	switch {
	case (s.fm != nil && s.fm.Loggable()):
		return true
	case len(s.Context) > 0:
		return true
	case s.Message != "":
		return true
	default:
		return false
	}
}

func (s *stringMessage) String() string {
	switch {
	case s.fm != nil:
		return s.fm.String()
	case len(s.Context) > 0:
		s.setupField()
		return s.fm.String()
	default:
		return s.Message
	}
}

func (s *stringMessage) Raw() any {
	switch {
	case s.fm != nil:
		return s.fm.Raw()
	case len(s.Context) > 0:
		s.setupField()
		return s.fm.Raw()
	default:
		return struct {
			Message string `bson:"msg" json:"msg" yaml:"msg"`
		}{
			Message: s.Message,
		}
	}
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
