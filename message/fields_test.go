package message

import (
	"testing"

	"github.com/tychoish/grip/level"
)

func TestFieldsLevelMutability(t *testing.T) {
	m := Fields{"message": "hello world"}
	c := Convert(m)
	c.SetPriority(level.Error)
	c.SetOption(OptionIncludeMetadata)

	r := c.Raw().(Fields)
	if level.Error != c.Priority() {
		t.Error("elements shold be equal")
	}

	if level.Error != r["meta"].(*Base).Level {
		t.Error("elements shold be equal")
	}

	c = Convert(m)
	c.SetPriority(level.Info)
	c.SetOption(OptionIncludeMetadata)

	r = c.Raw().(Fields)
	if level.Info != c.Priority() {
		t.Error("elements shold be equal")
	}
	if level.Info != r["meta"].(*Base).Level {
		t.Error("elements shold be equal")
	}
}

func TestDefaultFieldsMessage(t *testing.T) {
	if out := GetDefaultFieldsMessage(MakeFields(Fields{"msg": "hello world"}), "what"); out != "hello world" {
		t.Fatal("incorrect form resolved")
	}

	if out := GetDefaultFieldsMessage(MakeFields(Fields{"msg": ""}), "what"); out != "" {
		t.Fatal("bad default for empty value")
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
