package main

import (
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func documentDidOpen(context *glsp.Context, params *protocol.DidOpenTextDocumentParams) error {
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
