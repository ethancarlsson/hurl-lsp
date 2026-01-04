package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/ethancarlsson/hurl-lsp/completions"
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
	}

	server := server.NewServer(&handler, lsName, false)

	server.RunStdio()
}

func documentDidOpen(context *glsp.Context, params *protocol.DidOpenTextDocumentParams) error {
	if err := parseDocument(params.TextDocument.URI); err != nil {
		return err
	}

	return nil
}

func documentDidChange(context *glsp.Context, params *protocol.DidChangeTextDocumentParams) error {
	if err := parseDocument(params.TextDocument.URI); err != nil {
		return err
	}

	return nil
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

	props := op.Detail.RequestBody.Content.Json.Schema.Properties

	if len(props) == 0 {
		return items
	}

	currLine := line - req.Body.Range.StartLine
	if len(req.Body.Value)-1 < currLine {
		return items
	}

	var propsMap map[string]json.RawMessage
	reqBody := rawjson.CleanTrailingCommas(strings.Join(req.Body.Value, "\n"))
	reqBody = rawjson.CleanVariables(reqBody)

	if err := json.Unmarshal([]byte(reqBody), &propsMap); err != nil {
		return items
	}

	schemaPath := rawjson.PathToObj(req.Body.Value, currLine, col)
	for _, path := range schemaPath {
		innerSchema, ok := props[path.Name()]
		if !ok {
			props = make(map[string]openapi.Schema, 0)
			break
		}
		var innerPropMap map[string]json.RawMessage
		if path.IsArr() {
			var innerPropMapList []map[string]json.RawMessage

			innerJSON, ok := propsMap[path.Name()]

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
			innerJSON, ok := propsMap[path.Name()]

			if ok {
				if err := json.Unmarshal(innerJSON, &innerPropMap); err != nil {
					break
				}
			}
		}

		propsMap = innerPropMap
		if path.IsArr() {
			props = innerSchema.Items.Properties
		} else {
			props = innerSchema.Properties
		}
	}

	isOnLastProp := rawjson.IsOnLastProp(req.Body.Value, currLine, col)

	for name, schema := range props {
		if _, isAlreadyThere := propsMap[name]; isAlreadyThere {
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
