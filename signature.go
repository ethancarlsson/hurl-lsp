package main

import (
	"fmt"

	"github.com/ethancarlsson/hurl-lsp/pkg/signaturehelp"
	"github.com/ethancarlsson/hurl-lsp/pkg/state"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func signatureHelp(context *glsp.Context, params *protocol.SignatureHelpParams) (*protocol.SignatureHelp, error) {
	// Reparse on signatureHelp because 1) signatureHelp will not be called as often,
	// and; 2) because we can end up 1 character behind when relying on onChanged
	parseDocument(params.TextDocument.URI)

	line := int(params.Position.Line)
	col := int(params.Position.Character) - 1 // zero base

	lines := state.Lines()
	hf := state.Hf()

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
	op := state.OAI().GetOp(req.Method.Name, req.Target.Target)
	if hf.OnMethod(line, col) || hf.OnUri(line, col) {
		help := protocol.SignatureHelp{Signatures: []protocol.SignatureInformation{
			{
				Label: op.Method + " " + op.Path,
				Documentation: fmt.Sprintf(
					"Summary: %s\nDescription: %s",
					op.Detail.Summary.Or("N/A"), op.Detail.Description.Or("N/A"),
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
