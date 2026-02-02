package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/ethancarlsson/hurl-lsp/completions"
	"github.com/ethancarlsson/hurl-lsp/diagnose"
	"github.com/ethancarlsson/hurl-lsp/hurlfile"
	"github.com/ethancarlsson/hurl-lsp/openapi"
	"github.com/ethancarlsson/hurl-lsp/rawjson"
	"github.com/ethancarlsson/hurl-lsp/signaturehelp"
	"github.com/tliron/commonlog"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
	"github.com/tliron/glsp/server"

	// Must include a backend implementation
	// See CommonLog for other options: https://github.com/tliron/commonlog
	_ "github.com/tliron/commonlog/simple"
)

const lsName = "hurl_ls"

type oaiPath string

func (p oaiPath) Ft() string {
	splitPath := strings.Split(string(p), ".")
	if len(splitPath) == 0 {
		return string(p)
	}

	return splitPath[len(splitPath)-1]
}

type config struct {
	OpenapiDefPaths []oaiPath `json:"openapi_def"`
}

var (
	version string = "0.0.1"
	handler protocol.Handler
	lines   []string           = []string{}
	hf      *hurlfile.HurlFile = &hurlfile.HurlFile{}

	conf config      = config{}
	oai  openapi.OAI = openapi.OAI{}
	errs []error     = []error{}
)

func main() {
	commonlog.Configure(1, nil)

	handler = protocol.Handler{
		Initialize:                initialize,
		Initialized:               initialized,
		Shutdown:                  shutdown,
		SetTrace:                  setTrace,
		TextDocumentCompletion:    completion,
		TextDocumentSignatureHelp: signatureHelp,
		TextDocumentDidOpen:       documentDidOpen,
		TextDocumentDidChange:     documentDidChange,
		TextDocumentDidSave:       documentDidSave,
	}

	server := server.NewServer(&handler, lsName, false)

	server.RunStdio()
}

func documentDidOpen(context *glsp.Context, params *protocol.DidOpenTextDocumentParams) error {
	if err := parseDocument(params.TextDocument.URI); err != nil {
		return err
	}

	runDiagnostics(context, params.TextDocument.URI)

	return nil
}

func documentDidChange(context *glsp.Context, params *protocol.DidChangeTextDocumentParams) error {
	if err := parseDocument(params.TextDocument.URI); err != nil {
		return err
	}
	runDiagnostics(context, params.TextDocument.URI)

	return nil
}

func documentDidSave(context *glsp.Context, params *protocol.DidSaveTextDocumentParams) error {
	parseOpenapi() // reparse on save so users can update their schemas
	if err := parseDocument(params.TextDocument.URI); err != nil {
		return err
	}
	runDiagnostics(context, params.TextDocument.URI)

	return nil
}

func runDiagnostics(context *glsp.Context, documentURI string) {
	diagnostics := make([]protocol.Diagnostic, 0)
	source := lsName
	infoErr := protocol.DiagnosticSeverityInformation
	errorErr := protocol.DiagnosticSeverityError

	for _, entry := range hf.Entries {
		if d := diagnose.HTTPMethod(entry.Request.Method.Name); d.HasProblem {
			diagnostics = append(diagnostics, protocol.Diagnostic{
				Range:    toProtocolRange(entry.Request.Method.Range),
				Message:  d.Message,
				Source:   &source,
				Severity: &errorErr,
			})
		}

		op := oai.GetOp(entry.Request.Method.Name, entry.Request.Target.Target)

		opMethods := oai.GetPathMethods(entry.Request.Target.Target)
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

		if d := diagnose.QueryParams(entry.Request.Target.Target, op.Detail.Parameters.Required()); d.HasProblem {
			diagnostics = append(diagnostics, protocol.Diagnostic{
				Range:    toProtocolRange(entry.Request.Target.Range),
				Message:  d.Message,
				Source:   &source,
				Severity: &errorErr,
			})
		}

		diagnostics = addReqBodyDiagnostics(diagnostics, entry.Request, op)
	}

	context.Notify(protocol.ServerTextDocumentPublishDiagnostics, protocol.PublishDiagnosticsParams{
		URI:         documentURI,
		Diagnostics: diagnostics,
	})
}

