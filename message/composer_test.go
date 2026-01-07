package message

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/tychoish/fun/assert/check"
	"github.com/tychoish/fun/fn"
	"github.com/tychoish/fun/irt"
	"github.com/tychoish/fun/testt"
	"github.com/tychoish/grip/level"
)

func TestPopulatedMessageComposerConstructors(t *testing.T) {
	const testMsg = "hello"
	// map objects to output
	cases := map[Composer]string{
		MakeString(testMsg):                                        testMsg,
		MakeBytes([]byte(testMsg)):                                 testMsg,
		MakeError(errors.New(testMsg)):                             testMsg,
		MakeFormat(string(testMsg[0])+"%s", testMsg[1:]):           testMsg,
		WrapError(errors.New("hello"), "world"):                    "world error='hello'",
		WrapErrorf(errors.New("hello"), "world"):                   "world error='hello'",
		MakeLines(testMsg, ""):                                     testMsg,
		MakeLines(testMsg):                                         testMsg,
		BuildGroupComposer(MakeString(testMsg)):                    testMsg,
		MakeGroupComposer([]Composer{MakeString(testMsg)}):         testMsg,
		MakeFields(Fields{"test": testMsg}):                        fmt.Sprintf("[test='%s']", testMsg),
		When(true, testMsg):                                        testMsg,
		Whenf(true, testMsg):                                       testMsg,
		Whenln(true, testMsg):                                      testMsg,
		Whenln(true, testMsg):                                      testMsg,
		MakeFuture(func() Composer { return MakeString(testMsg) }): testMsg,
		MakeFuture(func() error { return errors.New(testMsg) }):    testMsg,
	}

	for msg, output := range cases {
		t.Run(fmt.Sprintf("$%T", msg), func(t *testing.T) {
			if msg == nil {
				t.Error("value should not be nill")
			}

			if !msg.Loggable() {
				t.Errorf("value should be true [%T]", msg)
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
				if str := msg.String(); str != output {
					t.Errorf("%T [%s != %s]", msg, str, output)
				}
				if str := msg.String(); str != output {
					t.Errorf("%T [%s != %s]", msg, str, output)
				}
			}

			if msg.Priority() != level.Invalid {
				if level.Error != msg.Priority() {
					t.Error("elements should be equal")
				}
				previous := msg.Priority()
				msg.SetPriority(msg.Priority())

				if previous != msg.Priority() {
					t.Error(previous, ">", msg.Priority())
				}
			}
		})
	}
}

func TestUnpopulatedMessageComposers(t *testing.T) {
	// map objects to output
	cases := []Composer{
		&stringMessage{},
		MakeString(""),
		&bytesMessage{},
		MakeBytes([]byte{}),
		&lineMessenger{},
		MakeLines(),
		&formatMessenger{},
		MakeFormat(""),
		BuildGroupComposer(),
		MakeError(nil),
		When(false, ""),
		Whenln(false, "", ""),
		MakeFuture(func() Composer { return nil }),
		MakeFuture(func() Fields { return nil }),
		MakeFuture(func() Fields { return Fields{} }),
		MakeFuture(func() error { return nil }),
	}

	for idx, msg := range cases {
		t.Run(fmt.Sprintf("%T<%d>", idx, msg), func(t *testing.T) {
			if msg.Loggable() {
				t.Errorf("message %T at %d should not be loggable", msg, idx)
			}
			if msg.String() != "" {
				t.Errorf("string value %T: [%s]", msg, msg.String())
			}
		})
	}
}

