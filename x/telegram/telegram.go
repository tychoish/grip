package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/tychoish/fun/erc"
	"github.com/tychoish/grip/message"
	"github.com/tychoish/grip/send"
)

type Options struct {
	BaseURL string
	Token   string
	ChatID  string
	Client  *http.Client
}

type sender struct {
	opts   Options
	url    string
	ctx    context.Context
	cancel context.CancelFunc
	send.Base
}

func (opts *Options) Validate() error {
	if opts.BaseURL == "" {
		opts.BaseURL = "https://api.telegram.org"
	}
	if opts.Client == nil {
		opts.Client = &http.Client{}
	}

	ec := &erc.Collector{}
	erc.When(ec, opts.Token == "", "must specify a token")
	erc.When(ec, opts.ChatID == "", "must specify a chatID")
	return ec.Resolve()
}

func New(opts Options) send.Sender {
	s := &sender{
		opts: opts,
		url:  fmt.Sprintf("%s/bot%s/sendMessage", opts.BaseURL, opts.Token),
	}
	s.ctx, s.cancel = context.WithCancel(context.Background())
	return s
}

type payload struct {
	ChatID string `json:"chat_id"`
	Text   string `json:"text"`
}

func (s *sender) Send(m message.Composer) {
	txt, err := s.Formatter()(m)
	if err != nil {
		s.ErrorHandler()(err, m)
		return
	}

	body, err := json.Marshal(payload{
		ChatID: s.opts.ChatID,
		Text:   txt,
	})

	if err != nil {
		s.ErrorHandler()(err, m)
		return
	}

	req, err := http.NewRequestWithContext(s.ctx, http.MethodPost, s.url, bytes.NewBuffer(body))
	if err != nil {
		s.ErrorHandler()(err, m)
		return
	}

	resp, err := s.opts.Client.Do(req)
	if err != nil {
		s.ErrorHandler()(err, m)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		ec := &erc.Collector{}
		ec.Add(fmt.Errorf("received response %s", resp.Status))
		out, err := io.ReadAll(resp.Body)
		ec.Add(err)
		ec.Add(fmt.Errorf("data: %q", string(out)))
		s.ErrorHandler()(ec.Resolve(), m)
		return
	}
}
