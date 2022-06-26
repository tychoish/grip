package message

import (
	"testing"

	"github.com/tychoish/grip/level"
)

func TestFieldsLevelMutability(t *testing.T) {
	m := Fields{"message": "hello world"}
	c := ConvertWithPriority(level.Error, m)

	r := c.Raw().(Fields)
	if level.Error != c.Priority() {
		t.Error("elements shold be equal")
	}
	if level.Error != r["metadata"].(*Base).Level {
		t.Error("elements shold be equal")
	}

	c = ConvertWithPriority(level.Info, m)
	r = c.Raw().(Fields)
	if level.Info != c.Priority() {
		t.Error("elements shold be equal")
	}
	if level.Info != r["metadata"].(*Base).Level {
		t.Error("elements shold be equal")
	}
}

func TestDefaultFieldsMessage(t *testing.T) {
	if out := GetDefaultFieldsMessage(MakeFields(Fields{"message": "hello world"}), "what"); out != "hello world" {
		t.Fatal("incorrect form resolved")
	}

	if out := GetDefaultFieldsMessage(MakeFields(Fields{"message": ""}), "what"); out != "" {
		t.Fatal("bad default for empty value")
	}

	if out := GetDefaultFieldsMessage(MakeAnnotated("hello", Fields{"message": ""}), "what"); out != "hello" {
		t.Fatal("bad default for annotated value")
	}

	if out := GetDefaultFieldsMessage(MakeAnnotated("", Fields{"message": "hello world"}), "what"); out != "hello world" {
		t.Fatal("bad default for annotated value")
	}

	if out := GetDefaultFieldsMessage(MakeAnnotated("", Fields{"message": ""}), "what"); out != "" {
		t.Fatal("bad default for annotated value")
	}

	if out := GetDefaultFieldsMessage(MakeAnnotated("", Fields{}), "what"); out != "what" {
		t.Fatal("bad default for annotated value")
	}

	if out := GetDefaultFieldsMessage(&fieldMessage{}, "what"); out != "what" {
		t.Fatal("bad default for annotated value")
	}

	if out := GetDefaultFieldsMessage(MakeFields(Fields{"val": "hello world"}), "what"); out != "what" {
		t.Fatal("missed message")
	}

	if out := GetDefaultFieldsMessage(MakeString("hello world"), "what"); out != "what" {
		t.Fatal("unsafe for non-fields messages")
	}

}
