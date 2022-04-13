package send

import (
	"net/http"
	"os"
	"testing"

	"github.com/stretchr/testify/suite"
	"github.com/tychoish/grip/level"
	"github.com/tychoish/grip/message"
	"github.com/tychoish/grip/send"
)

type SplunkSuite struct {
	info   ConnectionInfo
	sender splunkLogger
	suite.Suite
}

func TestSplunkSuite(t *testing.T) {
	suite.Run(t, new(SplunkSuite))
}

func (s *SplunkSuite) SetupTest() {
	s.sender = splunkLogger{
		info:   ConnectionInfo{},
		client: &splunkClientMock{},
		Base:   send.NewBase("name"),
	}

	s.NoError(s.sender.client.Create(http.DefaultClient, s.info))
	s.NoError(s.sender.SetLevel(send.LevelInfo{Default: level.Debug, Threshold: level.Info}))
}

func (s *SplunkSuite) TestEnvironmentVariableReader() {
	serverVal := "serverURL"
	tokenVal := "token"

	defer os.Setenv(splunkServerURL, os.Getenv(splunkServerURL))
	defer os.Setenv(splunkClientToken, os.Getenv(splunkClientToken))

	s.NoError(os.Setenv(splunkServerURL, serverVal))
	s.NoError(os.Setenv(splunkClientToken, tokenVal))

	info := GetConnectionInfo()

	s.Equal(serverVal, info.ServerURL)
	s.Equal(tokenVal, info.Token)
}

func (s *SplunkSuite) TestNewConstructor() {
	sender, err := NewSender("name", s.info, send.LevelInfo{Default: level.Debug, Threshold: level.Info})
	s.NoError(err)
	s.NotNil(sender)
}

func (s *SplunkSuite) TestAutoConstructor() {
	serverVal := "serverURL"
	tokenVal := "token"

	defer os.Setenv(splunkServerURL, os.Getenv(splunkServerURL))
	defer os.Setenv(splunkClientToken, os.Getenv(splunkClientToken))

	s.NoError(os.Setenv(splunkServerURL, serverVal))
	s.NoError(os.Setenv(splunkClientToken, tokenVal))

	sender, err := MakeSender("name")
	s.NoError(err)
	s.NotNil(sender)
}

func (s *SplunkSuite) TestAutoConstructorFailsWhenEnvVarFails() {
	serverVal := ""
	tokenVal := ""

	defer os.Setenv(splunkServerURL, os.Getenv(splunkServerURL))
	defer os.Setenv(splunkClientToken, os.Getenv(splunkClientToken))

	s.NoError(os.Setenv(splunkServerURL, serverVal))
	s.NoError(os.Setenv(splunkClientToken, tokenVal))

	sender, err := MakeSender("name")
	s.Error(err)
	s.Nil(sender)

	serverVal = "serverVal"

	s.NoError(os.Setenv(splunkServerURL, serverVal))
	sender, err = MakeSender("name")
	s.Error(err)
	s.Nil(sender)
}

func (s *SplunkSuite) TestSendMethod() {
	mock, ok := s.sender.client.(*splunkClientMock)
	s.True(ok)
	s.Equal(mock.numSent, 0)
	s.Equal(mock.httpSent, 0)

	m := message.NewDefaultMessage(level.Debug, "hello")
	s.sender.Send(m)
	s.Equal(mock.numSent, 0)
	s.Equal(mock.httpSent, 0)

	m = message.NewDefaultMessage(level.Alert, "")
	s.sender.Send(m)
	s.Equal(mock.numSent, 0)
	s.Equal(mock.httpSent, 0)

	m = message.NewDefaultMessage(level.Alert, "world")
	s.sender.Send(m)
	s.Equal(mock.numSent, 1)
	s.Equal(mock.httpSent, 1)
}

func (s *SplunkSuite) TestSendMethodWithError() {
	mock, ok := s.sender.client.(*splunkClientMock)
	s.True(ok)
	s.Equal(mock.numSent, 0)
	s.Equal(mock.httpSent, 0)
	s.False(mock.failSend)

	m := message.NewDefaultMessage(level.Alert, "world")
	s.sender.Send(m)
	s.Equal(mock.numSent, 1)
	s.Equal(mock.httpSent, 1)

	mock.failSend = true
	s.sender.Send(m)
	s.Equal(mock.numSent, 1)
	s.Equal(mock.httpSent, 1)
}

func (s *SplunkSuite) TestBatchSendMethod() {
	mock, ok := s.sender.client.(*splunkClientMock)
	s.True(ok)
	s.Equal(mock.numSent, 0)
	s.Equal(mock.httpSent, 0)

	m1 := message.NewDefaultMessage(level.Alert, "hello")
	m2 := message.NewDefaultMessage(level.Debug, "hello")
	m3 := message.NewDefaultMessage(level.Alert, "")
	m4 := message.NewDefaultMessage(level.Alert, "hello")

	g := message.MakeGroupComposer(m1, m2, m3, m4)

	s.sender.Send(g)
	s.Equal(mock.numSent, 2)
	s.Equal(mock.httpSent, 1)
}

func (s *SplunkSuite) TestBatchSendMethodWithEror() {
	mock, ok := s.sender.client.(*splunkClientMock)
	s.True(ok)
	s.Equal(mock.numSent, 0)
	s.Equal(mock.httpSent, 0)
	s.False(mock.failSend)

	m1 := message.NewDefaultMessage(level.Alert, "hello")
	m2 := message.NewDefaultMessage(level.Debug, "hello")
	m3 := message.NewDefaultMessage(level.Alert, "")
	m4 := message.NewDefaultMessage(level.Alert, "hello")

	g := message.MakeGroupComposer(m1, m2, m3, m4)

	s.sender.Send(g)
	s.Equal(mock.numSent, 2)
	s.Equal(mock.httpSent, 1)

	mock.failSend = true
	s.sender.Send(g)
	s.Equal(mock.numSent, 2)
	s.Equal(mock.httpSent, 1)
}