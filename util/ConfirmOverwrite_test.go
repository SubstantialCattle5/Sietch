package util

import (
	"bytes"
	"io"
	"strings"
	"testing"
)

func TestConfirmOverwrite(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
		err      error
	}{
		{
			name:     "yes input",
			input:    "y\n",
			expected: true,
			err:      nil,
		},
		{
			name:     "yes uppercase",
			input:    "YES\n",
			expected: true,
			err:      nil,
		},
		{
			name:     "no input",
			input:    "n\n",
			expected: false,
			err:      nil,
		},
		{
			name:     "empty input",
			input:    "\n",
			expected: false,
			err:      nil,
		},
		{
			name:     "input not terminating with \\n",
			input:    "yes",
			expected: false,
			err:      io.EOF,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := strings.NewReader(tt.input)
			out := &bytes.Buffer{}
			ok, err := ConfirmOverwrite("testfile", in, out)
			if err != tt.err || ok != tt.expected {
				t.Errorf("ConfirmOverwrite('testfile', %s, out) = (%t, %s), want = (%t, nil)", tt.input, ok, err.Error(), tt.expected)
			}
		})
	}
}
