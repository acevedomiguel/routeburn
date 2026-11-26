package format

import "testing"

func TestShortModelName(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"deepseek/deepseek-v4.1-flash-20260910", "deepseek-v4.1-flash"},
		{"anthropic/claude-sonnet-4", "claude-sonnet-4"},
		{"openai/gpt-5", "gpt-5"},
		{"meta/llama-4-70b", "llama-4-70b"},
		{"openrouter/auto", "auto"},
		{"kimi-k3.1-pro-20261023", "kimi-k3.1-pro"},
		{"Other (remaining)", "Other (remaining)"},
		{"Other", "Other"},
		{"", ""},
		{"vendor/", "vendor"},
		{"-20260910", "-20260910"},
		{"a/b/c", "b/c"},
	}
	for _, tc := range cases {
		if got := ShortModelName(tc.in); got != tc.want {
			t.Errorf("ShortModelName(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// A short name must never be empty when the input had a name, otherwise a
// panel row would lose its label entirely.
func TestShortModelNameNeverEmptyForNamedModel(t *testing.T) {
	for _, in := range []string{"vendor/model", "vendor/", "model-20260101"} {
		if got := ShortModelName(in); got == "" {
			t.Errorf("ShortModelName(%q) returned empty", in)
		}
	}
}
