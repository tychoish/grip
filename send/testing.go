package send

import (
	"testing"

	"github.com/tychoish/grip/level"
	"github.com/tychoish/grip/message"
)

type testLogger struct {
	t *testing.T
	Base
}

// MakeTesting constructs a fully configured Sender implementation that
// logs using the testing.T's logging facility for better integration
// with unit tests. Construct and register such a sender for
// grip.Journaler instances that you use inside of tests to have
// logging that correctly respects go test's verbosity.
//
// By default, this constructor creates a sender with a level threshold
// of "debug" and a default log level of "info."
func MakeTesting(t *testing.T) Sender {
	s := &testLogger{t: t}
	s.SetName(t.Name())
	if err := s.SetLevel(LevelInfo{Threshold: level.Debug, Default: level.Info}); err != nil {
		t.Fatalf("problem setting up logger %s: %v", t.Name(), err)
	}
	return s
}

func (s *testLogger) Send(m message.Composer) {
	if s.Level().ShouldLog(m) {
		out, err := s.Formatter()(m)
		if err != nil {
			s.t.Logf("formating message [type=%T]: %v", m, err)
			return
		}

		s.t.Log(out)
	}
}
