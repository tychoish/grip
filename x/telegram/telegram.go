package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/tychoish/fun/erc"
	"github.com/tychoish/fun/ers"
	"github.com/tychoish/grip/message"
	"github.com/tychoish/grip/send"
)

// Options are used to configure a telegram sender.
type Options struct {
	Name   string `bson:"name" json:"name" yaml:"name"`
	Token  string `bson:"token" json:"token" yaml:"token"`
	Target string `bson:"target" json:"target" yaml:"target"`

	// Optional: BaseURL defaults to https://api.telegram.org, and
	// a new [unconfigured] HTTP client is constructed if one is
	// not provided.
	BaseURL string       `bson:"base_url" json:"base_url" yaml:"base_url"`
	Client  *http.Client `bson:"-" json:"-" yaml:"-"`
}

type sender struct {
	opts   Options
	url    string
	ctx    context.Context
	cancel context.CancelFunc
	send.Base
}

func (opts Options) IsZero() bool {
	return opts.Client == nil && opts.BaseURL == "" && opts.Token == "" && opts.Target == ""
}

func (opts *Options) Validate() error {
	if opts.BaseURL == "" {
		opts.BaseURL = "https://api.telegram.org"
	}
	if opts.Client == nil {
		opts.Client = &http.Client{}
	}

	ec := &erc.Collector{}
	ec.If(opts.Token == "", ers.New("must specify a token"))
	ec.If(opts.Target == "", ers.New("must specify a target or chatID"))
	return ec.Resolve()
}

// New constructs a telegram sender. The implementation posts each
// message independently. Use the buffered send.Sender implementation
// to batch these messages.
func New(opts Options) send.Sender {
	s := &sender{
		opts: opts,
		url:  fmt.Sprintf("%s/bot%s/sendMessage", opts.BaseURL, opts.Token),
	}
	s.SetName(opts.Name)

	s.ctx, s.cancel = context.WithCancel(context.Background())

	return s
}

type payload struct {
	ChatID string `json:"chat_id"`
	Text   string `json:"text"`
}

func (s *sender) Send(m message.Composer) {
	if !send.ShouldLog(s, m) {
		return
	}

	txt, err := s.Format(m)
	if !s.HandleErrorOK(send.WrapError(err, m)) {
		return
	}

	body, err := json.Marshal(payload{
		ChatID: s.opts.Target,
		Text:   txt,
	})

	if !s.HandleErrorOK(send.WrapError(err, m)) {
		return
	}

	req, err := http.NewRequestWithContext(s.ctx, http.MethodPost, s.url, bytes.NewBuffer(body))
	if !s.HandleErrorOK(send.WrapError(err, m)) {
		return
	}
	req.Header.Set("content-type", "application/json")

	resp, err := s.opts.Client.Do(req)
	if !s.HandleErrorOK(send.WrapError(err, m)) {
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		ec := &erc.Collector{}
		ec.Push(fmt.Errorf("received response %s", resp.Status))

		out, err := io.ReadAll(resp.Body)
		ec.Push(err)
		ec.Push(fmt.Errorf("data: %q", string(out)))
		if !s.HandleErrorOK(send.WrapError(ec.Resolve(), m)) {
			return
		}
	}
}
