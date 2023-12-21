// main_test.go
package main

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRot(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		args     []string
		expected string
	}{
		{
			name:     "default rotation",
			input:    "IBM",
			args:     []string{"rot"},
			expected: "VOZ\n",
		},
		{
			name:     "custom rotation, long flag",
			input:    "IBM",
			args:     []string{"rot", "--rotate", "25"},
			expected: "HAL\n",
		},
		{
			name:     "custom rotation, short flag",
			input:    "IBM",
			args:     []string{"rot", "--r", "1"},
			expected: "JCN\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := runTestApp(tt.args, tt.input)
			assert.Equal(t, tt.expected, output)
		})
	}
}

func runTestApp(args []string, input string) string {
	in := strings.NewReader(input)
	out := new(bytes.Buffer)

	cmd := setupCmd(in, out)
	cmd.Run(context.Background(), args)
	return out.String()
}
