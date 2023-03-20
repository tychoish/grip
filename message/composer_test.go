package message

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/tychoish/fun"
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
		NewConvertedFieldsProducer(level.Error, func() map[string]any { return map[string]any{"pro": "ducer"} }): "[pro='ducer']",
		// NewEmailMessage(level.Error, Email{
		//	Recipients: []string{"someone@example.com"},
		//	Subject:    "Test msg",
		//	Body:       testMsg,
		// }): fmt.Sprintf("To: someone@example.com; Body: %s", testMsg),
		// NewGithubStatusMessage(level.Error, "tests", GithubStateError, "https://example.com", testMsg): fmt.Sprintf("tests error: %s (https://example.com)", testMsg),
		// NewGithubStatusMessageWithRepo(level.Error, GithubStatus{
		//	Owner:       "tychoish",
		//	Repo:        "grip",
		//	Ref:         "master",
		//	Context:     "tests",
		//	State:       GithubStateError,
		//	URL:         "https://example.com",
		//	Description: testMsg,
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
		&lineMessenger{},
		MakeLines(),
		NewLines(level.Error),
		&formatMessenger{},
		MakeSimpleKV(),
		MakeSimpleBytes(nil),
		MakeSimpleKVs(KVs{}),
		MakeFormat(""),
		NewFormat(level.Error, ""),
		MakeStack(1, ""),
		BuildGroupComposer(),
		&GroupComposer{},
		When(false, ""),
		Whenf(false, "", ""),
		Whenln(false, "", ""),
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

	cases := []any{
		MakeLines(testMsg),
		testMsg,
		errors.New(testMsg),
		[]string{testMsg},
		[]any{testMsg},
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

	cases = []any{
		nil,
		"",
		[]any{},
		[]string{},
		[]byte{},
		Fields{},
		map[string]any{},
	}

	for idx, msg := range cases {
		t.Run(fmt.Sprint(idx), func(t *testing.T) {
			comp := ConvertWithPriority(level.Error, msg)
			if comp.Loggable() {
				t.Errorf("should be false: %T", comp)
			}
			if "" != comp.String() {
				t.Errorf("%T", msg)
			}
		})

	}

	outputCases := map[string]any{
		"1":           1,
		"2":           int32(2),
		"message='3'": Fields{"message": 3},
		"message='4'": map[string]any{"message": "4"},
	}

	for out, in := range outputCases {
		comp := ConvertWithPriority(level.Error, in)
		if !comp.Loggable() {
			t.Error("value should be true")
		}
		if !strings.HasPrefix(comp.String(), out) {
			t.Logf("out=%q comp=%q", out, comp.String())
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
				switch {
				case fun.IsWrapped(cmp):
				case fun.IsWrapped(cmp.(error)):
				default:
					t.Error("should be wrapped error or wrapped composer")
				}
			})
			t.Run("Value", func(t *testing.T) {
				if cmp.String() != cmp.(error).Error() {
					t.Error("elements should be equal")
				}
			})
			t.Run("ExtendedFormat", func(t *testing.T) {
				if fmt.Sprintf("%+v", cmp) == fmt.Sprintf("%v", cmp) {
					t.Errorf("extended values should not be the same")
				}
			})

		})
	}
}

func TestSlice(t *testing.T) {
	cases := []struct {
		name   string
		input  []any
		output Composer
	}{
		{
			name:   "OneString",
			input:  []any{"hello world"},
			output: MakeLines("hello world"),
		},
		{
			name:   "ThreeStrings",
			input:  []any{"hello", "world", "3000"},
			output: MakeLines("hello", "world", "3000"),
		},
		{
			name:   "PairsStrings",
			input:  []any{"hello", "world", "val", "3000"},
			output: MakeKV(KV{"hello", "world"}, KV{"val", "3000"}),
		},
		{
			name:   "PairsMixed",
			input:  []any{"hello", "world", "val", 3000},
			output: MakeKV(KV{"hello", "world"}, KV{"val", 3000}),
		},
		{
			name:   "KeyNotString",
			input:  []any{"hello", "world", 3000, "kip"},
			output: MakeLines("hello", "world", 3000, "kip"),
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ex := buildFromSlice(c.input)
			fixTimestamps(t, ex, c.output)

			if ex.String() != c.output.String() {
				t.Log("output", ex.String())
				t.Log("expected", c.output.String())
				t.Fatalf("unexpected output: %T", ex)
			}
		})

	}
}

func fixTimestamps(t *testing.T, msgs ...Composer) {
	ts := time.Now().Round(time.Millisecond)
	for idx, msg := range msgs {
		switch m := msg.(type) {
		case *fieldMessage:
			m.Base.Time = ts
		case *lineMessenger:
			m.Base.Time = ts
		case *kvMsg:
			m.skipMetadata = true
			m.Base.Time = ts
		default:
			t.Errorf("id=%d %T", idx, m)
		}
	}
}
