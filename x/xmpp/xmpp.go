package xmpp

import (
	"fmt"
	"os"

	"github.com/tychoish/grip"
	"github.com/tychoish/grip/message"
	"github.com/tychoish/grip/send"
	xmpp "github.com/xmppo/go-xmpp"
)

type xmppLogger struct {
	target string
	info   ConnectionInfo
	send.Base
}

// ConnectionInfo stores all information needed to connect to an
// XMPP (jabber) server to send log messages.
type ConnectionInfo struct {
	Hostname string
	Username string
	Password string

	DisableTLS           bool
	AllowUnencryptedAuth bool
	client               xmppClient
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

const completeFormatTmpl = "[%s] (p=%s) %s"

// MakeXMPPFormatter returns a MessageFormatter that will produce
// messages in the following format, used primarily by the xmpp logger:
//
//	[<name>] (p=<priority>) <message>
//
// It can never error.
func MakeXMPPFormatter(name string) send.MessageFormatter {
	return func(m message.Composer) (string, error) {
		return formatXmppMessage(name, m)
	}
}

func formatXmppMessage(name string, m message.Composer) (string, error) {
	return fmt.Sprintf(completeFormatTmpl, name, m.Priority(), m.String()), nil
}

// MakeSender constructs an XMPP logging backend that reads the
// hostname, username, and password from environment variables:
//
//   - GRIP_XMPP_HOSTNAME
//   - GRIP_XMPP_USERNAME
//   - GRIP_XMPP_PASSWORD
//
// The instance is otherwise unconfigured.
func MakeSender(target string) (send.Sender, error) { return NewSender(target, GetConnectionInfo()) }

// NewSender creates a sender with the configuration for the
// connection to the XMPP server.
func NewSender(target string, info ConnectionInfo) (send.Sender, error) {
	s := &xmppLogger{
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
	s.SetErrorHandler(send.ErrorHandlerFromSender(grip.Sender()))
	s.SetFormatter(func(m message.Composer) (string, error) {
		return formatXmppMessage(s.Name(), m)
	})

	return s, nil

}

func (s *xmppLogger) Send(m message.Composer) {
	if send.ShouldLog(s, m) {
		text, err := s.Format(m)
		if !s.HandleErrorOK(send.WrapError(err, m)) {
			return
		}

		c := xmpp.Chat{
			Remote: s.target,
			Type:   "chat",
			Text:   text,
		}

		if _, err := s.info.client.Send(c); err != nil {
			s.HandleError(send.WrapError(err, m))
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

		NoTLS:                        info.DisableTLS,
		InsecureAllowUnencryptedAuth: info.AllowUnencryptedAuth,
	}
	var err error
	c.Client, err = opts.NewClient()
	if err != nil {
		return err
	}

	return nil
}
