package main

import (
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"strings"
	"unicode"

	"github.com/ethancarlsson/hurl-lsp/completions"
	"github.com/ethancarlsson/hurl-lsp/hurlfile"
	"github.com/ethancarlsson/hurl-lsp/openapi"
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

	if hf.OnMethod(line, col) || hf.OnUri(line, col) {
		req := hf.GetReq(line, col)
		op := oai.GetOp(req.Method.Name, req.Target.Target)
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
	reqBody := cleanTrailingCommas(strings.Join(req.Body.Value, "\n"))

	if err := json.Unmarshal([]byte(reqBody), &propsMap); err != nil {
		return items
	}

	schemaPath := pathToObj(req.Body.Value, currLine, col)
	for _, path := range schemaPath {
		innerSchema, ok := props[path]
		if !ok {
			props = make(map[string]openapi.Schema, 0)
			break
		}
		var innerPropMap map[string]json.RawMessage
		innerJSON, ok := propsMap[path]
		if ok {
			if err := json.Unmarshal(innerJSON, &propsMap); err != nil {
				break
			}
		}

		propsMap = innerPropMap
		props = innerSchema.Properties
	}

	for name, schema := range props {
		if _, isAlreadyThere := propsMap[name]; isAlreadyThere {
			continue
		}

		items = completions.AddRequestBodyProperty(items, name, schema.Type)
	}

	return items
}

func pathToObj(lines []string, currLine, currCol int) []string {
	path := make([]string, 0)

	expectedPathLen := 0

	for i := currLine; i < len(lines); i++ {
		for j := currCol; j < len(lines[i]); j++ {
			if lines[i][j] == '}' {
				expectedPathLen += 1
			} else if lines[i][j] == '{' {
				expectedPathLen -= 1
			}
		}

		// not on last line
		if i < len(lines)-1 {
			currCol = 0
		}
	}

	// used to keep track of how many of '}' character has been seen since the
	// last '{' character
	endBraceCount := 0
	for i := currLine; i >= 0; i-- {
		lookingForObjName := false
		objPathName := make([]byte, 0)

		for j := currCol; j > 0; j-- {
			currChar := lines[i][j]

			if currChar == '}' && !lookingForObjName {
				endBraceCount += 1
				continue
			}

			if currChar == '{' {
				lookingForObjName = endBraceCount == 0
				endBraceCount -= 1
				continue
			}

			if lookingForObjName && (unicode.IsSpace(rune(currChar)) || string(currChar) == ":" || string(currChar) == "\"") {
				continue
			}

			if lookingForObjName && (unicode.IsLetter(rune(currChar)) || unicode.IsNumber(rune(currChar))) {
				objPathName = append(objPathName, currChar)
				continue
			}

			lookingForObjName = false
		}

		if i > 0 {
			currCol = len(lines[i-1]) - 1
		}

		if len(objPathName) > 0 {
			slices.Reverse(objPathName)
			path = append(path, string(objPathName))
			endBraceCount = 0
		}
	}

	slices.Reverse(path)
	return path
}

// This goes through and removes any trailing commas in a JSON string. This allows
// the tool to work with partially broken JSON which is very common when a user might
// be adding a new property to the object or array.
func cleanTrailingCommas(jsonStr string) string {
	lookingForComma := false
	splitStr := strings.Split(jsonStr, "")
	for i := len(splitStr) - 1; i >= 0; i-- {
		if splitStr[i] == "}" || splitStr[i] == "]" {
			lookingForComma = true
			continue
		}

		if unicode.IsSpace(rune(jsonStr[i])) {
			continue
		}

		if lookingForComma == true && splitStr[i] == "," {
			splitStr[i] = ""
		}

		lookingForComma = false
	}

	return strings.Join(splitStr, "")
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
