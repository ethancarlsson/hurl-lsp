package openapi

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"pgregory.net/rapid"
)

func TestPropTypeDiagnostics(t *testing.T) {

	tests := []struct {
		schema   Schema
		val      any
		expected string
	}{
		{
			schema:   Schema{Type: MParsedS("string")},
			val:      123,
			expected: `Expected string got number`,
		},
		{
			schema:   Schema{Type: MParsedS("string")},
			val:      nil,
			expected: `Expected string got null`,
		},
		{
			schema:   Schema{Type: MParsedS("object")},
			val:      []any{},
			expected: `Expected object got array`,
		},
		{
			schema:   Schema{Type: MParsedS("object")},
			val:      1.23,
			expected: `Expected object got number`,
		},
		{
			schema: Schema{
				OneOf: MParsedV([]Schema{
					{
						Type: MParsedS("number"),
					},
					{
						Type: MParsedS("array"),
					},
				}),
			},
			val:      "not a number or an array",
			expected: `Expected number|array got string`,
		},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("invalid type with schema: %s", tt.schema.ToString(0)), func(t *testing.T) {
			diagnostics := propTypeDiagnostics([]Diagnostic{}, []string{"path"}, tt.val, tt.schema)
			require.NotEmpty(t, diagnostics)
			assert.Equal(t, tt.expected, diagnostics[0].Diagnostic)
		})
	}

	testsValid := []struct {
		schema Schema
		val    any
	}{
		{
			schema: Schema{Type: MParsedS("string")},
			val:    "",
		},
		{
			schema: Schema{Type: MParsedS("string"), Nullable: MParsedV(true)},
			val:    nil,
		},
		{
			schema: Schema{Type: MParsedS("object")},
			val:    map[string]any{},
		},
		{
			schema: Schema{Type: MParsedS("array")},
			val:    []any{},
		},
		{
			schema: Schema{
				OneOf: MParsedV([]Schema{
					{
						Type: MParsedS("number"),
					},
					{
						Type: MParsedS("array"),
					},
				}),
			},
			val: 123,
		},
	}
	for _, tt := range testsValid {
		t.Run(fmt.Sprintf("type with schema: %s", tt.schema.ToString(0)), func(t *testing.T) {
			diagnostics := propTypeDiagnostics([]Diagnostic{}, []string{"path"}, tt.val, tt.schema)
			assert.Empty(t, diagnostics)
		})
	}

	t.Run("number properties", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			num := rapid.Float64().Draw(t, "float64")
			schema := Schema{
				Type: MParsedS("number"),
			}

			diagnostics := propTypeDiagnostics([]Diagnostic{}, []string{"path"}, num, schema)
			assert.Empty(t, diagnostics)
		})

		rapid.Check(t, func(t *rapid.T) {
			num := rapid.Int().Draw(t, "int")
			schema := Schema{
				Type: MParsedS("number"),
			}

			diagnostics := propTypeDiagnostics([]Diagnostic{}, []string{"path"}, num, schema)
			assert.Empty(t, diagnostics)
		})

		rapid.Check(t, func(t *rapid.T) {
			num := rapid.Int().Draw(t, "int")
			schema := Schema{
				Type: MParsedS("integer"),
			}

			diagnostics := propTypeDiagnostics([]Diagnostic{}, []string{"path"}, num, schema)
			assert.Empty(t, diagnostics)
		})
	})
}
