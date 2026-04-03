package main

import (
	"encoding/json"
	"os"
	"strings"
	"time"

	"github.com/ethancarlsson/hurl-lsp/pkg/hurlfile"
	"github.com/ethancarlsson/hurl-lsp/pkg/state"
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

	conf config = config{}
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

func documentDidChange(context *glsp.Context, params *protocol.DidChangeTextDocumentParams) error {
	content := strings.Join(state.Lines(), "\n")
	for _, cc := range params.ContentChanges {
		switch v := cc.(type) {
		case protocol.TextDocumentContentChangeEvent:
			start := v.Range.Start.IndexIn(content)
			end := v.Range.End.IndexIn(content)
			content = content[:start] + v.Text + content[end:]

		case protocol.TextDocumentContentChangeEventWhole:
			content = v.Text
		}
	}

	state.SetLines(strings.Split(content, "\n"))

	state.ResetHf(100*time.Millisecond, func(*hurlfile.HurlFile) {
		diagnostics := runDiagnostics()

		context.Notify(protocol.ServerTextDocumentPublishDiagnostics, protocol.PublishDiagnosticsParams{
			URI:         params.TextDocument.URI,
			Diagnostics: diagnostics,
		})
	})

	return nil
}

func documentDidSave(context *glsp.Context, params *protocol.DidSaveTextDocumentParams) error {
	parseOpenapi() // reparse on save so users can update their schemas
	if err := parseDocument(params.TextDocument.URI); err != nil {
		return err
	}

	diagnostics := runDiagnostics()

	context.Notify(protocol.ServerTextDocumentPublishDiagnostics, protocol.PublishDiagnosticsParams{
		URI:         params.TextDocument.URI,
		Diagnostics: diagnostics,
	})

	return nil
}

func initialize(context *glsp.Context, params *protocol.InitializeParams) (any, error) {
	if m, ok := params.InitializationOptions.(map[string]interface{}); ok {
		if defs, ok := m["openapi_def"].([]interface{}); ok {
			oaiPaths := make([]oaiPath, 0, len(defs))
			for _, def := range defs {
				if s, ok := def.(string); ok {
					oaiPaths = append(oaiPaths, oaiPath(s))
				}
			}
			conf.OpenapiDefPaths = oaiPaths
		}
	}

	capabilities := handler.CreateServerCapabilities()
	capabilities.CompletionProvider.TriggerCharacters = []string{`"`, `/`}

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
		// do nothing if there's an error
		contents = []byte("{}")
	}

	var localConf config

	if err := json.Unmarshal(contents, &localConf); err != nil {
		// do nothing
	}

	for _, path := range localConf.OpenapiDefPaths {
		conf.OpenapiDefPaths = append(conf.OpenapiDefPaths, path)
	}

	if len(conf.OpenapiDefPaths) == 0 {
		return nil
	}

	deduppedPaths := make(map[oaiPath]struct{})
	for _, p := range conf.OpenapiDefPaths {
		deduppedPaths[p] = struct{}{}
	}
	conf.OpenapiDefPaths = make([]oaiPath, 0, len(deduppedPaths))
	for p := range deduppedPaths {
		conf.OpenapiDefPaths = append(conf.OpenapiDefPaths, p)
	}

	parseOpenapi()

	return nil
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
