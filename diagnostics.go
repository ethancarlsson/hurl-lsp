package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/ethancarlsson/hurl-lsp/pkg/diagnose"
	"github.com/ethancarlsson/hurl-lsp/pkg/hurlfile"
	"github.com/ethancarlsson/hurl-lsp/pkg/openapi"
	"github.com/ethancarlsson/hurl-lsp/pkg/rawjson"
	"github.com/ethancarlsson/hurl-lsp/pkg/state"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func runDiagnostics() []protocol.Diagnostic {
	diagnostics := make([]protocol.Diagnostic, 0)
	source := lsName
	infoErr := protocol.DiagnosticSeverityInformation
	errorErr := protocol.DiagnosticSeverityError

	for _, entry := range state.Hf().Entries {
		if d := diagnose.HTTPMethod(entry.Request.Method.Name); d.HasProblem {
			diagnostics = append(diagnostics, protocol.Diagnostic{
				Range:    toProtocolRange(entry.Request.Method.Range),
				Message:  d.Message,
				Source:   &source,
				Severity: &errorErr,
			})
		}

		op := state.OAI().GetOp(entry.Request.Method.Name, entry.Request.Target.Target)

		opMethods := state.OAI().GetPathMethods(entry.Request.Target.Target)
		if op.NotDocumented() && len(opMethods) > 0 {
			diagnostics = append(diagnostics, protocol.Diagnostic{
				Range: toProtocolRange(entry.Request.Method.Range),
				Message: fmt.Sprintf(
					"Only the following methods are documented for this URI %s.",
					strings.ToUpper(strings.Join(opMethods, ", ")),
				),
				Source:   &source,
				Severity: &infoErr,
			})
		}

		if d := diagnose.QueryParams(entry.Request.Target.Target, op.Detail.Parameters.Required("query")); d.HasProblem {
			diagnostics = append(diagnostics, protocol.Diagnostic{
				Range:    toProtocolRange(entry.Request.Target.Range),
				Message:  d.Message,
				Source:   &source,
				Severity: &errorErr,
			})
		}

		diagnostics = addReqBodyDiagnostics(diagnostics, entry.Request, op)
	}

	return diagnostics
}

func addReqBodyDiagnostics(diagnostics []protocol.Diagnostic, req hurlfile.Request, op openapi.Op) []protocol.Diagnostic {
	schema := op.Detail.RequestBody.Content.Json.Schema
	source := lsName
	errorErr := protocol.DiagnosticSeverityError

	bodyLines := slices.Clone(req.Body.Value)
	reqBody := strings.Join(bodyLines, "\n")

	if strings.TrimSpace(reqBody) == "" {
		return diagnostics
	}

	if err := json.Unmarshal([]byte(reqBody), &struct{}{}); err != nil {
		var syntaxErr *json.SyntaxError
		isSyntax := errors.As(err, &syntaxErr)

		if isSyntax {
			jsonRange := rawjson.OffsetToPos(bodyLines, syntaxErr.Offset)
			jsonRange.Start.Line += req.Body.Range.StartLine // push down to start of reqBody
			jsonRange.End.Line += req.Body.Range.StartLine   // push down to start of reqBody
			diagnostics = append(diagnostics, protocol.Diagnostic{
				Range:    jsonRangeToProtocolRange(jsonRange),
				Message:  fmt.Sprintf("Could not parse JSON %s", syntaxErr.Error()),
				Source:   &source,
				Severity: &errorErr,
			})
		}
	}

	if len(schema.Properties.OrEmpty()) == 0 {
		return diagnostics
	}

	var alreadyProvidedProps map[string]json.RawMessage

	reqBody = rawjson.CleanTrailingCommas(reqBody)
	reqBody = rawjson.CleanVariables(reqBody)

	if err := json.Unmarshal([]byte(reqBody), &alreadyProvidedProps); err != nil {
		return diagnostics
	}

	schemaDiagnostics := openapi.DiagnoseAgainstSchema(
		alreadyProvidedProps, schema,
	)

	for _, diagnostic := range schemaDiagnostics {
		if len(diagnostic.Path) == 0 {
			continue
		}
		// path without the last element
		objPath := diagnostic.Path[:len(diagnostic.Path)-1]

		pathRange, err := rawjson.PathToRange(bodyLines, objPath)
		additionalPathErr := ""
		if err != nil {
			additionalPathErr = fmt.Sprintf(". [note] could not read whole JSON: %s", err.Error())
		}

		diagnostics = append(diagnostics, protocol.Diagnostic{
			Range: protocol.Range{
				Start: protocol.Position{
					Line:      protocol.UInteger(req.Body.Range.StartLine + pathRange.Start.Line),
					Character: protocol.UInteger(pathRange.Start.Col),
				},
				End: protocol.Position{
					Line:      protocol.UInteger(req.Body.Range.StartLine + pathRange.End.Line),
					Character: protocol.UInteger(pathRange.End.Col),
				},
			},
			Message:  fmt.Sprintf(`%s%s`, diagnostic.Error(), additionalPathErr),
			Source:   &source,
			Severity: &errorErr,
		})

	}

	return diagnostics
}

func collectMissingProps(missingProps [][]string, path []string, provided map[string]json.RawMessage, schema openapi.Schema) [][]string {
	for _, required := range schema.Required.OrEmpty() {
		if _, ok := provided[required]; !ok {
			missingProps = append(missingProps, append(path, required))
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

		var obj map[string]json.RawMessage
		if err := json.Unmarshal(provided, &obj); err != nil {
			continue
		}

		newPath := append(path, name)
		missingProps = collectMissingProps(missingProps, newPath, obj, p)
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
			missingProps = collectMissingProps(missingProps, newPath, obj, *additionalProperties)
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
			missingProps = collectMissingProps(missingProps, newPath, obj, *propItems)
		}
	}

	return missingProps
}

func toProtocolRange(srcRange hurlfile.SourceRange) protocol.Range {
	return protocol.Range{
		Start: protocol.Position{
			Line:      protocol.UInteger(srcRange.StartLine),
			Character: protocol.UInteger(srcRange.StartCol),
		},
		End: protocol.Position{
			Line:      protocol.UInteger(srcRange.EndLine),
			Character: protocol.UInteger(srcRange.EndCol),
		},
	}
}

// jsonPosToProtocolRange pads front and back cols to make lines the error more obvious
func jsonRangeToProtocolRange(r rawjson.Range) protocol.Range {
	return protocol.Range{
		Start: protocol.Position{
			Line:      protocol.UInteger(r.Start.Line),
			Character: protocol.UInteger(r.Start.Col),
		},
		End: protocol.Position{
			Line:      protocol.UInteger(r.End.Line),
			Character: protocol.UInteger(r.End.Col),
		},
	}
}
