package add

import (
	"testing"
)

func TestParseAddTaskNumber(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantNumber int64
		wantOK     bool
	}{
		{
			name:       "valid command",
			input:      "/add 42",
			wantNumber: 42,
			wantOK:     true,
		},
		{
			name:       "valid with large number",
			input:      "/add 3000",
			wantNumber: 3000,
			wantOK:     true,
		},
		{
			name:       "valid with leading spaces",
			input:      "  /add 1  ",
			wantNumber: 1,
			wantOK:     true,
		},
		{
			name:   "missing number",
			input:  "/add",
			wantOK: false,
		},
		{
			name:   "extra parts",
			input:  "/add 42 extra",
			wantOK: false,
		},
		{
			name:   "zero number",
			input:  "/add 0",
			wantOK: false,
		},
		{
			name:   "negative number",
			input:  "/add -5",
			wantOK: false,
		},
		{
			name:   "non-numeric",
			input:  "/add abc",
			wantOK: false,
		},
		{
			name:   "empty string",
			input:  "",
			wantOK: false,
		},
		{
			name:   "only spaces",
			input:  "   ",
			wantOK: false,
		},
		{
			name:   "single word",
			input:  "hello",
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			num, ok := parseAddTaskNumber(tt.input)
			if ok != tt.wantOK {
				t.Errorf("parseAddTaskNumber(%q) ok = %v, want %v", tt.input, ok, tt.wantOK)
			}
			if ok && num != tt.wantNumber {
				t.Errorf("parseAddTaskNumber(%q) number = %d, want %d", tt.input, num, tt.wantNumber)
			}
		})
	}
}
