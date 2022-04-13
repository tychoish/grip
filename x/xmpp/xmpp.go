package send

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
	info   XMPPConnectionInfo
	*send.Base
}

// XMPPConnectionInfo stores all information needed to connect to an
// XMPP (jabber) server to send log messages.
type XMPPConnectionInfo struct {
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

// GetXMPPConnectionInfo builds an XMPPConnectionInfo structure
// reading default values from the following environment variables:
//
//    GRIP_XMPP_HOSTNAME
//    GRIP_XMPP_USERNAME
//    GRIP_XMPP_PASSWORD
func GetXMPPConnectionInfo() XMPPConnectionInfo {
	return XMPPConnectionInfo{
		Hostname: os.Getenv(xmppHostEnvVar),
		Username: os.Getenv(xmppUsernameEnvVar),
		Password: os.Getenv(xmppPasswordEnvVar),
	}
}

// NewXMPPSender constructs a new Sender implementation that sends
// messages to an XMPP user, "target", using the credentials specified in
// the XMPPConnectionInfo struct. The constructor will attempt to exablish
// a connection to the server via SSL, falling back automatically to an
// unencrypted connection if the the first attempt fails.
func NewXMPPSender(name, target string, info XMPPConnectionInfo, l send.LevelInfo) (send.Sender, error) {
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

// MakeXMPPSender constructs an XMPP logging backend that reads the
// hostname, username, and password from environment variables:
//
//    - GRIP_XMPP_HOSTNAME
//    - GRIP_XMPP_USERNAME
//    - GRIP_XMPP_PASSWORD
//
// The instance is otherwise unconquered. Call SetName or inject it
// into a Journaler instance using SetSender before using.
func MakeXMPPSender(target string) (send.Sender, error) {
	info := GetXMPPConnectionInfo()

	s, err := constructXMPPLogger("", target, info)
	if err != nil {
		return nil, err
	}

	return s, nil
}

// NewDefaultXMPPSender constructs an XMPP logging backend that reads the
// hostname, username, and password from environment variables:
//
//    - GRIP_XMPP_HOSTNAME
//    - GRIP_XMPP_USERNAME
//    - GRIP_XMPP_PASSWORD
//
// Otherwise, the semantics of NewXMPPDefault are the same as NewXMPPLogger.
func NewDefaultXMPPSender(name, target string, l send.LevelInfo) (send.Sender, error) {
	info := GetXMPPConnectionInfo()

	return NewXMPPSender(name, target, info, l)
}

func constructXMPPLogger(name, target string, info XMPPConnectionInfo) (send.Sender, error) {
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

	s.SetCloseHook(func() error { return s.info.client.Close() })

	fallback := log.New(os.Stdout, "", log.LstdFlags)
	if err := s.SetErrorHandler(send.ErrorHandlerFromLogger(fallback)); err != nil {
		return nil, err
	}

	if err := s.SetFormatter(send.MakeXMPPFormatter(s.Name())); err != nil {
		return nil, err
	}

	s.SetResetHook(func() {
		_ = s.SetFormatter(send.MakeXMPPFormatter(s.Name()))
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
	Create(XMPPConnectionInfo) error
	Send(xmpp.Chat) (int, error)
	Close() error
}

type xmppClientImpl struct {
	*xmpp.Client
}

func (c *xmppClientImpl) Create(info XMPPConnectionInfo) error {
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
