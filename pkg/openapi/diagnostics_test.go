package openapi_test

import (
	"encoding/json"
	"testing"

	"github.com/ethancarlsson/hurl-lsp/pkg/openapi"
	"github.com/stretchr/testify/assert"
	"pgregory.net/rapid"
)

func TestDiagnoseAgainstSchema(t *testing.T) {

	rapid.Check(t, func(t *rapid.T) {
		var (
			in              = rapid.Make[map[string]json.RawMessage]().Draw(t, "provided")
			required        = rapid.SliceOf(rapid.String()).Draw(t, "required")
			otherProperties = rapid.SliceOf(rapid.String()).Draw(t, "other_properties")
		)

		properties := make(map[string]openapi.MaybeParsed[openapi.Schema], len(required)+len(otherProperties))

		for _, prop := range otherProperties {
			properties[prop] = openapi.MParsedV(openapi.Schema{
				Type: openapi.MParsedS("array"),
				Items: openapi.MParsedV(&openapi.Schema{
					Type: openapi.MParsedS("string"),
				}),
			})

		}

		alwaysMissingProp := "missing_prop"
		delete(in, alwaysMissingProp)
		required = append(required, alwaysMissingProp)

		schema := openapi.Schema{
			Required:   openapi.MParsedV(required),
			Properties: openapi.MParsedV(properties),
		}

		diagnostics := openapi.DiagnoseAgainstSchema(in, schema)

		errs := make([]string, 0, len(diagnostics))
		for _, d := range diagnostics {
			errs = append(errs, d.Error())
		}

		// There should always at least be a diagnostic about the missing prop
		assert.GreaterOrEqual(t, len(diagnostics), 1)
		assert.Contains(t, errs, `Missing required property "missing_prop"`)
	})
}