func TestStackMessages(t *testing.T) {
	const testMsg = "hello"
	stackMsg := "message/composer_test"

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

	for idx, msg := range cases {
		t.Run(fmt.Sprint(idx), func(t *testing.T) {
			comp := Convert(msg)
			comp.SetPriority(level.Error)
			if !comp.Loggable() {
				t.Error("value should be true")
			}
			if testMsg != comp.String() {
				t.Log("expected:", testMsg)
				t.Log("actual", comp)
				t.Errorf("%T >> %T", msg, comp)
			}
		})
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
		t.Run(fmt.Sprintf("%T/%d", msg, idx), func(t *testing.T) {
			comp := Convert(msg)
			comp.SetPriority(level.Error)
			if comp.Loggable() {
				t.Errorf("should be false: %T", comp)
			}
			if comp.String() != "" {
				testt.Logf(t, "%T>%s", comp, comp.String())
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
		comp := Convert(in)
		comp.SetPriority(level.Error)
		if !comp.Loggable() {
			t.Error("value should be true")
		}
		if !strings.HasPrefix(comp.String(), out) {
			t.Logf("out=%q comp=%q", out, comp.String())
			t.Error("value should be true")
		}
	}
}

func TestErrors(t *testing.T) {
	for name, cmp := range map[string]Composer{
		"Wrapped": WrapError(errors.New("err"), "wrap"),
		"Plain":   MakeError(errors.New("err")),
	} {
		t.Run(name, func(t *testing.T) {
			t.Run("Interfaces", func(t *testing.T) {
				check.True(t, cmp.Loggable())
				if _, ok := cmp.(error); !ok {
					t.Errorf("%T should implement error, but doesn't", cmp)
				}
				switch {
				case IsMulti(cmp):
				case errors.Unwrap(cmp.(error)) != nil:
				default:
					t.Errorf("should be wrapped error or wrapped composer: %T", cmp)
				}
			})
			t.Run("Value", func(t *testing.T) {
				if cmp.String() != cmp.(error).Error() {
					t.Error("elements should be equal")
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
			name:  "PairsStrings",
			input: []any{"hello", "world", "val", "3000"},
			output: MakeKV(
				irt.KVargs(
					irt.MakeKV[string, any]("hello", "world"),
					irt.MakeKV[string, any]("val", "3000"),
				),
			),
		},
		{
			name:  "PairsMixed",
			input: []any{"hello", "world", "val", 3000},
			output: MakeKV(
				irt.KVargs(
					irt.MakeKV[string, any]("hello", "world"),
					irt.MakeKV[string, any]("val", 3000),
				),
			),
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
				t.Errorf("unexpected output: %+v vs %+v",
					ex, c.output)
			}
		})
	}
}

func fixTimestamps(t *testing.T, msgs ...Composer) {
	ts := time.Now().Round(time.Millisecond)
	for _, msg := range msgs {
		switch m := msg.(type) {
		case *fieldMessage:
			m.Time = ts
		case *lineMessenger:
			m.Time = ts
		case *BuilderKV:
			m.Time = ts
		case *GroupComposer:
			fixTimestamps(t, m.Messages()...)
		}
	}
}

type ConverterCases struct {
	Name         string
	Input        any
	Expected     Composer
	IsStructured bool
	Unloggable   bool
}

func TestConverter(t *testing.T) {
	cases := []ConverterCases{
		{
			Name:         "ComposerProducerFunction",
			Input:        func() Composer { return MakeString("hello world") },
			Expected:     MakeString("hello world"),
			IsStructured: false,
		},
		{
			Name:         "ComposerNilFunction",
			Input:        fn.Future[Composer](nil),
			Expected:     MakeError(nil),
			IsStructured: true,
			Unloggable:   true,
		},
		{
			Name:         "NilComposerProducer",
			Input:        &composerFutureMessage{},
			Expected:     MakeKV(irt.Map(map[string]any{})),
			IsStructured: true,
			Unloggable:   true,
		},
		{
			Name:         "ErrorProducerFunction",
			Input:        func() error { return errors.New("hello world") },
			Expected:     MakeError(errors.New("hello world")),
			IsStructured: false,
		},
		{
			Name:         "ComposerProducer",
			Input:        fn.Future[Composer](func() Composer { return MakeString("hello world") }),
			Expected:     MakeString("hello world"),
			IsStructured: false,
		},
		{
			Name:         "ErrorProducer",
			Input:        fn.MakeFuture(func() error { return errors.New("hello world") }),
			Expected:     MakeError(errors.New("hello world")),
			IsStructured: false,
		},
		{
			Name:         "StaticErrorProducer",
			Input:        fn.AsFuture(errors.New("hello world")),
			Expected:     MakeError(errors.New("hello world")),
			IsStructured: false,
		},
		{
			Name:         "FieldsProducerFunction",
			Input:        func() Fields { return Fields{"hello": "world"} },
			Expected:     MakeFields(Fields{"hello": "world"}),
			IsStructured: true,
		},
		{
			Name:         "AnyMapFunction",
			Input:        func() map[string]any { return map[string]any{"hello": "world"} },
			Expected:     MakeFields(Fields{"hello": "world"}),
			IsStructured: true,
		},
		{
			Name:         "FieldsProducer",
			Input:        MakeFuture(func() Fields { return Fields{"hello": "world"} }),
			Expected:     MakeFields(Fields{"hello": "world"}),
			IsStructured: true,
		},
		{
			Name:     "SliceSingle",
			Input:    []any{"hello world"},
			Expected: MakeString("hello world"),
		},
		{
			Name:     "Bytes",
			Input:    []byte("hello world"),
			Expected: MakeBytes([]byte("hello world")),
		},
		{
			Name:     "SliceSingle",
			Input:    []any{MakeLines("hello world")},
			Expected: MakeString("hello world"),
		},
		{
			Name:         "EmptySlice",
			Input:        []any{},
			Expected:     MakeLines(),
			IsStructured: true,
			Unloggable:   true,
		},
		{
			Name:         "NestedEmptySlice",
			Input:        []any{[]Composer{}},
			Expected:     MakeLines(),
			IsStructured: false,
			Unloggable:   true,
		},
		{
			Name:         "SliceComposerProducer",
			Input:        []fn.Future[Composer]{func() Composer { return MakeString("hello world") }},
			Expected:     MakeString("hello world"),
			IsStructured: false,
		},
		{
			Name:         "EmptySliceComposerProducer",
			Input:        []fn.Future[Composer]{},
			Expected:     MakeString(""),
			IsStructured: true,
			Unloggable:   true,
		},
		{
			Name:         "GroupSlice",
			Input:        []Composer{MakeString("kip"), MakeString("merlin")},
			Expected:     MakeLines("kip\nmerlin"),
			IsStructured: false,
		},
		{
			Name:  "GroupFields",
			Input: []Fields{{"hello": 2001}, {"world": 42}},
			Expected: BuildGroupComposer(
				MakeFields(Fields{"hello": 2001}),
				MakeFields(Fields{"world": 42}),
			),
			IsStructured: true,
		},
		{
			Name:         "BuilderFields",
			Input:        Fields{"hello": 2001, "world": 42},
			Expected:     NewBuilder(func(Composer) {}, testConverter(t, true)).Fields(Fields{"hello": 2001, "world": 42}).Message(),
			IsStructured: true,
		},
	}
	for _, tt := range cases {
		t.Run(tt.Name, func(t *testing.T) {
			tt.Expected.SetOption(OptionSkipCollectInfo)
			tt.Expected.SetOption(OptionSkipMetadata)
			for convMethod, got := range map[string]Composer{
				"Converter":       Convert(tt.Input),
				"BuilderProducer": NewBuilder(nil, testConverter(t, true)).Future().Convert(func() any { return tt.Input }).Builder().Message(),
				"AddToBuilder":    WithFuture(NewBuilder(nil, testConverter(t, true)), func() Composer { return Convert(tt.Input) }).Message(),
				"Builder":         NewBuilder(nil, testConverter(t, true)).Any(tt.Input).Message(),
				"BuilderComposer": NewBuilder(nil, testConverter(t, true)).Composer(Convert(tt.Input)).Message(),
			} {
				t.Run(convMethod, func(t *testing.T) {
					got.SetOption(OptionSkipCollectInfo)
					got.SetOption(OptionSkipMetadata)
					check.Equal(t, got.Loggable(), tt.Expected.Loggable())
					check.Equal(t, got.String(), tt.Expected.String())
					check.True(t, got.Structured() == tt.IsStructured)
					check.True(t, got.Loggable() == !tt.Unloggable)
					testt.Logf(t, "got<%T>:%q", got, got)
					testt.Logf(t, "had<%T>:%q", tt.Expected, tt.Expected)
				})
			}
		})
	}
}
