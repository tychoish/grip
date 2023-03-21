package sumogrip

import (
	"os"
	"testing"

	"github.com/tychoish/grip/level"
	"github.com/tychoish/grip/message"
	"github.com/tychoish/grip/send"
)

type SumoSuite struct {
	endpoint string
	client   sumoClient
	sender   sumoLogger
}

func setupFixture(t *testing.T) *SumoSuite {
	t.Helper()
	s := &SumoSuite{
		endpoint: "http://endpointVal",
		client:   &sumoClientMock{},
		sender: sumoLogger{
			Base: send.NewBase("name"),
		},
	}

	s.sender.endpoint = s.endpoint
	s.sender.client = s.client
	s.sender.SetFormatter(send.MakeJSONFormatter())

	if err := s.sender.SetLevel(send.LevelInfo{Default: level.Debug, Threshold: level.Info}); err != nil {
		t.Fatal(err)
	}
	return s
}

func TestConstructorEnvVar(t *testing.T) {
	s := setupFixture(t)
	defer os.Setenv(sumoEndpointEnvVar, os.Getenv(sumoEndpointEnvVar))

	if err := os.Setenv(sumoEndpointEnvVar, s.endpoint); err != nil {
		t.Fatal(err)
	}

	sender, err := MakeSumo()
	if err != nil {
		t.Fatal(err)
	}
	if sender == nil {
		t.Fatal("sender should not have been nil")
	}

	if err = os.Unsetenv(sumoEndpointEnvVar); err != nil {
		t.Fatal(err)
	}

	sender, err = MakeSumo()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if sender != nil {
		t.Fatal("expected nil sender, but got object", sender)
	}
}

func TestConstructorEndpointString(t *testing.T) {
	s := setupFixture(t)
	sender, err := NewSumo("name", s.endpoint)
	if err != nil {
		t.Fatal(err)
	}
	if sender == nil {
		t.Fatal("sender should not have been nil")
	}

	sender, err = NewSumo("name", "invalidEndpoint")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if sender != nil {
		t.Fatal("expected nil sender, but got object", sender)
	}
}

func TestSendMethod(t *testing.T) {
	s := setupFixture(t)
	mock, ok := s.client.(*sumoClientMock)
	if !ok {
		t.Fatal("expected true")
	}
	if mock.numSent != 0 {
		t.Fatalf("expected 'mock.numSent' to be 0, but was %d", mock.numSent)
	}

	m := message.MakeString("hello")
	m.SetPriority(level.Debug)
	s.sender.Send(m)
	if mock.numSent != 0 {
		t.Fatalf("expected 'mock.numSent' to be 0, but was %d", mock.numSent)
	}

	m = message.MakeString("")
	m.SetPriority(level.Alert)
	s.sender.Send(m)
	if mock.numSent != 0 {
		t.Fatalf("expected 'mock.numSent' to be 0, but was %d", mock.numSent)
	}

	m = message.MakeString("world")
	m.SetPriority(level.Alert)
	s.sender.Send(m)
	if mock.numSent != 1 {
		t.Fatalf("expected 'mock.numSent' to be 1, but was %d", mock.numSent)
	}
}

func TestSendMethodWithError(t *testing.T) {
	s := setupFixture(t)
	mock, ok := s.client.(*sumoClientMock)
	if !ok {
		t.Fatal("expected true")
	}
	if mock.numSent != 0 {
		t.Fatalf("expected 'mock.numSent' to be 0, but was %d", mock.numSent)
	}
	if mock.failSend {
		t.Fatal("expected false")
	}

	m := message.MakeString("world")
	m.SetPriority(level.Alert)
	s.sender.Send(m)
	if mock.numSent != 1 {
		t.Fatalf("expected 'mock.numSent' to be 1, but was %d", mock.numSent)
	}

	mock.failSend = true
	s.sender.Send(m)
	if mock.numSent != 1 {
		t.Fatalf("expected 'mock.numSent' to be 1, but was %d", mock.numSent)
	}
}
