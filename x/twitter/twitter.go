package send

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/dghubble/go-twitter/twitter"
	"github.com/dghubble/oauth1"
	"github.com/tychoish/grip/level"
	"github.com/tychoish/grip/message"
	"github.com/tychoish/grip/send"
)

type twitterLogger struct {
	twitter twitterClient
	*send.Base
}

// TwitterOptions describes the credentials required to connect to the
// twitter API. While the name is used for internal reporting, the
// other values should be populated with credentials obtained from the
// Twitter API.
type TwitterOptions struct {
	Name           string
	ConsumerKey    string
	ConsumerSecret string
	AccessToken    string
	AccessSecret   string
}

func (opts *TwitterOptions) resolve(ctx context.Context) *twitter.Client {
	return twitter.NewClient(oauth1.NewConfig(opts.ConsumerKey, opts.ConsumerSecret).
		Client(ctx, oauth1.NewToken(opts.AccessToken, opts.AccessSecret)))
}

// MakeSender constructs a default sender implementation that
// posts messages to a twitter account. The implementation does not
// rate limit outgoing messages, which should be the responsibility of
// the caller.
func MakeSender(ctx context.Context, opts *TwitterOptions) (send.Sender, error) {
	return NewSender(ctx, opts, send.LevelInfo{Default: level.Trace, Threshold: level.Trace})
}

// NewSender constructs a sender implementation that posts
// messages to a twitter account, with configurable level
// information. The implementation does not rate limit outgoing
// messages, which should be the responsibility of the caller.
func NewSender(ctx context.Context, opts *TwitterOptions, l send.LevelInfo) (send.Sender, error) {
	s := &twitterLogger{
		twitter: newTwitterClient(ctx, opts),
		Base:    send.NewBase(opts.Name),
	}

	if err := s.SetLevel(l); err != nil {
		return nil, fmt.Errorf("invalid level specification: %w", err)
	}

	fallback := log.New(os.Stdout, "", log.LstdFlags)
	if err := s.SetErrorHandler(send.ErrorHandlerFromLogger(fallback)); err != nil {
		return nil, err
	}

	s.SetResetHook(func() {
		fallback.SetPrefix(fmt.Sprintf("[%s] ", s.Name()))
	})

	s.SetName(opts.Name)

	if err := s.twitter.Verify(); err != nil {
		return nil, fmt.Errorf("problem connecting to twitter: %w", err)
	}

	return s, nil
}

func (s *twitterLogger) Send(m message.Composer) {
	if s.Level().ShouldLog(m) {
		if err := s.twitter.Send(m.String()); err != nil {
			s.ErrorHandler()(err, m)
		}
	}
}

type twitterClient interface {
	Verify() error
	Send(string) error
}

type twitterClientImpl struct {
	twitter *twitter.Client
}

func newTwitterClient(ctx context.Context, opts *TwitterOptions) twitterClient {
	return &twitterClientImpl{twitter: opts.resolve(ctx)}
}

func (tc *twitterClientImpl) Verify() error {
	_, _, err := tc.twitter.Accounts.VerifyCredentials(&twitter.AccountVerifyParams{})
	return fmt.Errorf("could not verify account: %w", err)
}

func (tc *twitterClientImpl) Send(in string) error {
	_, _, err := tc.twitter.Statuses.Update(in, nil)
	return err
}
