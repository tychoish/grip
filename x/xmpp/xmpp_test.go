package xmpp

import (
	"os"
	"testing"

	"github.com/tychoish/grip/level"
	"github.com/tychoish/grip/message"
)

func TestEnvironmentVariableReader(t *testing.T) {
	hostVal := "hostName"
	userVal := "userName"
	passVal := "passName"

	defer os.Setenv(xmppHostEnvVar, os.Getenv(xmppHostEnvVar))
	defer os.Setenv(xmppUsernameEnvVar, os.Getenv(xmppUsernameEnvVar))
	defer os.Setenv(xmppPasswordEnvVar, os.Getenv(xmppPasswordEnvVar))

	if err := os.Setenv(xmppHostEnvVar, hostVal); err != nil {
		t.Fatal(err)
	}
	if err := os.Setenv(xmppUsernameEnvVar, userVal); err != nil {
		t.Fatal(err)
	}
	if err := os.Setenv(xmppPasswordEnvVar, passVal); err != nil {
		t.Fatal(err)
	}

	info := GetConnectionInfo()

	if info.Hostname != hostVal {
		t.Error("incorrect value for info.Hostname:", hostVal)
	}
	if info.Username != userVal {
		t.Error("incorrect value for info.Username:", userVal)
	}
	if info.Password != passVal {
		t.Error("incorrect value for info.Password:", passVal)
	}
}

func TestNewConstructor(t *testing.T) {
	info := ConnectionInfo{client: &xmppClientMock{}}
	sender, err := NewSender("target", info)
	if err != nil {
		t.Fatal(err)
	}
	if sender == nil {
		t.Fatal("expected value, but got nil")
	}
}

func TestNewConstructorFailsWhenClientCreateFails(t *testing.T) {
	info := ConnectionInfo{client: &xmppClientMock{failCreate: true}}

	sender, err := NewSender("target", info)
	if err == nil {
		t.Fatal("expected error but got nil")
	}
	if sender != nil {
		t.Fatal("expected nil, but got value")
	}
}

func TestCloseMethod(t *testing.T) {
	info := ConnectionInfo{client: &xmppClientMock{}}
	sender, err := NewSender("target", info)
	if err != nil {
		t.Fatal(err)
	}
	if sender == nil {
		t.Fatal("expected value, but got nil")
	}

	mock, ok := info.client.(*xmppClientMock)
	if !ok {
		t.Fatal("expected true but got falsey value")
	}
	if mock.numCloses != 0 {
		t.Error("incorrect value for mock.numCloses:", 0)
	}
	if err := sender.Close(); err != nil {
		t.Fatal(err)
	}
	if mock.numCloses != 1 {
		t.Error("incorrect value for mock.numCloses:", 1)
	}
}

func TestAutoConstructorErrorsWithoutValidEnvVar(t *testing.T) {
	sender, err := MakeSender("target")
	if err == nil {
		t.Fatal("expected error but got nil")
	}
	if sender != nil {
		t.Fatal("expected nil, but got value")
	}
}

func TestSendMethod(t *testing.T) {
	info := ConnectionInfo{client: &xmppClientMock{}}
	sender, err := NewSender("target", info)
	if err != nil {
		t.Fatal(err)
	}
	if sender == nil {
		t.Fatal("expected value, but got nil")
	}

	mock, ok := info.client.(*xmppClientMock)
	if !ok {
		t.Fatal("expected true but got falsey value")
	}
	if 0 != mock.numSent {
		t.Error("incorrect value for mock.numSent:", 0)
	}
	var m message.Composer

	m = message.MakeString("hello")
	m.SetPriority(level.Debug)
	sender.SetPriority(level.Info)
	sender.Send(m)
	if 0 != mock.numSent {
		t.Error("incorrect value for mock.numSent:", 0)
	}

	m = message.MakeString("")
	m.SetPriority(level.Alert)
	sender.Send(m)
	if 0 != mock.numSent {
		t.Error("incorrect value for mock.numSent:", 0)
	}

	m = message.MakeString("world")
	m.SetPriority(level.Alert)
	sender.Send(m)
	if 1 != mock.numSent {
		t.Error("incorrect value for 1:", mock.numSent)
	}
}

func TestSendMethodWithError(t *testing.T) {
	info := ConnectionInfo{client: &xmppClientMock{}}
	sender, err := NewSender("target", info)
	if err != nil {
		t.Fatal(err)
	}
	if sender == nil {
		t.Fatal("expected value, but got nil")
	}

	mock, ok := info.client.(*xmppClientMock)
	if !ok {
		t.Fatal("expected true but got falsey value")
	}
	if 0 != mock.numSent {
		t.Error("incorrect value for mock.numSent:", 0)
	}
	if mock.failSend {
		t.Error("failed to send but should not have")
	}

	m := message.MakeString("world")
	m.SetPriority(level.Alert)
	sender.Send(m)
	if 1 != mock.numSent {
		t.Error("incorrect value for mock.numSent:", 1)
	}

	mock.failSend = true
	sender.Send(m)
	if 1 != mock.numSent {
		t.Error("incorrect value for mock.numSent:", 1)
	}
}
