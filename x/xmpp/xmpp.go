package xmpp

import (
	"fmt"
	"log"
	"os"
	"strings"

	xmpp "github.com/mattn/go-xmpp"
	"github.com/tychoish/grip/message"
	"github.com/tychoish/grip/send"
)

type xmppLogger struct {
	target string
	info   ConnectionInfo
	*send.Base
}

// ConnectionInfo stores all information needed to connect to an
// XMPP (jabber) server to send log messages.
type ConnectionInfo struct {
	Hostname string
	Username string
	Password string

	client xmppClient
}

const (
	xmppHostEnvVar     = "GRIP_XMPP_HOSTNAME"
	xmppUsernameEnvVar = "GRIP_XMPP_USERNAME"
	xmppPasswordEnvVar = "GRIP_XMPP_PASSWORD"
)

// GetConnectionInfo builds an XMPPConnectionInfo structure
// reading default values from the following environment variables:
//
//	GRIP_XMPP_HOSTNAME
//	GRIP_XMPP_USERNAME
//	GRIP_XMPP_PASSWORD
func GetConnectionInfo() ConnectionInfo {
	return ConnectionInfo{
		Hostname: os.Getenv(xmppHostEnvVar),
		Username: os.Getenv(xmppUsernameEnvVar),
		Password: os.Getenv(xmppPasswordEnvVar),
	}
}

// NewSender constructs a new Sender implementation that sends
// messages to an XMPP user, "target", using the credentials specified in
// the XMPPConnectionInfo struct. The constructor will attempt to exablish
// a connection to the server via SSL, falling back automatically to an
// unencrypted connection if the the first attempt fails.
func NewSender(name, target string, info ConnectionInfo, l send.LevelInfo) (send.Sender, error) {
	s, err := constructXMPPLogger(name, target, info)
	if err != nil {
		return nil, err
	}

	if err := s.SetLevel(l); err != nil {
		return nil, err
	}

	s.SetName(name)

	return s, nil
}

// MakeSender constructs an XMPP logging backend that reads the
// hostname, username, and password from environment variables:
//
//   - GRIP_XMPP_HOSTNAME
//   - GRIP_XMPP_USERNAME
//   - GRIP_XMPP_PASSWORD
//
// The instance is otherwise unconquered. Call SetName or inject it
// into a Journaler instance using SetSender before using.
func MakeSender(target string) (send.Sender, error) {
	info := GetConnectionInfo()

	s, err := constructXMPPLogger("", target, info)
	if err != nil {
		return nil, err
	}

	return s, nil
}

// NewDefaultSender constructs an XMPP logging backend that reads the
// hostname, username, and password from environment variables:
//
//   - GRIP_XMPP_HOSTNAME
//   - GRIP_XMPP_USERNAME
//   - GRIP_XMPP_PASSWORD
//
// Otherwise, the semantics of NewXMPPDefault are the same as NewXMPPLogger.
func NewDefaultSender(name, target string, l send.LevelInfo) (send.Sender, error) {
	info := GetConnectionInfo()

	return NewSender(name, target, info, l)
}

func constructXMPPLogger(name, target string, info ConnectionInfo) (send.Sender, error) {
	s := &xmppLogger{
		Base:   send.NewBase(name),
		target: target,
		info:   info,
	}

	if s.info.client == nil {
		s.info.client = &xmppClientImpl{}
	}

	if err := s.info.client.Create(info); err != nil {
		return nil, err
	}

	fallback := log.New(os.Stdout, "", log.LstdFlags)

	s.SetCloseHook(func() error { return s.info.client.Close() })
	s.SetErrorHandler(send.ErrorHandlerFromLogger(fallback))
	s.SetFormatter(send.MakeXMPPFormatter(s.Name()))
	s.SetResetHook(func() {
		s.SetFormatter(send.MakeXMPPFormatter(s.Name()))
		fallback.SetPrefix(fmt.Sprintf("[%s] ", s.Name()))
	})

	return s, nil
}

func (s *xmppLogger) Send(m message.Composer) {
	if s.Level().ShouldLog(m) {
		text, err := s.Formatter()(m)
		if err != nil {
			s.ErrorHandler()(err, m)
			return
		}

		c := xmpp.Chat{
			Remote: s.target,
			Type:   "chat",
			Text:   text,
		}

		if _, err := s.info.client.Send(c); err != nil {
			s.ErrorHandler()(err, m)
		}
	}
}

////////////////////////////////////////////////////////////////////////
//
// interface to wrap xmpp client interaction
//
////////////////////////////////////////////////////////////////////////

type xmppClient interface {
	Create(ConnectionInfo) error
	Send(xmpp.Chat) (int, error)
	Close() error
}

type xmppClientImpl struct {
	*xmpp.Client
}

func (c *xmppClientImpl) Create(info ConnectionInfo) error {
	opts := xmpp.Options{
		Host:     info.Hostname,
		User:     info.Username,
		Password: info.Password,
	}
	var err error
	c.Client, err = opts.NewClient()
	if err == nil {
		return nil
	}
	errs := []string{err.Error()}

	opts.NoTLS = true
	opts.InsecureAllowUnencryptedAuth = true

	c.Client, err = opts.NewClient()
	if err == nil {
		return nil
	}

	return fmt.Errorf("problem establishing connection to xmpp server: %s", strings.Join(append(errs, err.Error()), ";"))
}
