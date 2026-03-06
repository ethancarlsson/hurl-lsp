package openapi

import (
	"encoding/json"
	"fmt"
	"strings"
)

type Diagnostic struct {
	Diagnostic string
	Path       []string
}

func (d Diagnostic) Error() string {
	return fmt.Sprintf(`%s "%s"`, d.Diagnostic, strings.Join(d.Path, "/"))
}

func DiagnoseAgainstSchema(provided map[string]json.RawMessage, schema Schema) []Diagnostic {
	path := []string{}
	diagnostics := make([]Diagnostic, 0, len(schema.Required.OrEmpty())*2)

	return collectSchemaDiagnostics(diagnostics, path, provided, schema)
}

func collectSchemaDiagnostics(diagnostics []Diagnostic, path []string, provided map[string]json.RawMessage, schema Schema) []Diagnostic {
	for _, required := range schema.Required.OrEmpty() {
		if _, ok := provided[required]; !ok {
			diagnostics = append(diagnostics, Diagnostic{
				Diagnostic: "Missing required property",
				Path:       append(path, required),
			})
		}
	}

	// collect for objects
	for name, prop := range schema.Properties.OrEmpty() {
		p := prop.OrEmpty()
		if p.Properties.Or(nil) == nil {
			continue
		}

		provided, ok := provided[name]

		if !ok {
			continue
		}

		var providedVal any
		if err := json.Unmarshal(provided, &providedVal); err != nil {
			continue
		}

		var obj map[string]json.RawMessage
		if err := json.Unmarshal(provided, &obj); err != nil {
			continue
		}

		newPath := append(path, name)
		diagnostics = collectSchemaDiagnostics(diagnostics, newPath, obj, p)
	}

	// collect for additionalProperties objects
	for name, prop := range schema.Properties.OrEmpty() {
		additionalProperties := prop.OrEmpty().AdditionalProperties.Or(nil)
		if additionalProperties == nil {
			continue
		}

		provided, ok := provided[name]
		if !ok {
			continue
		}

		var additionalProps map[string]map[string]json.RawMessage
		if err := json.Unmarshal(provided, &additionalProps); err != nil {
			continue
		}

		for additionalName, obj := range additionalProps {
			newPath := append(path, name, additionalName)
			diagnostics = collectSchemaDiagnostics(diagnostics, newPath, obj, *additionalProperties)
		}
	}

	// collect for arrays
	for name, prop := range schema.Properties.OrEmpty() {
		propItems := prop.OrEmpty().Items.Or(nil)
		if propItems == nil {
			continue
		}

		provided, ok := provided[name]
		if !ok {
			continue
		}

		var items []map[string]json.RawMessage
		if err := json.Unmarshal(provided, &items); err != nil {
			continue
		}

		for i, obj := range items {
			newPath := append(path, name, fmt.Sprintf("%d", i))
			diagnostics = collectSchemaDiagnostics(diagnostics, newPath, obj, *propItems)
		}
	}

	return diagnostics
}
