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
		NewAnnotatedSimple(level.Error, testMsg, Fields{}):                              fmt.Sprintf("[message='%s']", testMsg),
		MakeAnnotatedSimple(testMsg, Fields{}):                                          fmt.Sprintf("[message='%s']", testMsg),
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
		if msg == nil {
			t.Error("value should not be nill")
		}

		if _, ok := msg.(Composer); !ok {
			t.Errorf("message %T should implement composer", msg)
		}
		if !msg.Loggable() {
			t.Error("value should be true")
		}
		if msg.Raw() == nil {
			t.Error("value should not be nill")
		}

		if strings.HasPrefix(output, "[") {
			output = strings.Trim(output, "[]")
			if !strings.Contains(msg.String(), output) {
				t.Logf("%T: %s (%s)", msg, msg.String(), output)
				t.Error("value should be true")
			}

		} else {
			// run the string test to make sure it doesn't change:
			if msg.String() != output {
				t.Errorf("%T", msg)
			}
			if msg.String() != output {
				t.Errorf("%T", msg)
			}
		}

		if msg.Priority() != level.Invalid {
			if level.Error != msg.Priority() {
				t.Error("elements should be equal")
			}
			if err := msg.SetPriority(msg.Priority()); err != nil {
				t.Error(err)
			}
		}

		// check message annotation functionality
		switch msg.(type) {
		case *GroupComposer:
			continue
		default:
			if err := msg.Annotate("k1", "foo"); err != nil {
				t.Error(err)
			}
			if err := msg.Annotate("k1", "foo"); err == nil {
				t.Error("annotation should be an error")
			}
			if err := msg.Annotate("k2", "foo"); err != nil {
				t.Error(err)
			}
		}
	}
}

func TestUnpopulatedMessageComposers(t *testing.T) {
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
		if msg.Loggable() {
			t.Errorf("message %T at %d should not be loggable", msg, idx)
		}
	}
}

func TestStackMessages(t *testing.T) {
	const testMsg = "hello"
	var stackMsg = "message/composer_test"

	// map objects to output (prefix)
	cases := map[Composer]string{
		MakeStack(1, testMsg): testMsg,

		// with 0 frame
		MakeStack(0, testMsg): testMsg,
	}

	for msg, text := range cases {
		if msg == nil {
			t.Error("value should not be nill")
		}
		if _, ok := msg.(Composer); !ok {
			t.Errorf("message %T should implement composer", msg)
		}
		if msg.Raw() == nil {
			t.Error("value should not be nill")
		}
		if text != "" {
			if !msg.Loggable() {
				t.Error("value should be true")
			}
		}

		diagMsg := fmt.Sprintf("%T: %+v", msg, msg)
		if !strings.Contains(msg.String(), text) {
			t.Log(diagMsg)
			t.Error("value should be true")
		}
		if !strings.Contains(msg.String(), stackMsg) {
			t.Logf("%s\n%s\n%s\n", diagMsg, msg.String(), stackMsg)
			t.Error("value should be true")
		}
	}
}

func TestComposerConverter(t *testing.T) {
	const testMsg = "hello world"

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
		if !comp.Loggable() {
			t.Error("value should be true")
		}
		if testMsg != comp.String() {
			t.Errorf("%T", msg)
		}
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
		if comp.Loggable() {
			t.Error("should be false")
		}
		if "" != comp.String() {
			t.Errorf("%T", msg)
		}
	}

	outputCases := map[string]interface{}{
		"1":            1,
		"2":            int32(2),
		"[message='3'": Fields{"message": 3},
		"[message='4'": map[string]interface{}{"message": "4"},
	}

	for out, in := range outputCases {
		comp := ConvertWithPriority(level.Error, in)
		if !comp.Loggable() {
			t.Error("value should be true")
		}
		if !strings.HasPrefix(comp.String(), out) {
			t.Error("value should be true")
		}
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
				if _, ok := cmp.(error); !ok {
					t.Errorf("%T should implement error, but doesn't", cmp)
				}
				if _, ok := cmp.(error); !ok {
					t.Errorf("%T should implement error, but doesn't", cmp)
				}
				if _, ok := cmp.(unwrapper); !ok {
					t.Errorf("%T should implement unwrapper, but doesn't", cmp)
				}
			})
			t.Run("Value", func(t *testing.T) {
				if cmp.String() != cmp.(error).Error() {
					t.Error("elements should be equal")
				}
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
