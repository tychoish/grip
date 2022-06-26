package message

import (
	"strings"
	"testing"
)

func TestPrintStack(t *testing.T) {
	stack := funcA()
	const (
		strA = `message/stack_test.go:26 (funcC)`
		strB = `message/stack_test.go:22 (funcB)`
		strC = `message/stack_test.go:18 (funcA)`
	)
	if !strings.Contains(stack, strA) {
		t.Errorf("%q should contain %q", stack, strA)
	}
	if !strings.Contains(stack, strB) {
		t.Errorf("%q should contain %q", stack, strB)
	}
	if !strings.Contains(stack, strC) {
		t.Errorf("%q should contain %q", stack, strC)
	}
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
