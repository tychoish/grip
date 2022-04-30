package splunk

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"time"

	hec "github.com/fuyufjh/splunk-hec-go"
	"github.com/tychoish/grip/level"
	"github.com/tychoish/grip/message"
	"github.com/tychoish/grip/send"
)

const (
	splunkServerURL   = "GRIP_SPLUNK_SERVER_URL"
	splunkClientToken = "GRIP_SPLUNK_CLIENT_TOKEN"
	splunkChannel     = "GRIP_SPLUNK_CHANNEL"
)

type splunkLogger struct {
	info     ConnectionInfo
	client   splunkClient
	hostname string
	*send.Base
}

// ConnectionInfo stores all information needed to connect
// to a splunk server to send log messsages.
type ConnectionInfo struct {
	ServerURL string `bson:"url" json:"url" yaml:"url"`
	Token     string `bson:"token" json:"token" yaml:"token"`
	Channel   string `bson:"channel" json:"channel" yaml:"channel"`
}

// GetConnectionInfo builds a SplunkConnectionInfo structure
// reading default values from the following environment variables:
//
//		GRIP_SPLUNK_SERVER_URL
//		GRIP_SPLUNK_CLIENT_TOKEN
//		GRIP_SPLUNK_CHANNEL
func GetConnectionInfo() ConnectionInfo {
	return ConnectionInfo{
		ServerURL: os.Getenv(splunkServerURL),
		Token:     os.Getenv(splunkClientToken),
		Channel:   os.Getenv(splunkChannel),
	}
}

// Populated validates a SplunkConnectionInfo, and returns false if
// there is missing data.
func (info ConnectionInfo) Populated() bool {
	return info.ServerURL != "" && info.Token != ""
}

func (info ConnectionInfo) validateFromEnv() error {
	if info.ServerURL == "" {
		return fmt.Errorf("environment variable %s not defined, cannot create splunk client", splunkServerURL)
	}
	if info.Token == "" {
		return fmt.Errorf("environment variable %s not defined, cannot create splunk client", splunkClientToken)
	}
	return nil
}

func (s *splunkLogger) Send(m message.Composer) {
	lvl := s.Level()

	if lvl.ShouldLog(m) {
		g, ok := m.(*message.GroupComposer)
		if ok {
			batch := []*hec.Event{}
			for _, c := range g.Messages() {
				if lvl.ShouldLog(c) {
					e := hec.NewEvent(c.Raw())
					e.SetHost(s.hostname)
					batch = append(batch, e)
				}
			}
			if err := s.client.WriteBatch(batch); err != nil {
				s.ErrorHandler()(err, m)
			}
			return
		}

		e := hec.NewEvent(m.Raw())
		e.SetHost(s.hostname)
		if err := s.client.WriteEvent(e); err != nil {
			s.ErrorHandler()(err, m)
		}
	}
}

// NewSender constructs a new Sender implementation that sends
// messages to a Splunk event collector using the credentials specified
// in the SplunkConnectionInfo struct.
func NewSender(name string, info ConnectionInfo, l send.LevelInfo) (send.Sender, error) {
	client := (&http.Client{
		Transport: &http.Transport{
			Proxy:               http.ProxyFromEnvironment,
			DisableKeepAlives:   true,
			TLSHandshakeTimeout: 5 * time.Second,
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
		Timeout: 5 * time.Second,
	})

	s, err := buildSplunkLogger(name, client, info, l)
	if err != nil {
		return nil, err
	}

	if err := s.client.Create(client, info); err != nil {
		return nil, err
	}

	return s, nil
}

// NewWithClient makes it possible to pass an existing
// http.Client to the splunk instance, but is otherwise identical to
// NewSplunkLogger.
func NewWithClient(name string, info ConnectionInfo, l send.LevelInfo, client *http.Client) (send.Sender, error) {
	s, err := buildSplunkLogger(name, client, info, l)
	if err != nil {
		return nil, err
	}

	if err := s.client.Create(client, info); err != nil {
		return nil, err
	}

	return s, nil
}

func buildSplunkLogger(name string, client *http.Client, info ConnectionInfo, l send.LevelInfo) (*splunkLogger, error) {
	s := &splunkLogger{
		info:   info,
		client: &splunkClientImpl{},
		Base:   send.NewBase(name),
	}

	hostname, err := os.Hostname()
	if err != nil {
		return nil, err
	}
	s.hostname = hostname

	if err := s.SetLevel(l); err != nil {
		return nil, err
	}
	return s, nil
}

// MakeSender constructs a new Sender implementation that reads
// the hostname, username, and password from environment variables:
//
//		GRIP_SPLUNK_SERVER_URL
//		GRIP_SPLUNK_CLIENT_TOKEN
//		GRIP_SPLUNK_CLIENT_CHANNEL
func MakeSender(name string) (send.Sender, error) {
	info := GetConnectionInfo()
	if err := info.validateFromEnv(); err != nil {
		return nil, err
	}

	return NewSender(name, info, send.LevelInfo{Default: level.Trace, Threshold: level.Trace})
}

// MakeWithClient is identical to MakeSplunkLogger but
// allows you to pass in a http.Client.
func MakeWithClient(name string, client *http.Client) (send.Sender, error) {
	info := GetConnectionInfo()
	if err := info.validateFromEnv(); err != nil {
		return nil, err
	}

	return NewWithClient(name, info, send.LevelInfo{Default: level.Trace, Threshold: level.Trace}, client)
}

////////////////////////////////////////////////////////////////////////
//
// interface wrapper for the splunk client so that we can mock things out
//
////////////////////////////////////////////////////////////////////////

type splunkClient interface {
	Create(*http.Client, ConnectionInfo) error
	WriteEvent(*hec.Event) error
	WriteBatch([]*hec.Event) error
}

type splunkClientImpl struct {
	hec.HEC
}

func (c *splunkClientImpl) Create(client *http.Client, info ConnectionInfo) error {
	c.HEC = hec.NewClient(info.ServerURL, info.Token)
	if info.Channel != "" {
		c.HEC.SetChannel(info.Channel)
	}

	c.HEC.SetKeepAlive(false)
	c.HEC.SetMaxRetry(2)
	c.HEC.SetHTTPClient(client)

	return nil
}
