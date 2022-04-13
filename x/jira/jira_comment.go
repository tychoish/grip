package jira

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/tychoish/grip/level"
	"github.com/tychoish/grip/message"
	"github.com/tychoish/grip/send"
)

type jiraCommentJournal struct {
	issueID string
	opts    *Options
	*send.Base
}

// MakeCommentSender is the same as NewJiraCommentLogger but uses a warning
// level of Trace
func MakeCommentSender(ctx context.Context, id string, opts *Options) (send.Sender, error) {
	return NewCommentSender(ctx, id, opts, send.LevelInfo{Default: level.Trace, Threshold: level.Trace})
}

// NewCommentSender constructs a Sender that creates issues to jira, given
// options defined in a JiraOptions struct. id parameter is the ID of the issue.
// ctx is used as the request context in the OAuth HTTP client
func NewCommentSender(ctx context.Context, id string, opts *Options, l send.LevelInfo) (send.Sender, error) {
	if err := opts.Validate(); err != nil {
		return nil, err
	}

	j := &jiraCommentJournal{
		opts:    opts,
		issueID: id,
		Base:    send.NewBase(id),
	}

	if err := j.opts.client.CreateClient(opts.HTTPClient, opts.BaseURL); err != nil {
		return nil, err
	}

	authOpts := jiraAuthOpts{
		username:           opts.BasicAuthOpts.Username,
		password:           opts.BasicAuthOpts.Password,
		addBasicAuthHeader: opts.BasicAuthOpts.UseBasicAuth,
		accessToken:        opts.Oauth1Opts.AccessToken,
		tokenSecret:        opts.Oauth1Opts.TokenSecret,
		privateKey:         opts.Oauth1Opts.PrivateKey,
		consumerKey:        opts.Oauth1Opts.ConsumerKey,
	}
	if err := j.opts.client.Authenticate(ctx, authOpts); err != nil {
		return nil, fmt.Errorf("jira authentication error: %v", err)
	}

	if err := j.SetLevel(l); err != nil {
		return nil, err
	}

	fallback := log.New(os.Stdout, "", log.LstdFlags)
	if err := j.SetErrorHandler(send.ErrorHandlerFromLogger(fallback)); err != nil {
		return nil, err
	}

	j.SetName(id)
	j.SetResetHook(func() { fallback.SetPrefix(fmt.Sprintf("[%s] ", j.Name())) })

	return j, nil
}

// Send post issues via jiraCommentJournal with information in the message.Composer
func (j *jiraCommentJournal) Send(m message.Composer) {
	if j.Level().ShouldLog(m) {
		issue := j.issueID
		if c, ok := m.Raw().(*Comment); ok {
			issue = c.IssueID
		}
		if err := j.opts.client.PostComment(issue, m.String()); err != nil {
			j.ErrorHandler()(err, m)
		}
	}
}