func addReqBodyDiagnostics(diagnostics []protocol.Diagnostic, req hurlfile.Request, op openapi.Op) []protocol.Diagnostic {
	schema := op.Detail.RequestBody.Content.Json.Schema
	source := lsName
	errorErr := protocol.DiagnosticSeverityError

	if len(schema.Properties) == 0 {
		return diagnostics
	}

	var alreadyProvidedProps map[string]json.RawMessage
	reqBody := rawjson.CleanTrailingCommas(strings.Join(req.Body.Value, "\n"))
	reqBody = rawjson.CleanVariables(reqBody)

	if err := json.Unmarshal([]byte(reqBody), &alreadyProvidedProps); err != nil {
		return diagnostics
	}

	missingProps := make([][]string, 0, len(schema.Required))

	missingProps = append(missingProps, collectMissingProps(
		missingProps, []string{},
		alreadyProvidedProps, schema,
	)...)

	for _, propPath := range missingProps {
		if len(propPath) == 0 {
			continue
		}
		// path without the last element
		objPath := propPath[:len(propPath)-1]

		pathRange := rawjson.PathToRange(req.Body.Value, objPath)

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
			Message:  fmt.Sprintf("Missing required param %s", strings.Join(propPath, "/")),
			Source:   &source,
			Severity: &errorErr,
		})

	}

	return diagnostics
}

