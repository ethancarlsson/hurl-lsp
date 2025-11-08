package main

import (
	"fmt"

	"github.com/ethancarlsson/hurl-lsp/completions"
	"github.com/ethancarlsson/hurl-lsp/hurlfile"
	"github.com/tliron/commonlog"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
	"github.com/tliron/glsp/server"

	// Must include a backend implementation
	// See CommonLog for other options: https://github.com/tliron/commonlog
	_ "github.com/tliron/commonlog/simple"
)

const lsName = "hurl_ls"

var (
	version string = "0.0.1"
	handler protocol.Handler
)

func main() {
	commonlog.Configure(1, nil)

	handler = protocol.Handler{
		Initialize:             initialize,
		Initialized:            initialized,
		Shutdown:               shutdown,
		SetTrace:               setTrace,
		TextDocumentCompletion: completion,
	}

	server := server.NewServer(&handler, lsName, false)

	server.RunStdio()
}

func completion(context *glsp.Context, params *protocol.CompletionParams) (any, error) {
	hf, err := hurlfile.Parse(params.TextDocument.URI)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse the hurl file %w", err)
	}

	items := make([]protocol.CompletionItem, 0)
	line := int(params.Position.Line)
	col := int(params.Position.Character) - 1 // zero base

	if hf.OnMethod(line, col) {
		items = completions.AddMethod(items)
	}

	if hf.OnRespSection(line, col) {
		items = completions.AddRespSection(items)
	}

	if caps := hf.Captures().Before(line); len(caps) > 0 {
		items = completions.AddVars(items, caps.Variables())
	}

	return items, nil
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
