package github

import (
	"context"
	"fmt"

	"github.com/google/go-github/github"
	"github.com/tychoish/grip"
	"github.com/tychoish/grip/message"
	"github.com/tychoish/grip/send"
	"golang.org/x/oauth2"
)

type githubLogger struct {
	opts *GithubOptions
	gh   githubClient
	send.Base
}

// GithubOptions contains information about a github account and
// repository, used in the GithubIssuesLogger and the
// GithubCommentLogger Sender implementations.
type GithubOptions struct {
	Account string
	Repo    string
	Token   string
}

// NewIssuesLogger builds a sender implementation that creates a
// new issue in a Github Project for each log message.
func NewIssueSender(name string, opts *GithubOptions) (send.Sender, error) {
	s := &githubLogger{
		opts: opts,
		gh:   &githubClientImpl{},
	}

	ctx := context.TODO()
	s.gh.Init(ctx, opts.Token)

	s.SetName(name)
	s.SetErrorHandler(send.ErrorHandlerFromSender(grip.Sender()))
	s.SetFormatter(send.MakeDefaultFormatter())

	return s, nil
}

func (s *githubLogger) Send(m message.Composer) {
	if send.ShouldLog(s, m) {
		text, err := s.Format(m)
		if !s.HandleErrorOK(send.WrapError(err, m)) {
			return
		}

		title := fmt.Sprintf("[%s]: %s", s.Name(), m.String())
		issue := &github.IssueRequest{
			Title: &title,
			Body:  &text,
		}

		ctx := context.TODO()
		if _, _, err := s.gh.Create(ctx, s.opts.Account, s.opts.Repo, issue); err != nil {
			s.HandleError(send.WrapError(err, m))
		}
	}
}

//////////////////////////////////////////////////////////////////////////
//
// interface wrapper for the github client so that we can mock things out
//
//////////////////////////////////////////////////////////////////////////

type githubClient interface {
	Init(context.Context, string)
	// Issues
	Create(context.Context, string, string, *github.IssueRequest) (*github.Issue, *github.Response, error)
	CreateComment(context.Context, string, string, int, *github.IssueComment) (*github.IssueComment, *github.Response, error)

	// Status API
	CreateStatus(ctx context.Context, owner, repo, ref string, status *github.RepoStatus) (*github.RepoStatus, *github.Response, error)
}

type githubClientImpl struct {
	*github.IssuesService
	repos *github.RepositoriesService
}

func (c *githubClientImpl) Init(ctx context.Context, token string) {
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})

	tc := oauth2.NewClient(ctx, ts)

	client := github.NewClient(tc)

	c.IssuesService = client.Issues
	c.repos = client.Repositories
}

func (c *githubClientImpl) CreateStatus(ctx context.Context, owner, repo, ref string, status *github.RepoStatus) (*github.RepoStatus, *github.Response, error) {
	return c.repos.CreateStatus(ctx, owner, repo, ref, status)
}
