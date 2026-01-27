package message

import (
	"strings"
	"testing"

	"github.com/tychoish/grip/level"
)

func TestBuilderKVMethods(t *testing.T) {
	tests := []struct {
		name     string
		setup    func() *KV
		testFunc func(*testing.T, *KV)
	}{
		{
			name: "OptionMethod",
			setup: func() *KV {
				return NewKV()
			},
			testFunc: func(t *testing.T, kv *KV) {
				result := kv.WithOptions(OptionIncludeMetadata)
				if result != kv {
					t.Error("Option should return self")
				}
			},
		},
		{
			name: "LevelMethod",
			setup: func() *KV {
				return NewKV()
			},
			testFunc: func(t *testing.T, kv *KV) {
				result := kv.Level(level.Warning)
				if result != kv {
					t.Error("Level should return self")
				}
				if kv.Priority() != level.Warning {
					t.Error("Level should set priority")
				}
			},
		},
		{
			name: "KVMethod",
			setup: func() *KV {
				return NewKV()
			},
			testFunc: func(t *testing.T, kv *KV) {
				result := kv.KV("key", "value")
				if result != kv {
					t.Error("KV should return self")
				}
				if !kv.Loggable() {
					t.Error("KV with values should be loggable")
				}
			},
		},
		{
			name: "WhenKVTrue",
			setup: func() *KV {
				return NewKV()
			},
			testFunc: func(t *testing.T, kv *KV) {
				result := kv.WhenKV(true, "key", "value")
				if result != kv {
					t.Error("WhenKV should return self")
				}
				if !kv.Loggable() {
					t.Error("WhenKV(true) should add value")
				}
			},
		},
		{
			name: "WhenKVFalse",
			setup: func() *KV {
				return NewKV()
			},
			testFunc: func(t *testing.T, kv *KV) {
				result := kv.WhenKV(false, "key", "value")
				if result != kv {
					t.Error("WhenKV should return self")
				}
				if kv.Loggable() {
					t.Error("WhenKV(false) should not add value")
				}
			},
		},
		{
			name: "AnnotateMethod",
			setup: func() *KV {
				return NewKV()
			},
			testFunc: func(t *testing.T, kv *KV) {
				kv.Annotate("newkey", "newvalue")
				if !kv.Loggable() {
					t.Error("Annotate should make loggable")
				}
				str := kv.String()
				if !strings.Contains(str, "newkey") {
					t.Error("String should contain annotated key")
				}
			},
		},
		{
			name: "FieldsMethod",
			setup: func() *KV {
				return NewKV()
			},
			testFunc: func(t *testing.T, kv *KV) {
				fields := Fields{
					"key1": "value1",
					"key2": 42,
					"key3": true,
				}
				result := kv.Fields(fields)
				if result != kv {
					t.Error("Fields should return self")
				}
				if !kv.Loggable() {
					t.Error("Fields should make loggable")
				}
			},
		},
		{
			name: "ExtendMethod",
			setup: func() *KV {
				return NewKV()
			},
			testFunc: func(t *testing.T, kv *KV) {
				seq := func(yield func(string, any) bool) {
					yield("k1", "v1")
					yield("k2", "v2")
				}
				result := kv.Extend(seq)
				if result != kv {
					t.Error("Extend should return self")
				}
				if !kv.Loggable() {
					t.Error("Extend should make loggable")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kv := tt.setup()
			tt.testFunc(t, kv)
		})
	}
}

func TestBuilderKVLoggable(t *testing.T) {
	tests := []struct {
		name     string
		setup    func() *KV
		expected bool
	}{
		{
			name: "EmptyKV",
			setup: func() *KV {
				return NewKV()
			},
			expected: false,
		},
		{
			name: "WithOneKV",
			setup: func() *KV {
				return NewKV().KV("key", "value")
			},
			expected: true,
		},
		{
			name: "WithMultipleKVs",
			setup: func() *KV {
				return NewKV().KV("k1", "v1").KV("k2", "v2")
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kv := tt.setup()
			result := kv.Loggable()
			if result != tt.expected {
				t.Errorf("Loggable() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestBuilderKVString(t *testing.T) {
	tests := []struct {
		name       string
		setup      func() *KV
		wantSubstr []string
	}{
		{
			name: "EmptyKV",
			setup: func() *KV {
				return NewKV()
			},
			wantSubstr: []string{},
		},
		{
			name: "SingleKV",
			setup: func() *KV {
				return NewKV().KV("key", "value")
			},
			wantSubstr: []string{"key", "value"},
		},
		{
			name: "MultipleKVs",
			setup: func() *KV {
				return NewKV().KV("name", "test").KV("count", 42).KV("flag", true)
			},
			wantSubstr: []string{"name", "test", "count", "42", "flag", "true"},
		},
		{
			name: "CachedString",
			setup: func() *KV {
				kv := NewKV().KV("key", "val")
				_ = kv.String() // Call once to cache
				return kv
			},
			wantSubstr: []string{"key", "val"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kv := tt.setup()
			str := kv.String()

			for _, substr := range tt.wantSubstr {
				if !strings.Contains(str, substr) {
					t.Errorf("String() = %q, should contain %q", str, substr)
				}
			}

			// Call String again to test caching
			str2 := kv.String()
			if str != str2 {
				t.Error("String() should return same cached result")
			}
		})
	}
}

func TestBuilderKVStructured(t *testing.T) {
	tests := []struct {
		name     string
		setup    func() *KV
		expected bool
	}{
		{
			name: "AlwaysStructured",
			setup: func() *KV {
				return NewKV()
			},
			expected: true,
		},
		{
			name: "WithKVsAlwaysStructured",
			setup: func() *KV {
				return NewKV().KV("k", "v")
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kv := tt.setup()
			result := kv.Structured()
			if result != tt.expected {
				t.Errorf("Structured() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestBuilderKVRaw(t *testing.T) {
	tests := []struct {
		name     string
		setup    func() *KV
		validate func(*testing.T, any)
	}{
		{
			name: "EmptyKV",
			setup: func() *KV {
				return NewKV()
			},
			validate: func(t *testing.T, raw any) {
				if raw == nil {
					t.Error("Raw() should not return nil")
				}
			},
		},
		{
			name: "WithKVPairs",
			setup: func() *KV {
				return NewKV().KV("field1", "value1").KV("field2", 123)
			},
			validate: func(t *testing.T, raw any) {
				if raw == nil {
					t.Fatal("Raw() should not return nil")
				}
			},
		},
		{
			name: "WithMetadata",
			setup: func() *KV {
				kv := NewKV().KV("key", "val")
				kv.SetOption(OptionIncludeMetadata)
				return kv
			},
			validate: func(t *testing.T, raw any) {
				if raw == nil {
					t.Error("Raw() should include metadata")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kv := tt.setup()
			raw := kv.Raw()
			tt.validate(t, raw)
		})
	}
}

func TestMakeKV(t *testing.T) {
	tests := []struct {
		name     string
		create   func() Composer
		validate func(*testing.T, Composer)
	}{
		{
			name: "WithStringValues",
			create: func() Composer {
				seq := func(yield func(string, string) bool) {
					yield("k1", "v1")
					yield("k2", "v2")
				}
				return MakeKV(seq)
			},
			validate: func(t *testing.T, c Composer) {
				if !c.Loggable() {
					t.Error("should be loggable")
				}
				if !c.Structured() {
					t.Error("should be structured")
				}
			},
		},
		{
			name: "WithIntValues",
			create: func() Composer {
				seq := func(yield func(string, int) bool) {
					yield("count", 42)
					yield("total", 100)
				}
				return MakeKV(seq)
			},
			validate: func(t *testing.T, c Composer) {
				if !c.Loggable() {
					t.Error("should be loggable")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := tt.create()
			if c == nil {
				t.Fatal("MakeKV should not return nil")
			}
			tt.validate(t, c)
		})
	}
}

func TestKVFunction(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		value    any
		validate func(*testing.T, any)
	}{
		{
			name:  "StringValue",
			key:   "key",
			value: "value",
			validate: func(t *testing.T, result any) {
				// Should create an iterator
				if result == nil {
					t.Error("KV should return non-nil")
				}
			},
		},
		{
			name:  "IntValue",
			key:   "count",
			value: 42,
			validate: func(t *testing.T, result any) {
				if result == nil {
					t.Error("KV should return non-nil")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NewKV().KV(tt.key, tt.value)
			tt.validate(t, result)
		})
	}
}
