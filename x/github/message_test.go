package github

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tychoish/grip/level"
)

func TestStatus(t *testing.T) {
	assert := assert.New(t) //nolint: vetshadow

	c := NewStatusMessage(level.Info, "example", StatePending, "https://example.com/hi", "description")
	assert.NotNil(c)
	assert.True(c.Loggable())

	raw, ok := c.Raw().(*Status)
	assert.True(ok)

	assert.NotPanics(func() {
		assert.Equal("example", raw.Context)
		assert.Equal(StatePending, raw.State)
		assert.Equal("https://example.com/hi", raw.URL)
		assert.Equal("description", raw.Description)
	})

	assert.Equal("example pending: description (https://example.com/hi)", c.String())
}

func TestStatusInvalidStatusesAreNotLoggable(t *testing.T) {
	assert := assert.New(t) //nolint: vetshadow

	c := NewStatusMessage(level.Info, "", StatePending, "https://example.com/hi", "description")
	assert.False(c.Loggable())
	c = NewStatusMessage(level.Info, "example", "nope", "https://example.com/hi", "description")
	assert.False(c.Loggable())
	c = NewStatusMessage(level.Info, "example", StatePending, ":foo", "description")
	assert.False(c.Loggable())

	p := Status{
		Owner:       "",
		Repo:        "grip",
		Ref:         "master",
		Context:     "example",
		State:       StatePending,
		URL:         "https://example.com/hi",
		Description: "description",
	}
	c = NewStatusMessageWithRepo(level.Info, p)
	assert.False(c.Loggable())

	p.Owner = "tychoish"
	p.Repo = ""
	c = NewStatusMessageWithRepo(level.Info, p)
	assert.False(c.Loggable())

	p.Repo = "grip"
	p.Ref = ""
	c = NewStatusMessageWithRepo(level.Info, p)
	assert.False(c.Loggable())
}
