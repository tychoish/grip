package splunk

import (
	"net/http"
	"os"
	"testing"

	"github.com/tychoish/grip/level"
	"github.com/tychoish/grip/message"
	"github.com/tychoish/grip/send"
)

type SplunkSuite struct {
	info   ConnectionInfo
	sender splunkLogger
}

func setupFixture(t *testing.T) *SplunkSuite {
	t.Helper()
	s := &SplunkSuite{}
	s.sender = splunkLogger{
		info:   ConnectionInfo{},
		client: &splunkClientMock{},
		Base:   send.NewBase("name"),
	}

	if err := s.sender.client.Create(http.DefaultClient, s.info); err != nil {
		t.Fatal(err)
	}
	if err := s.sender.SetLevel(send.LevelInfo{Default: level.Debug, Threshold: level.Info}); err != nil {
		t.Fatal(err)
	}
	return s
}

func TestEnvironmentVariableReader(t *testing.T) {
	serverVal := "serverURL"
	tokenVal := "token"

	defer os.Setenv(splunkServerURL, os.Getenv(splunkServerURL))
	defer os.Setenv(splunkClientToken, os.Getenv(splunkClientToken))

	if err := os.Setenv(splunkServerURL, serverVal); err != nil {
		t.Fatal(err)
	}
	if err := os.Setenv(splunkClientToken, tokenVal); err != nil {
		t.Fatal(err)
	}

	info := GetConnectionInfo()

	if serverVal != info.ServerURL {
		t.Errorf("%q should be equal to %q", serverVal, info.ServerURL)
	}
	if tokenVal != info.Token {
		t.Errorf("%q should be equal to %q", tokenVal, info.Token)
	}
}

func TestNewConstructor(t *testing.T) {
	s := setupFixture(t)
	sender, err := NewSender("name", s.info, send.LevelInfo{Default: level.Debug, Threshold: level.Info})
	if err := err; err != nil {
		t.Fatal(err)
	}
	if sender == nil {
		t.Error("should not have been nil")
	}
}

func TestAutoConstructor(t *testing.T) {
	serverVal := "serverURL"
	tokenVal := "token"

	defer os.Setenv(splunkServerURL, os.Getenv(splunkServerURL))
	defer os.Setenv(splunkClientToken, os.Getenv(splunkClientToken))

	if err := os.Setenv(splunkServerURL, serverVal); err != nil {
		t.Fatal(err)
	}
	if err := os.Setenv(splunkClientToken, tokenVal); err != nil {
		t.Fatal(err)
	}

	sender, err := MakeSender("name")
	if err := err; err != nil {
		t.Fatal(err)
	}
	if sender == nil {
		t.Error("should not have been nil")
	}
}

func TestAutoConstructorFailsWhenEnvVarFails(t *testing.T) {
	serverVal := ""
	tokenVal := ""

	defer os.Setenv(splunkServerURL, os.Getenv(splunkServerURL))
	defer os.Setenv(splunkClientToken, os.Getenv(splunkClientToken))

	if err := os.Setenv(splunkServerURL, serverVal); err != nil {
		t.Fatal(err)
	}
	if err := os.Setenv(splunkClientToken, tokenVal); err != nil {
		t.Fatal(err)
	}

	sender, err := MakeSender("name")
	if err == nil {
		t.Fatal("error should not have been nil")
	}
	if sender != nil {
		t.Fatal("sender should have been nil")
	}

	serverVal = "serverVal"

	if err = os.Setenv(splunkServerURL, serverVal); err != nil {
		t.Fatal(err)
	}
	sender, err = MakeSender("name")
	if err == nil {
		t.Fatal("error should not have been nil")
	}
	if sender != nil {
		t.Fatal("sender should have been nil")
	}

}

func TestSendMethod(t *testing.T) {
	s := setupFixture(t)
	mock, ok := s.sender.client.(*splunkClientMock)
	if !ok {
		t.Error("shoud not have been false")
	}
	if mock.numSent != 0 {
		t.Errorf("%q should be equal to %q", mock.numSent, 0)
	}
	if mock.httpSent != 0 {
		t.Errorf("%q should be equal to %q", mock.httpSent, 0)
	}

	var m message.Composer
	m = message.MakeString("hello")
	m.SetPriority(level.Debug)
	s.sender.Send(m)
	if mock.numSent != 0 {
		t.Errorf("%q should be equal to %q", mock.numSent, 0)
	}
	if mock.httpSent != 0 {
		t.Errorf("%q should be equal to %q", mock.httpSent, 0)
	}

	m = message.MakeString("")
	m.SetPriority(level.Alert)
	s.sender.Send(m)
	if mock.numSent != 0 {
		t.Errorf("%q should be equal to %q", mock.numSent, 0)
	}
	if mock.httpSent != 0 {
		t.Errorf("%q should be equal to %q", mock.httpSent, 0)
	}

	m = message.MakeString("world")
	m.SetPriority(level.Alert)
	s.sender.Send(m)
	if mock.numSent != 1 {
		t.Errorf("%q should be equal to %q", mock.numSent, 1)
	}
	if mock.httpSent != 1 {
		t.Errorf("%q should be equal to %q", mock.httpSent, 1)
	}
}

