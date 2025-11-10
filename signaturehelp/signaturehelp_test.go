package signaturehelp_test

import (
	"fmt"
	"testing"

	"github.com/ethancarlsson/hurl-lsp/expect"
	"github.com/ethancarlsson/hurl-lsp/signaturehelp"
)

func TestLines(t *testing.T) {
	//					   tab|space |4 spaces
	//					     v v      v
	commonLines := []string{"hello world!", "this	line    is the middle", "the last line"}
	tests := []struct {
		lines    []string
		expected string
		coords   [2]int
	}{
		{
			lines:    commonLines,
			expected: "world!",
			coords:   [2]int{0, 5},
		},
		{
			lines:    commonLines,
			expected: "hello",
			coords:   [2]int{0, len("hello") - 2},
		},
		{
			lines:    commonLines,
			expected: "world!",
			coords:   [2]int{0, 6},
		},
		{
			lines:    commonLines,
			expected: "world!",
			coords:   [2]int{0, len("hello world!") - 2},
		},
		{
			lines:    commonLines,
			expected: "",
			coords:   [2]int{0, len("hello world!")},
		},
		{
			lines:    commonLines,
			expected: "line",
			coords:   [2]int{1, 5},
		},
		{
			lines:    commonLines,
			expected: "line",
			coords:   [2]int{1, 4},
		},
		{
			lines:    commonLines,
			expected: "this",
			coords:   [2]int{1, 0},
		},
		{
			lines:    commonLines,
			expected: "is",
			coords:   [2]int{1, 12},
		},
		{
			lines:    commonLines,
			expected: "is",
			coords:   [2]int{1, 13},
		},
		{
			lines:    commonLines,
			expected: "",
			coords:   [2]int{1, 14},
		},
		{
			lines:    commonLines,
			expected: "the",
			coords:   [2]int{1, 16},
		},
		{
			lines:    commonLines,
			expected: "the",
			coords:   [2]int{1, 17},
		},
		{
			lines:    commonLines,
			expected: "",
			coords:   [2]int{1, 18},
		},
		{
			lines:    commonLines,
			expected: "",
			coords:   [2]int{9, 18},
		},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("lines: %v, coords: %v, expected: \"%s\"", tt.lines, tt.coords, tt.expected), func(t *testing.T) {
			actual := signaturehelp.Lines(tt.lines).SymbolAt(tt.coords[0], tt.coords[1])
			expect.Equals(t, tt.expected, actual.String())
		})
	}
}
