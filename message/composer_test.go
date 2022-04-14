package message

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tychoish/grip/level"
)

func TestPopulatedMessageComposerConstructors(t *testing.T) {
	const testMsg = "hello"
	assert := assert.New(t)
	// map objects to output
	cases := map[Composer]string{
		MakeString(testMsg):                                          testMsg,
		NewString(level.Error, testMsg):                              testMsg,
		MakeBytes([]byte(testMsg)):                                   testMsg,
		NewBytes(level.Error, []byte(testMsg)):                       testMsg,
		MakeError(errors.New(testMsg)):                               testMsg,
		NewError(level.Error, errors.New(testMsg)):                   testMsg,
		NewErrorWrap(errors.New(testMsg), ""):                        testMsg,
		NewErrorWrapMessage(level.Error, errors.New(testMsg), ""):    testMsg,
		MakeFormat(string(testMsg[0])+"%s", testMsg[1:]):             testMsg,
		NewFormat(level.Error, string(testMsg[0])+"%s", testMsg[1:]): testMsg,
		WrapError(errors.New(testMsg), ""):                           testMsg,
		WrapErrorf(errors.New(testMsg), ""):                          testMsg,
		MakeLines(testMsg, ""):                                       testMsg,
		NewLines(level.Error, testMsg, ""):                           testMsg,
		MakeLines(testMsg):                                           testMsg,
		NewLines(level.Error, testMsg):                               testMsg,
		BuildGroupComposer(MakeString(testMsg)):                      testMsg,
		MakeGroupComposer([]Composer{MakeString(testMsg)}):           testMsg,
		// MakeJiraMessage(&JiraIssue{Summary: testMsg, Type: "Something"}):                       testMsg,
		// NewJiraMessage("", testMsg, JiraField{Key: "type", Value: "Something"}):                testMsg,
		NewAnnotated(level.Error, testMsg, Fields{}):                                    fmt.Sprintf("[message='%s']", testMsg),
		NewFields(level.Error, Fields{"test": testMsg}):                                 fmt.Sprintf("[test='%s']", testMsg),
		MakeAnnotated(testMsg, Fields{}):                                                fmt.Sprintf("[message='%s']", testMsg),
		MakeFields(Fields{"test": testMsg}):                                             fmt.Sprintf("[test='%s']", testMsg),
		NewErrorWrappedComposer(errors.New("hello"), MakeString("world")):               "world: hello",
		When(true, testMsg):                                                             testMsg,
		Whenf(true, testMsg):                                                            testMsg,
		Whenln(true, testMsg):                                                           testMsg,
		Whenln(true, testMsg):                                                           testMsg,
		MakeProducer(func() Composer { return MakeString(testMsg) }):                    testMsg,
		NewProducer(level.Error, func() Composer { return MakeString(testMsg) }):        testMsg,
		MakeErrorProducer(func() error { return errors.New(testMsg) }):                  testMsg,
		NewErrorProducer(level.Error, func() error { return errors.New(testMsg) }):      testMsg,
		NewFieldsProducer(level.Error, func() Fields { return Fields{"pro": "ducer"} }): "[pro='ducer']",
		NewConvertedFieldsProducer(level.Error, func() map[string]interface{} { return map[string]interface{}{"pro": "ducer"} }): "[pro='ducer']",
		// NewEmailMessage(level.Error, Email{
		// 	Recipients: []string{"someone@example.com"},
		// 	Subject:    "Test msg",
		// 	Body:       testMsg,
		// }): fmt.Sprintf("To: someone@example.com; Body: %s", testMsg),
		// NewGithubStatusMessage(level.Error, "tests", GithubStateError, "https://example.com", testMsg): fmt.Sprintf("tests error: %s (https://example.com)", testMsg),
		// NewGithubStatusMessageWithRepo(level.Error, GithubStatus{
		// 	Owner:       "tychoish",
		// 	Repo:        "grip",
		// 	Ref:         "master",
		// 	Context:     "tests",
		// 	State:       GithubStateError,
		// 	URL:         "https://example.com",
		// 	Description: testMsg,
		// }): fmt.Sprintf("tychoish/grip@master tests error: %s (https://example.com)", testMsg),
		// NewJIRACommentMessage(level.Error, "ABC-123", testMsg): testMsg,
		// NewSlackMessage(level.Error, "@someone", testMsg, nil): fmt.Sprintf("@someone: %s", testMsg),
	}

	for msg, output := range cases {
		assert.NotNil(msg)
		assert.NotEmpty(output)
		assert.Implements((*Composer)(nil), msg)
		assert.True(msg.Loggable())
		assert.NotNil(msg.Raw())

		if strings.HasPrefix(output, "[") {
			output = strings.Trim(output, "[]")
			assert.True(strings.Contains(msg.String(), output), fmt.Sprintf("%T: %s (%s)", msg, msg.String(), output))

		} else {
			// run the string test to make sure it doesn't change:
			assert.Equal(msg.String(), output, "%T", msg)
			assert.Equal(msg.String(), output, "%T", msg)
		}

		if msg.Priority() != level.Invalid {
			assert.Equal(msg.Priority(), level.Error)
		}

		// check message annotation functionality
		switch msg.(type) {
		case *GroupComposer:
			continue
		default:
			assert.NoError(msg.Annotate("k1", "foo"), "%T", msg)
			assert.Error(msg.Annotate("k1", "foo"), "%T", msg)
			assert.NoError(msg.Annotate("k2", "foo"), "%T", msg)
		}
	}
}