func TestSendMethodWithError(t *testing.T) {
	s := setupFixture(t)
	mock, ok := s.sender.client.(*splunkClientMock)
	if !ok {
		t.Error("shoud not have been false")
	}
	if mock.numSent != 0 {
		t.Errorf("%q should be equal to %q", mock.numSent, 0)
	}
	if mock.httpSent != 0 {
		t.Errorf("%q should be equal to %q", mock.httpSent, 0)
	}
	if mock.failSend {
		t.Error("shoud not have been true")
	}

	m := message.MakeString("world")
	m.SetPriority(level.Alert)
	s.sender.Send(m)
	if mock.numSent != 1 {
		t.Errorf("%q should be equal to %q", mock.numSent, 1)
	}
	if mock.httpSent != 1 {
		t.Errorf("%q should be equal to %q", mock.httpSent, 1)
	}

	mock.failSend = true
	s.sender.Send(m)
	if mock.numSent != 1 {
		t.Errorf("%q should be equal to %q", mock.numSent, 1)
	}
	if mock.httpSent != 1 {
		t.Errorf("%q should be equal to %q", mock.httpSent, 1)
	}
}

func TestBatchSendMethod(t *testing.T) {
	s := setupFixture(t)
	mock, ok := s.sender.client.(*splunkClientMock)
	if !ok {
		t.Error("shoud not have been false")
	}
	if mock.numSent != 0 {
		t.Errorf("%q should be equal to %q", mock.numSent, 0)
	}
	if mock.httpSent != 0 {
		t.Errorf("%q should be equal to %q", mock.httpSent, 0)
	}

	m1 := message.MakeString("hello")
	m1.SetPriority(level.Alert)
	m2 := message.MakeString("hello")
	m2.SetPriority(level.Debug)
	m3 := message.MakeString("")
	m3.SetPriority(level.Alert)
	m4 := message.MakeString("hello")
	m4.SetPriority(level.Alert)

	g := message.BuildGroupComposer(m1, m2, m3, m4)

	s.sender.Send(g)
	if mock.numSent != 2 {
		t.Errorf("%q should be equal to %q", mock.numSent, 2)
	}
	if mock.httpSent != 1 {
		t.Errorf("%q should be equal to %q", mock.httpSent, 1)
	}
}

func TestBatchSendMethodWithEror(t *testing.T) {
	s := setupFixture(t)
	mock, ok := s.sender.client.(*splunkClientMock)
	if !ok {
		t.Error("shoud not have been false")
	}
	if mock.numSent != 0 {
		t.Errorf("%q should be equal to %q", mock.numSent, 0)
	}
	if mock.httpSent != 0 {
		t.Errorf("%q should be equal to %q", mock.httpSent, 0)
	}
	if mock.failSend {
		t.Error("shoud not have been true")
	}

	m1 := message.MakeString("hello")
	m1.SetPriority(level.Alert)
	m2 := message.MakeString("hello")
	m2.SetPriority(level.Debug)
	m3 := message.MakeString("")
	m3.SetPriority(level.Alert)
	m4 := message.MakeString("hello")
	m4.SetPriority(level.Alert)

	g := message.BuildGroupComposer(m1, m2, m3, m4)

	s.sender.Send(g)
	if mock.numSent != 2 {
		t.Errorf("%q should be equal to %q", mock.numSent, 2)
	}
	if mock.httpSent != 1 {
		t.Errorf("%q should be equal to %q", mock.httpSent, 1)
	}

	mock.failSend = true
	s.sender.Send(g)
	if mock.numSent != 2 {
		t.Errorf("%q should be equal to %q", mock.numSent, 2)
	}
	if mock.httpSent != 1 {
		t.Errorf("%q should be equal to %q", mock.httpSent, 1)
	}
}
