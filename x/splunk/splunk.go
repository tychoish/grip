package splunk

import (
	"crypto/tls"
	"net/http"
	"os"
	"time"

	hec "github.com/fuyufjh/splunk-hec-go"
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
	send.Base
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
//	GRIP_SPLUNK_SERVER_URL
//	GRIP_SPLUNK_CLIENT_TOKEN
//	GRIP_SPLUNK_CHANNEL
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

func (s *splunkLogger) Send(m message.Composer) {
	if send.ShouldLog(s, m) {

		switch msgs := message.Unwind(m); len(msgs) {
		case 0:
			return
		case 1:
			e := hec.NewEvent(m.Raw())
			e.SetHost(s.hostname)
			if err := s.client.WriteEvent(e); err != nil {
				s.HandleError(send.WrapError(err, m))
			}
		default:
			batch := []*hec.Event{}
			for _, c := range message.Unwind(m) {
				if send.ShouldLog(s, c) {
					e := hec.NewEvent(c.Raw())
					e.SetHost(s.hostname)
					batch = append(batch, e)
				}
			}

			if err := s.client.WriteBatch(batch); err != nil {
				s.HandleError(send.WrapError(err, m))
			}
			return
		}
	}
}

// MakeSender constructs a new Sender implementation that sends
// messages to a Splunk event collector using the credentials specified
// in the SplunkConnectionInfo struct.
func MakeSender(info ConnectionInfo) (send.Sender, error) {
	client := &http.Client{
		Transport: &http.Transport{
			Proxy:               http.ProxyFromEnvironment,
			DisableKeepAlives:   true,
			TLSHandshakeTimeout: 5 * time.Second,
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
		Timeout: 5 * time.Second,
	}
	return MakeSenderWithClient(info, client)
}

// MakeSenderWithClient makes it possible to pass an existing
// http.Client to the splunk instance, but is otherwise identical to
// MakeSender.
func MakeSenderWithClient(info ConnectionInfo, client *http.Client) (send.Sender, error) {
	s := &splunkLogger{info: info, client: &splunkClientImpl{}}
	if err := s.client.Create(client, info); err != nil {
		return nil, err
	}

	hostname, err := os.Hostname()
	if err != nil {
		return nil, err
	}
	s.hostname = hostname

	return s, nil
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
