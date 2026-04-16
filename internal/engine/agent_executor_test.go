package engine

import (
	"encoding/json"
	"testing"
)

func TestBuildUserPrompt(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		template string
		inputs   map[string]json.RawMessage
		want     string
	}{
		{
			name:     "template with placeholder and one input",
			template: "Write about: {{input}}",
			inputs:   map[string]json.RawMessage{"in": json.RawMessage(`"cats"`)},
			want:     "Write about: cats",
		},
		{
			name:     "empty template returns concatenated inputs",
			template: "",
			inputs:   map[string]json.RawMessage{"in": json.RawMessage(`"hello world"`)},
			want:     "hello world",
		},
		{
			name:     "fixed template with nil inputs",
			template: "Fixed prompt",
			inputs:   nil,
			want:     "Fixed prompt",
		},
		{
			name:     "template with placeholder replaces it",
			template: "Has {{input}} here",
			inputs:   map[string]json.RawMessage{"in": json.RawMessage(`"value"`)},
			want:     "Has value here",
		},
		{
			name:     "template without placeholder appends inputs",
			template: "Some prompt",
			inputs:   map[string]json.RawMessage{"in": json.RawMessage(`"extra data"`)},
			want:     "Some prompt\n\nextra data",
		},
		{
			name:     "empty template and nil inputs",
			template: "",
			inputs:   nil,
			want:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := buildUserPrompt(tt.template, tt.inputs)
			if got != tt.want {
				t.Errorf("buildUserPrompt(%q, ...) =\n  %q\nwant:\n  %q", tt.template, got, tt.want)
			}
		})
	}
}

func TestSubstituteVariables(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
		vars map[string]string
		want string
	}{
		{
			name: "single replacement",
			in:   "Write for {{name}}.",
			vars: map[string]string{"name": "Levi"},
			want: "Write for Levi.",
		},
		{
			name: "multiple replacements",
			in:   "Write for {{name}} about {{topic}}.",
			vars: map[string]string{"name": "Levi", "topic": "lighthouses"},
			want: "Write for Levi about lighthouses.",
		},
		{
			name: "undefined var stays literal so {{input}} keeps working downstream",
			in:   "Hello {{name}}, data: {{input}}",
			vars: map[string]string{"name": "Alex"},
			want: "Hello Alex, data: {{input}}",
		},
		{
			name: "empty vars map returns input unchanged",
			in:   "No {{var}} replacement",
			vars: nil,
			want: "No {{var}} replacement",
		},
		{
			name: "empty input returns empty",
			in:   "",
			vars: map[string]string{"x": "y"},
			want: "",
		},
		{
			name: "empty var name is ignored",
			in:   "Value: {{}}",
			vars: map[string]string{"": "nope"},
			want: "Value: {{}}",
		},
		{
			name: "repeated var replaced everywhere",
			in:   "{{x}} + {{x}} = 2{{x}}",
			vars: map[string]string{"x": "7"},
			want: "7 + 7 = 27",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := substituteVariables(tt.in, tt.vars)
			if got != tt.want {
				t.Errorf("substituteVariables(%q, %v) = %q, want %q", tt.in, tt.vars, got, tt.want)
			}
		})
	}
}

func TestConcatenateInputs(t *testing.T) {
	t.Parallel()

	t.Run("nil inputs returns empty string", func(t *testing.T) {
		t.Parallel()
		got := concatenateInputs(nil)
		if got != "" {
			t.Errorf("concatenateInputs(nil) = %q, want %q", got, "")
		}
	})

	t.Run("empty map returns empty string", func(t *testing.T) {
		t.Parallel()
		got := concatenateInputs(map[string]json.RawMessage{})
		if got != "" {
			t.Errorf("concatenateInputs({}) = %q, want %q", got, "")
		}
	})

	t.Run("JSON string input is unquoted", func(t *testing.T) {
		t.Parallel()
		inputs := map[string]json.RawMessage{
			"in": json.RawMessage(`"hello world"`),
		}
		got := concatenateInputs(inputs)
		if got != "hello world" {
			t.Errorf("concatenateInputs(...) = %q, want %q", got, "hello world")
		}
	})

	t.Run("raw JSON is returned as bytes", func(t *testing.T) {
		t.Parallel()
		inputs := map[string]json.RawMessage{
			"in": json.RawMessage(`{"key":"val"}`),
		}
		got := concatenateInputs(inputs)
		if got != `{"key":"val"}` {
			t.Errorf("concatenateInputs(...) = %q, want %q", got, `{"key":"val"}`)
		}
	})

	t.Run("number JSON is returned as raw bytes", func(t *testing.T) {
		t.Parallel()
		inputs := map[string]json.RawMessage{
			"in": json.RawMessage(`42`),
		}
		got := concatenateInputs(inputs)
		if got != "42" {
			t.Errorf("concatenateInputs(...) = %q, want %q", got, "42")
		}
	})
}
