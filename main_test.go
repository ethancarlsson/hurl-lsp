package main

import (
	"testing"

	"github.com/ethancarlsson/hurl-lsp/expect"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func TestCompletion(t *testing.T) {
	ctx := glsp.Context{}

	t.Cleanup(func() {
		conf.OpenapiDefPath = ""
	})

	t.Run("no hurlfile", func(t *testing.T) {
		hf = nil
		is, err := completion(&ctx, nil)
		expect.NoErr(t, err)

		items := is.([]protocol.CompletionItem)
		expect.Equals(t, 0, len(items))
	})

	t.Run("resp section", func(t *testing.T) {
		params := &protocol.CompletionParams{
			TextDocumentPositionParams: protocol.TextDocumentPositionParams{
				TextDocument: protocol.TextDocumentIdentifier{
					URI: "./fixtures/test_captures.hurl",
				},
				Position: protocol.Position{
					Line:      2,
					Character: 2,
				},
			},
		}

		parseDocument(params.TextDocument.URI)
		is, err := completion(&ctx, params)
		expect.NoErr(t, err)

		items := is.([]protocol.CompletionItem)
		expect.Equals(t, 2, len(items))
		kind := protocol.CompletionItemKindEnumMember
		insertAssert := "[Asserts]"
		insertCaptures := "[Captures]"
		expect.Equals(t, []protocol.CompletionItem{
			{
				Label:      "Captures",
				Kind:       &kind,
				InsertText: &insertCaptures,
			},
			{
				Label:      "Asserts",
				Kind:       &kind,
				InsertText: &insertAssert,
			},
		}, items)
	})

	t.Run("Captured variables in completions", func(t *testing.T) {
		params := &protocol.CompletionParams{
			TextDocumentPositionParams: protocol.TextDocumentPositionParams{
				TextDocument: protocol.TextDocumentIdentifier{
					URI: "./fixtures/test_captures.hurl",
				},
				Position: protocol.Position{
					Line:      11,
					Character: 2,
				},
			},
		}

		parseDocument(params.TextDocument.URI)
		is, err := completion(&ctx, params)
		expect.NoErr(t, err)

		items := is.([]protocol.CompletionItem)
		expect.Equals(t, 2, len(items))
		expect.Equals(t, "id", items[0].Label)
		expect.Equals(t, "name", items[1].Label)

		kind := protocol.CompletionItemKindVariable
		expect.Equals(t, kind, *items[0].Kind)
		expect.Equals(t, kind, *items[1].Kind)

		expect.Equals(t, "{{id}}", *items[0].InsertText)
		expect.Equals(t, "{{name}}", *items[1].InsertText)

		// name shouldn't be available until it is captured
		params.Position.Line = 6
		is, err = completion(&ctx, params)
		expect.NoErr(t, err)

		items = is.([]protocol.CompletionItem)
		expect.Equals(t, 1, len(items))
		expect.Equals(t, "id", items[0].Label)
	})

	t.Run("uri completions from openapi paths", func(t *testing.T) {
		conf.OpenapiDefPath = "./fixtures/petstore.yaml"
		params := &protocol.CompletionParams{
			TextDocumentPositionParams: protocol.TextDocumentPositionParams{
				TextDocument: protocol.TextDocumentIdentifier{
					URI: "./fixtures/test_captures.hurl",
				},
				Position: protocol.Position{
					Line:      0,
					Character: 20,
				},
			},
		}

		parseOpenapi()
		parseDocument(params.TextDocument.URI)

		is, err := completion(&ctx, params)
		expect.NoErr(t, err)

		items := is.([]protocol.CompletionItem)
		expect.Equals(t, 13, len(items))
		for _, item := range items {
			_, ok := oai.Paths[item.Label]
			expect.Equals(t, true, ok)
		}
	})
}
