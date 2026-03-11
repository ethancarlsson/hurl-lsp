package openapi

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
)

type Diagnostic struct {
	Diagnostic  string
	Path        []string
	errOnParent bool
}

func (d Diagnostic) Error() string {
	return fmt.Sprintf(`%s "%s"`, d.Diagnostic, strings.Join(d.Path, "/"))
}

// RangePath tells the system where the error is.by default it is the same as
// Path, but if the error is actually an issue with the parent it will return
// the Path without the last element.
func (d Diagnostic) RangePath() []string {
	if len(d.Path) == 0 {
		return d.Path
	}

	if d.errOnParent {
		return d.Path[:len(d.Path)-1]
	}

	return d.Path
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
				Diagnostic:  "Missing required property",
				Path:        append(path, required),
				errOnParent: true,
			})
		}
	}

	// collect for objects
	for name, prop := range schema.Properties.OrEmpty() {
		p := prop.OrEmpty()

		provided, ok := provided[name]

		if !ok {
			continue
		}

		var providedVal any
		if err := json.Unmarshal(provided, &providedVal); err != nil {
			continue
		}

		newPath := append(path, name)

		diagnostics = propTypeDiagnostics(diagnostics, newPath, providedVal, p)

		if p.Properties.Or(nil) == nil {
			continue
		}

		var obj map[string]json.RawMessage
		if err := json.Unmarshal(provided, &obj); err != nil {
			continue
		}

		diagnostics = collectSchemaDiagnostics(diagnostics, newPath, obj, p)
	}

	// collect for additionalProperties objects
	for name, prop := range schema.Properties.OrEmpty() {
		p := prop.OrEmpty()
		additionalProperties := p.AdditionalProperties.Or(nil)
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

			var providedVal any
			if err := json.Unmarshal(provided, &providedVal); err != nil {
				continue
			}

			diagnostics = propTypeDiagnostics(diagnostics, newPath, providedVal, *additionalProperties)
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

		var items []json.RawMessage
		if err := json.Unmarshal(provided, &items); err != nil {
			continue
		}

		for i, item := range items {
			newPath := append(path, name, fmt.Sprintf("%d", i))

			var providedItem any
			if err := json.Unmarshal(item, &providedItem); err != nil {
				continue
			}

			diagnostics = propTypeDiagnostics(diagnostics, newPath, providedItem, *propItems)

			var obj map[string]json.RawMessage
			if err := json.Unmarshal(item, &obj); err != nil {
				continue
			}
			diagnostics = collectSchemaDiagnostics(diagnostics, newPath, obj, *propItems)
		}
	}

	return diagnostics
}

func propTypeDiagnostics(diagnostics []Diagnostic, path []string, providedVal any, prop Schema) []Diagnostic {
	// We will ignore the use of any non-openapi types, because:
	// 1. Keeps the LSP forward compatible.
	// 2. Users sometimes use invalid types like "float" or "map", they're
	// these custom types can't really be dealt with in a consistent way.
	knownOpenAPITypes := []string{"string", "number", "integer", "boolean", "array", "object"}

	propType := prop.Type.OrEmpty()
	oneOfSchemas := prop.OneOf.OrEmpty()

	propTypes := make([]string, 0, len(oneOfSchemas)+1)

	if slices.Contains(knownOpenAPITypes, propType) {
		propTypes = append(propTypes, propType)

	}

	for _, p := range oneOfSchemas {
		t := p.Type.OrEmpty()
		if slices.Contains(knownOpenAPITypes, t) {
			propTypes = append(propTypes, t)
		}
	}

	if len(propTypes) == 0 {
		return diagnostics
	}

	switch v := providedVal.(type) {
	case string:
		if !slices.Contains(propTypes, "string") {
			diagnostics = append(diagnostics, Diagnostic{
				Diagnostic: fmt.Sprintf("Expected %s got string", strings.Join(propTypes, "|")),
				Path:       path,
			})
		}
	case float64:
		if !(slices.Contains(propTypes, "number") || (slices.Contains(propTypes, "integer") && float64(int(v)) == v)) {
			diagnostics = append(diagnostics, Diagnostic{
				Diagnostic: fmt.Sprintf("Expected %s got number", strings.Join(propTypes, "|")),
				Path:       path,
			})
		}
	case int:
		if !(slices.Contains(propTypes, "number") || slices.Contains(propTypes, "integer")) {
			diagnostics = append(diagnostics, Diagnostic{
				Diagnostic: fmt.Sprintf("Expected %s got number", strings.Join(propTypes, "|")),
				Path:       path,
			})
		}
	case bool:
		if !slices.Contains(propTypes, "boolean") {
			diagnostics = append(diagnostics, Diagnostic{
				Diagnostic: fmt.Sprintf("Expected %s got boolean", strings.Join(propTypes, "|")),
				Path:       path,
			})
		}
	case []any:
		if !slices.Contains(propTypes, "array") {
			diagnostics = append(diagnostics, Diagnostic{
				Diagnostic: fmt.Sprintf("Expected %s got array", strings.Join(propTypes, "|")),
				Path:       path,
			})
		}
	case map[string]any:
		if !slices.Contains(propTypes, "object") {
			diagnostics = append(diagnostics, Diagnostic{
				Diagnostic: fmt.Sprintf("Expected %s got object", strings.Join(propTypes, "|")),
				Path:       path,
			})
		}
	case nil:
		if prop.Nullable.Or(false) != true {
			diagnostics = append(diagnostics, Diagnostic{
				Diagnostic: fmt.Sprintf("Expected %s got null", strings.Join(propTypes, "|")),
				Path:       path,
			})
		}
	default:
		diagnostics = append(diagnostics, Diagnostic{
			Diagnostic: fmt.Sprintf("unknown type %T\n", v),
			Path:       path,
		})

	}

	return diagnostics
}
