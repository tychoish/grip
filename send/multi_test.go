package send

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tychoish/grip/level"
	"github.com/tychoish/grip/message"
)

func TestMultiSenderRespectsLevel(t *testing.T) {
	// this is a limited test to prevent level filtering to behave
	// differently than expected
	assert := assert.New(t) // nolint

	mock, err := NewInternalLogger("mock", LevelInfo{Default: level.Critical, Threshold: level.Alert})
	assert.NoError(err)
	s := MakeStdError()
	s.SetName("mock2")
	multi := MakeMulti(s, mock)

	assert.Equal(mock.Len(), 0)
	multi.Send(message.NewString(level.Info, "hello"))
	assert.Equal(mock.Len(), 1)
	m, ok := mock.GetMessageSafe()
	assert.True(ok)
	assert.False(m.Logged)

	multi.Send(message.NewString(level.Alert, "hello"))
	assert.Equal(mock.Len(), 1)
	m, ok = mock.GetMessageSafe()
	assert.True(ok)
	assert.True(m.Logged)

	multi.Send(message.NewString(level.Alert, "hello"))
	assert.Equal(mock.Len(), 1)
	m, ok = mock.GetMessageSafe()
	assert.True(ok)
	assert.True(m.Logged)

	assert.NoError(multi.Flush(context.TODO()))
}
