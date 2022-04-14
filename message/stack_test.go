package message

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPrintStack(t *testing.T) {
	assert := assert.New(t)
	stack := funcA()
	assert.Contains(stack, `message/stack_test.go:26 (funcC)`)
	assert.Contains(stack, `message/stack_test.go:22 (funcB)`)
	assert.Contains(stack, `message/stack_test.go:18 (funcA)`)
}

func funcA() string {
	return funcB()
}

func funcB() string {
	return funcC()
}

func funcC() string {
	return MakeStack(0, "").String()
}

// don't add any code above this line unless you modify the line numbers in TestPrintStack
