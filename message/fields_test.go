package message

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tychoish/grip/level"
)

func TestFieldsLevelMutability(t *testing.T) {
	assert := assert.New(t) // nolint

	m := Fields{"message": "hello world"}
	c := ConvertWithPriority(level.Error, m)

	r := c.Raw().(Fields)
	assert.Equal(level.Error, c.Priority())
	assert.Equal(level.Error, r["metadata"].(*Base).Level)

	c = ConvertWithPriority(level.Info, m)
	r = c.Raw().(Fields)
	assert.Equal(level.Info, c.Priority())
	assert.Equal(level.Info, r["metadata"].(*Base).Level)
}