func collectMissingProps(missingProps [][]string, path []string, provided map[string]json.RawMessage, schema openapi.Schema) [][]string {
	for _, required := range schema.Required {
		if _, ok := provided[required]; !ok {
			missingProps = append(missingProps, append(path, required))
		}
	}

	for name, prop := range schema.Properties {
		if prop.Properties == nil {
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
		missingProps = collectMissingProps(missingProps, newPath, obj, prop)
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

func signatureHelp(context *glsp.Context, params *protocol.SignatureHelpParams) (*protocol.SignatureHelp, error) {
	// Reparse on signatureHelp because 1) signatureHelp will not be called as often,
	// and; 2) because we can end up 1 character behind when relying on onChanged
	parseDocument(params.TextDocument.URI)

	line := int(params.Position.Line)
	col := int(params.Position.Character) - 1 // zero base

	sym := signaturehelp.Lines(lines).SymbolAt(line, col)
	if desc := sym.Description(); desc.Desctiption != "" {
		help := protocol.SignatureHelp{Signatures: []protocol.SignatureInformation{
			{
				Label:         sym.String(),
				Documentation: desc.Desctiption,
				Parameters: []protocol.ParameterInformation{
					{Label: desc.Detail.In},
				},
			},
		}}

		return &help, nil
	}

	if hf == nil {
		return nil, nil
	}

	req := hf.GetReq(line, col)
	op := oai.GetOp(req.Method.Name, req.Target.Target)
	if hf.OnMethod(line, col) || hf.OnUri(line, col) {
		help := protocol.SignatureHelp{Signatures: []protocol.SignatureInformation{
			{
				Label: op.Method + " " + op.Path,
				Documentation: fmt.Sprintf(
					"Summary: %s\nDescription: %s",
					op.Detail.Summary, op.Detail.Description,
				),
				Parameters: signaturehelp.ParamsFromMap(op.Detail.Parameters.ToDocMap()),
			},
		},
		}

		return &help, nil
	}

	if hf.OnRequestBody(line, col) {
		help := protocol.SignatureHelp{Signatures: []protocol.SignatureInformation{
			{
				Label: "Request body",
				Documentation: fmt.Sprintf(
					"Description: %s\nSchema: %s",
					op.Detail.RequestBody.Description,
					op.Detail.RequestBody.Content.Json.Schema.ToString(0),
				),
			},
		},
		}

		return &help, nil
	}

	return nil, nil
}

func completion(context *glsp.Context, params *protocol.CompletionParams) (any, error) {
	items := make([]protocol.CompletionItem, 0)
	if hf == nil {
		return items, nil
	}

	line := int(params.Position.Line)
	col := int(params.Position.Character) - 1 // zero base

	if hf.OnRespSectionName(line, col) {
		items = completions.AddRespSection(items)
	}

	if hf.OnEmptyResponse(line, col) {
		items = completions.AddRespVersion(items)
	}

	if caps := hf.Captures().Before(line); len(caps) > 0 {
		items = completions.AddVars(items, caps.Variables())
	}

	if hf.CanUseFilter(line, col) {
		items = completions.AddFilters(items)
	}

	items = addSpecAwareCompletions(items, line, col)

	return items, nil
}

func addSpecAwareCompletions(items []protocol.CompletionItem, line, col int) []protocol.CompletionItem {
	req := hf.GetReq(line, col)
	op := oai.GetOp(req.Method.Name, strings.ReplaceAll(strings.ReplaceAll(req.Target.Target, "{{", "{"), "}}", "}"))

	items = addUriCompletions(items, op, line, col)
	items = addBodyCompletions(items, req, op, line, col)

	return items
}

func addBodyCompletions(items []protocol.CompletionItem, req hurlfile.Request, op openapi.Op, line, col int) []protocol.CompletionItem {
	if !hf.OnRequestBody(line, col) {
		return items
	}

	currentSchema := op.Detail.RequestBody.Content.Json.Schema

	if len(currentSchema.Properties) == 0 {
		return items
	}

	currLine := line - req.Body.Range.StartLine
	if len(req.Body.Value)-1 < currLine {
		return items
	}

	var alreadyProvidedProps map[string]json.RawMessage
	reqBody := rawjson.CleanTrailingCommas(strings.Join(req.Body.Value, "\n"))
	reqBody = rawjson.CleanVariables(reqBody)

	if err := json.Unmarshal([]byte(reqBody), &alreadyProvidedProps); err != nil {
		return items
	}

	schemaPath := rawjson.PathToObj(req.Body.Value, currLine, col)
	for _, path := range schemaPath {
		innerSchema, ok := currentSchema.Properties[path.Name()]
		if !ok {
			currentSchema = openapi.Schema{}
			break
		}
		var innerPropMap map[string]json.RawMessage
		if path.IsArr() {
			var innerPropMapList []map[string]json.RawMessage

			innerJSON, ok := alreadyProvidedProps[path.Name()]

			if ok {
				if err := json.Unmarshal(innerJSON, &innerPropMapList); err != nil {
					break
				}
			}

			if len(innerPropMapList) == 0 {
				break
			}

			innerPropMap = innerPropMapList[0]
		} else {
			innerJSON, ok := alreadyProvidedProps[path.Name()]

			if ok {
				if err := json.Unmarshal(innerJSON, &innerPropMap); err != nil {
					break
				}
			}
		}

		alreadyProvidedProps = innerPropMap
		if path.IsArr() {
			currentSchema = *innerSchema.Items
		} else {
			currentSchema = innerSchema
		}
	}

	isOnLastProp := rawjson.IsOnLastProp(req.Body.Value, currLine, col)

	for name, schema := range currentSchema.Combined(alreadyProvidedProps).Properties {
		if _, isAlreadyThere := alreadyProvidedProps[name]; isAlreadyThere {
			continue
		}

		if isOnLastProp {
			items = completions.AddRequestBodyProperty(items, name, schema.Type, "")
		} else {
			items = completions.AddRequestBodyProperty(items, name, schema.Type, ",")
		}
	}

	return items
}

func addUriCompletions(items []protocol.CompletionItem, op openapi.Op, line, col int) []protocol.CompletionItem {
	if !hf.OnUri(line, col) {
		return items
	}

	if op.NotDocumented() {
		items = completions.AddPaths(items, oai.PathList())

		// nothing else can be done with this if it's not documented
		return items
	} else {
		items = completions.AddPaths(items, oai.ChildPaths(op))
	}

	items = completions.AddParams(items, op.Detail.Parameters.In("query").ToDocMap())

	return items
}

func initialize(context *glsp.Context, params *protocol.InitializeParams) (any, error) {
	capabilities := handler.CreateServerCapabilities()

	return protocol.InitializeResult{
		Capabilities: capabilities,
		ServerInfo: &protocol.InitializeResultServerInfo{
			Name:    lsName,
			Version: &version,
		},
	}, nil
}

func initialized(context *glsp.Context, params *protocol.InitializedParams) error {
	contents, err := os.ReadFile("./.hurl-ls.json")
	if err != nil {
		// do nothing if there's an error because it's not really needed
		return nil
	}

	if err := json.Unmarshal(contents, &conf); err != nil {
		return err
	}

	if len(conf.OpenapiDefPaths) == 0 {
		return nil
	}

	parseOpenapi()

	return nil
}

func parseDocument(uri string) error {
	parsedLines, err := hurlfile.ParseLines(uri)
	if err != nil {
		return fmt.Errorf("Failed to parse the hurl file %w", err)
	}

	lines = parsedLines

	hf, err = hurlfile.Parse(lines)
	if err != nil {
		return fmt.Errorf("Failed to parse the hurl file %w", err)
	}

	return nil
}

func parseOpenapi() {
	combinedOpenapiDef := openapi.New()
	for _, defPath := range conf.OpenapiDefPaths {
		fileContent, err := os.ReadFile(string(defPath))
		if err != nil {
			if m := commonlog.NewErrorMessage(0); m != nil {
				m.Set("_message", "Could not read openapi file").
					Set("err", err).Send()
			}
			errs = append(errs, err)
			return
		}

		openAPI, err := openapi.Parse(defPath.Ft(), fileContent)
		if err != nil {
			if m := commonlog.NewErrorMessage(0); m != nil {
				m.Set("_message", "Could not parse openapi file").
					Set("err", err).Send()
			}
			errs = append(errs, err)
			return
		}

		combinedOpenapiDef = combinedOpenapiDef.Merge(openAPI)
	}

	oai = combinedOpenapiDef
}

func shutdown(context *glsp.Context) error {
	protocol.SetTraceValue(protocol.TraceValueOff)
	return nil
}

func setTrace(context *glsp.Context, params *protocol.SetTraceParams) error {
	protocol.SetTraceValue(params.Value)
	return nil
}

func ptr[T any](v T) *T {
	return &v
}
