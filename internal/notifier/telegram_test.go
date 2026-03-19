package notifier

import "testing"

func TestEscapeTelegramMD_SpecialChars(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"hello_world", `hello\_world`},
		{"bold*text", `bold\*text`},
		{"`code`", "\\`code\\`"},
		{"[link]", `\[link]`},
		{"no special chars", "no special chars"},
		{"", ""},
		{"_*`[all", "\\_\\*\\`\\[all"},
	}
	for _, tt := range tests {
		got := escapeTelegramMD(tt.input)
		if got != tt.want {
			t.Errorf("escapeTelegramMD(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestEscapeTelegramMD_NoDoubleEscape(t *testing.T) {
	// A backslash already in the string should not be affected
	input := `plain\text`
	got := escapeTelegramMD(input)
	if got != input {
		t.Errorf("escapeTelegramMD(%q) = %q, want unchanged %q", input, got, input)
	}
}
