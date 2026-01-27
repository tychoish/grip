package message

import (
	"testing"

	"github.com/tychoish/fun/dt"
	"github.com/tychoish/fun/irt"
	"github.com/tychoish/grip/level"
)

func TestFieldsLevelMutability(t *testing.T) {
	m := Fields{"message": "hello world"}
	c := Convert(m)
	c.SetPriority(level.Error)
	c.SetOption(OptionIncludeMetadata)

	r := c.Raw().(*dt.OrderedMap[string, any])
	if level.Error != c.Priority() {
		t.Error("elements shold be equal")
	}

	if level.Error != r.Get("meta").(*Base).Level {
		t.Error("elements shold be equal")
	}

	c = Convert(m)
	c.SetPriority(level.Info)
	c.SetOption(OptionIncludeMetadata)

	r = c.Raw().(*dt.OrderedMap[string, any])
	if level.Info != c.Priority() {
		t.Error("elements shold be equal")
	}
	if level.Info != r.Get("meta").(*Base).Level {
		t.Error("elements shold be equal")
	}
}

func TestDefaultFieldsMessage(t *testing.T) {
	if out := GetDefaultFieldsMessage(NewKV().Extend(irt.Map(Fields{"msg": "hello world"})).WithOptions(OptionSortMessageComponents), "what"); out != "hello world" {
		t.Log(out)
		t.Fatal("incorrect form resolved")
	}

	if out := GetDefaultFieldsMessage(NewKV().Extend(irt.Map(Fields{"msg": ""})).WithOptions(OptionSortMessageComponents), "what"); out != "" {
		t.Fatal("bad default for empty value")
	}

	if out := GetDefaultFieldsMessage(NewKV(), "what"); out != "what" {
		t.Fatal("bad default for annotated value")
	}

	if out := GetDefaultFieldsMessage(MakeFields(Fields{"val": "hello world"}), "what"); out != "what" {
		t.Fatal("missed message")
	}

	if out := GetDefaultFieldsMessage(MakeString("hello world"), "what"); out != "what" {
		t.Fatal("unsafe for non-fields messages")
	}
}
