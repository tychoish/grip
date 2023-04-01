package github

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/google/go-github/github"
	"github.com/tychoish/grip/message"
	"github.com/tychoish/grip/send"
)

type githubStatusMessageLogger struct {
	opts *GithubOptions
	ref  string

	gh githubClient
	*send.Base
}

func (s *githubStatusMessageLogger) Send(m message.Composer) {
	if send.ShouldLog(s, m) {
		var status *github.RepoStatus
		owner := ""
		repo := ""
		ref := ""

		switch v := m.Raw().(type) {
		case *Status:
			status = githubStatusMessagePayloadToRepoStatus(v)
			if v != nil {
				owner = v.Owner
				repo = v.Repo
				ref = v.Ref
			}
		case Status:
			status = githubStatusMessagePayloadToRepoStatus(&v)
			owner = v.Owner
			repo = v.Repo
			ref = v.Ref

		case *message.Fields:
			status = s.githubMessageFieldsToStatus(v)
			owner, repo, ref = githubMessageFieldsToRepo(v)
		case message.Fields:
			status = s.githubMessageFieldsToStatus(&v)
			owner, repo, ref = githubMessageFieldsToRepo(&v)
		}
		if len(owner) == 0 {
			owner = s.opts.Account
		}
		if len(repo) == 0 {
			owner = s.opts.Repo
		}
		if len(ref) == 0 {
			owner = s.ref
		}
		if status == nil {
			s.ErrorHandler()(errors.New("composer cannot be converted to github status"), m)
			return
		}

		_, _, err := s.gh.CreateStatus(context.TODO(), owner, repo, ref, status)
		if err != nil {
			s.ErrorHandler()(err, m)
		}
	}
}

// NewStatusSender returns a Sender to send payloads to the Github Status
// API. Statuses will be attached to the given ref.
func NewStatusSender(name string, opts *GithubOptions, ref string) send.Sender {
	s := &githubStatusMessageLogger{
		Base: send.NewBase(name),
		gh:   &githubClientImpl{},
		ref:  ref,
	}

	ctx := context.TODO()
	s.gh.Init(ctx, opts.Token)

	fallback := log.New(os.Stdout, "", log.LstdFlags)

	s.SetName(name)
	s.SetFormatter(send.MakePlainFormatter())
	s.SetErrorHandler(send.ErrorHandlerFromLogger(fallback))
	s.SetResetHook(func() {
		fallback.SetPrefix(fmt.Sprintf("[%s] [%s/%s] ", s.Name(), opts.Account, opts.Repo))
	})

	return s
}

func (s *githubStatusMessageLogger) githubMessageFieldsToStatus(m *message.Fields) *github.RepoStatus {
	if m == nil {
		return nil
	}

	state, ok := getStringPtrFromField((*m)["state"])
	if !ok {
		return nil
	}
	context, ok := getStringPtrFromField((*m)["context"])
	if !ok {
		return nil
	}
	URL, ok := getStringPtrFromField((*m)["URL"])
	if !ok {
		return nil
	}
	var description *string
	if description != nil {
		description, ok = getStringPtrFromField((*m)["description"])
		if description != nil && len(*description) == 0 {
			description = nil
		}
		if !ok {
			return nil
		}
	}

	status := &github.RepoStatus{
		State:       state,
		Context:     context,
		TargetURL:   URL,
		Description: description,
	}

	return status
}

func getStringPtrFromField(i any) (*string, bool) {
	if ret, ok := i.(string); ok {
		return &ret, true
	}
	if ret, ok := i.(*string); ok {
		return ret, ok
	}

	return nil, false
}
func githubStatusMessagePayloadToRepoStatus(c *Status) *github.RepoStatus {
	if c == nil {
		return nil
	}

	s := &github.RepoStatus{
		Context: github.String(c.Context),
		State:   github.String(string(c.State)),
	}
	if len(c.URL) > 0 {
		s.TargetURL = github.String(c.URL)
	}
	if len(c.Description) > 0 {
		s.Description = github.String(c.Description)
	}

	return s
}

func githubMessageFieldsToRepo(m *message.Fields) (string, string, string) {
	if m == nil {
		return "", "", ""
	}

	owner, ok := getStringPtrFromField((*m)["owner"])
	if !ok {
		owner = github.String("")
	}
	repo, ok := getStringPtrFromField((*m)["repo"])
	if !ok {
		repo = github.String("")
	}
	ref, ok := getStringPtrFromField((*m)["ref"])
	if !ok {
		ref = github.String("")
	}

	return *owner, *repo, *ref
}