func TestUnpopulatedMessageComposers(t *testing.T) {
	assert := assert.New(t) // nolint
	// map objects to output
	cases := []Composer{
		&stringMessage{},
		MakeString(""),
		NewString(level.Error, ""),
		&bytesMessage{},
		MakeBytes([]byte{}),
		NewBytes(level.Error, []byte{}),
		// &ProcessInfo{},
		// &SystemInfo{},
		&lineMessenger{},
		MakeLines(),
		NewLines(level.Error),
		&formatMessenger{},
		MakeFormat(""),
		NewFormat(level.Error, ""),
		MakeStack(1, ""),
		BuildGroupComposer(),
		&GroupComposer{},
		// &GoRuntimeInfo{},
		When(false, ""),
		Whenf(false, "", ""),
		Whenln(false, "", ""),
		// NewEmailMessage(level.Error, Email{}),
		// NewGithubStatusMessage(level.Error, "", GithubState(""), "", ""),
		// NewGithubStatusMessageWithRepo(level.Error, GithubStatus{}),
		// NewJIRACommentMessage(level.Error, "", ""),
		// NewSlackMessage(level.Error, "", "", nil),
		MakeProducer(nil),
		MakeProducer(func() Composer { return nil }),
		MakeFieldsProducer(nil),
		MakeFieldsProducer(func() Fields { return nil }),
		MakeFieldsProducer(func() Fields { return Fields{} }),
		MakeErrorProducer(nil),
		MakeErrorProducer(func() error { return nil }),
	}

	for idx, msg := range cases {
		assert.False(msg.Loggable(), "%d:%T", idx, msg)
	}
}

func TestStackMessages(t *testing.T) {
	const testMsg = "hello"
	var stackMsg = "message/composer_test"

	assert := assert.New(t) // nolint
	// map objects to output (prefix)
	cases := map[Composer]string{
		MakeStack(1, testMsg): testMsg,

		// with 0 frame
		MakeStack(0, testMsg): testMsg,
	}

	for msg, text := range cases {
		assert.NotNil(msg)
		assert.Implements((*Composer)(nil), msg)
		assert.NotNil(msg.Raw())
		if text != "" {
			assert.True(msg.Loggable())
		}

		diagMsg := fmt.Sprintf("%T: %+v", msg, msg)
		assert.True(strings.Contains(msg.String(), text), diagMsg)
		assert.True(strings.Contains(msg.String(), stackMsg), "%s\n%s\n%s\n", diagMsg, msg.String(), stackMsg)
	}
}

func TestComposerConverter(t *testing.T) {
	const testMsg = "hello world"
	assert := assert.New(t) // nolint

	cases := []interface{}{
		MakeLines(testMsg),
		testMsg,
		errors.New(testMsg),
		[]string{testMsg},
		[]interface{}{testMsg},
		[]byte(testMsg),
		[]Composer{MakeString(testMsg)},
	}

	for _, msg := range cases {
		comp := ConvertWithPriority(level.Error, msg)
		assert.True(comp.Loggable())
		assert.Equal(testMsg, comp.String(), "%T", msg)
	}

	cases = []interface{}{
		nil,
		"",
		[]interface{}{},
		[]string{},
		[]byte{},
		Fields{},
		map[string]interface{}{},
	}

	for _, msg := range cases {
		comp := ConvertWithPriority(level.Error, msg)
		assert.False(comp.Loggable())
		assert.Equal("", comp.String(), "%T", msg)
	}

	outputCases := map[string]interface{}{
		"1":            1,
		"2":            int32(2),
		"[message='3'": Fields{"message": 3},
		"[message='4'": map[string]interface{}{"message": "4"},
	}

	for out, in := range outputCases {
		comp := ConvertWithPriority(level.Error, in)
		assert.True(comp.Loggable())
		assert.True(strings.HasPrefix(comp.String(), out))
	}

}

type causer interface {
	Cause() error
}

type unwrapper interface {
	Unwrap() error
}

func TestErrors(t *testing.T) {
	for name, cmp := range map[string]Composer{
		"Wrapped":         WrapError(errors.New("err"), "wrap"),
		"Plain":           MakeError(errors.New("err")),
		"Producer":        MakeErrorProducer(func() error { return errors.New("message") }),
		"WrapperProducer": WrapErrorFunc(func() error { return errors.New("message") }, Fields{"op": "wrap"}),
	} {
		t.Run(name, func(t *testing.T) {
			t.Run("Interfaces", func(t *testing.T) {
				assert.Implements(t, (*error)(nil), cmp)
				assert.Implements(t, (*error)(nil), cmp)
				assert.Implements(t, (*unwrapper)(nil), cmp)
			})
			t.Run("Value", func(t *testing.T) {
				assert.Equal(t, cmp.(error).Error(), cmp.String())
			})
			t.Run("Causer", func(t *testing.T) {
				cause := unwrapCause(cmp.(error))
				assert.NotEqual(t, cause, cmp)
			})
			t.Run("ExtendedFormat", func(t *testing.T) {
				assert.NotEqual(t, fmt.Sprintf("%+v", cmp), fmt.Sprintf("%v", cmp))
			})

		})
	}
}
