package github

import (
	"fmt"
	"net/url"

	"github.com/tychoish/grip/level"
	"github.com/tychoish/grip/message"
)

// State represents the 4 valid states for the Github State API in
// a safer way
type State string

// The list of valid states for Github Status API requests
const (
	StatePending = State("pending")
	StateSuccess = State("success")
	StateError   = State("error")
	StateFailure = State("failure")
)

// Status is a message to be posted to Github's Status API
type Status struct {
	Owner string `bson:"owner,omitempty" json:"owner,omitempty" yaml:"owner,omitempty"`
	Repo  string `bson:"repo,omitempty" json:"repo,omitempty" yaml:"repo,omitempty"`
	Ref   string `bson:"ref,omitempty" json:"ref,omitempty" yaml:"ref,omitempty"`

	Context     string `bson:"context" json:"context" yaml:"context"`
	State       State  `bson:"state" json:"state" yaml:"state"`
	URL         string `bson:"url" json:"url" yaml:"url"`
	Description string `bson:"description" json:"description" yaml:"description"`
}

// Valid returns true if the message is well formed
func (p *Status) Valid() bool {
	// owner, repo and ref must be empty or must be set
	ownerEmpty := len(p.Owner) == 0
	repoEmpty := len(p.Repo) == 0
	refLen := len(p.Ref) == 0
	if ownerEmpty != repoEmpty || repoEmpty != refLen {
		return false
	}

	switch p.State {
	case StatePending, StateSuccess, StateError, StateFailure:
	default:
		return false
	}

	_, err := url.Parse(p.URL)
	if err != nil || len(p.Context) == 0 {
		return false
	}

	return true
}

type githubStatusMessage struct {
	raw Status
	str string

	message.Base `bson:"metadata" json:"metadata" yaml:"metadata"`
}

// NewStatusMessageWithRepo creates a composer for sending payloads to the Github Status
// API, with the repository and ref stored in the composer
func NewStatusMessageWithRepo(p level.Priority, status Status) message.Composer {
	s := MakeStatusMessageWithRepo(status)
	_ = s.SetPriority(p)

	return s
}

// MakeStatusMessageWithRepo creates a composer for sending payloads to the Github Status
// API, with the repository and ref stored in the composer
func MakeStatusMessageWithRepo(status Status) message.Composer {
	return &githubStatusMessage{
		raw: status,
	}
}

// NewStatusMessage creates a composer for sending payloads to the Github Status
// API.
func NewStatusMessage(p level.Priority, context string, state State, URL, description string) message.Composer {
	s := MakeStatusMessage(context, state, URL, description)
	_ = s.SetPriority(p)

	return s
}

// MakeStatusMessage creates a composer for sending payloads to the Github Status
// API without setting a priority
func MakeStatusMessage(context string, state State, URL, description string) message.Composer {
	return &githubStatusMessage{
		raw: Status{
			Context:     context,
			State:       state,
			URL:         URL,
			Description: description,
		},
	}
}

func (c *githubStatusMessage) Loggable() bool { return c.raw.Valid() }
func (*githubStatusMessage) Structured() bool { return false }
func (c *githubStatusMessage) String() string {
	if len(c.str) != 0 {
		return c.str
	}

	base := c.raw.Ref
	if len(c.raw.Owner) > 0 {
		base = fmt.Sprintf("%s/%s@%s ", c.raw.Owner, c.raw.Repo, c.raw.Ref)
	}
	if len(c.raw.Description) == 0 {
		c.str = base + fmt.Sprintf("%s %s (%s)", c.raw.Context, string(c.raw.State), c.raw.URL)
	} else {
		c.str = base + fmt.Sprintf("%s %s: %s (%s)", c.raw.Context, string(c.raw.State), c.raw.Description, c.raw.URL)
	}

	return c.str
}

func (c *githubStatusMessage) Raw() any {
	return &c.raw
}
