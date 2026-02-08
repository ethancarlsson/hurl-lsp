package main

import (
	"slices"
	"strings"
	"testing"

	"github.com/ethancarlsson/hurl-lsp/expect"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func TestCompletion(t *testing.T) {
	ctx := glsp.Context{}

	t.Cleanup(func() {
		conf.OpenapiDefPaths = []oaiPath{}
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
		conf.OpenapiDefPaths = []oaiPath{"./fixtures/petstore.yaml"}
		params := &protocol.CompletionParams{
			TextDocumentPositionParams: protocol.TextDocumentPositionParams{
				TextDocument: protocol.TextDocumentIdentifier{
					URI: "./fixtures/test_captures.hurl",
				},
				Position: protocol.Position{
					Line:      15,
					Character: 12,
				},
			},
		}

		parseOpenapi()
		parseDocument(params.TextDocument.URI)

		is, err := completion(&ctx, params)
		expect.NoErr(t, err)

		items := is.([]protocol.CompletionItem)
		// 13 paths + the 2 captured variables
		expect.Equals(t, 15, len(items))
		for _, item := range items {
			if item.Label == "id" || item.Label == "name" {
				// Don't test the captured variables
				continue
			}
			_, ok := oai.Paths[item.Label]
			expect.Equals(t, true, ok)
		}
	})

	t.Run("request body completions", func(t *testing.T) {
		conf.OpenapiDefPaths = []oaiPath{"./fixtures/petstore.yaml"}
		params := &protocol.CompletionParams{
			TextDocumentPositionParams: protocol.TextDocumentPositionParams{
				TextDocument: protocol.TextDocumentIdentifier{
					URI: "./fixtures/test_request_body.hurl",
				},
				Position: protocol.Position{
					Line:      6,
					Character: 1,
				},
			},
		}
		// {
		// 	"name": "Fido",
		// 	"category": {
		//
		// 	},
		//	<- HERE
		// 	"tags": [
		// 		{
		//
		//		"name": "test"
		// 		}
		// 	]
		// }
		//

		parseOpenapi()
		parseDocument(params.TextDocument.URI)

		is, err := completion(&ctx, params)
		expect.NoErr(t, err)

		items := is.([]protocol.CompletionItem)
		expectedPropertyNames := []string{"id", "photoUrls",
			"status",
		}
		expect.Equals(t, len(expectedPropertyNames), len(items))
		itemNames := make([]string, 0, len(expectedPropertyNames))
		for _, item := range items {
			itemNames = append(itemNames, strings.Trim(item.Label, "\""))
		}

		slices.Sort(itemNames)

		expect.Equals(t, expectedPropertyNames, itemNames)
	})

	t.Run("request body completion inner object", func(t *testing.T) {
		conf.OpenapiDefPaths = []oaiPath{"./fixtures/petstore.yaml"}
		params := &protocol.CompletionParams{
			TextDocumentPositionParams: protocol.TextDocumentPositionParams{
				TextDocument: protocol.TextDocumentIdentifier{
					URI: "./fixtures/test_request_body.hurl",
				},
				Position: protocol.Position{
					Line:      4,
					Character: 1,
				},
			},
		}
		// {
		// 	"name": "Fido",
		// 	"category": {
		//	<- HERE
		// 	},
		//
		// 	"tags": [
		// 		{
		//
		//		"name": "test"
		// 		}
		// 	]
		// }
		//

		parseOpenapi()
		parseDocument(params.TextDocument.URI)

		is, err := completion(&ctx, params)
		expect.NoErr(t, err)

		items := is.([]protocol.CompletionItem)
		expectedPropertyNames := []string{"id", "name", "rank"}
		itemNames := make([]string, 0, len(expectedPropertyNames))
		for _, item := range items {
			itemNames = append(itemNames, strings.Trim(item.Label, "\""))
		}

		slices.Sort(itemNames)

		expect.Equals(t, expectedPropertyNames, itemNames)
	})

	t.Run("request body completion inner list object", func(t *testing.T) {
		conf.OpenapiDefPaths = []oaiPath{"./fixtures/petstore.yaml"}
		params := &protocol.CompletionParams{
			TextDocumentPositionParams: protocol.TextDocumentPositionParams{
				TextDocument: protocol.TextDocumentIdentifier{
					URI: "./fixtures/test_request_body.hurl",
				},
				Position: protocol.Position{
					Line:      9,
					Character: 1,
				},
			},
		}
		// {
		// 	"name": "Fido",
		// 	"category": {
		//
		// 	},
		//
		// 	"tags": [
		// 		{
		//	<- HERE
		//		"name": "test"
		// 		}
		// 	]
		// }

		parseOpenapi()
		parseDocument(params.TextDocument.URI)

		is, err := completion(&ctx, params)
		expect.NoErr(t, err)

		items := is.([]protocol.CompletionItem)
		expectedPropertyNames := []string{"id"}
		itemNames := make([]string, 0, len(expectedPropertyNames))
		for _, item := range items {
			itemNames = append(itemNames, strings.Trim(item.Label, "\""))
		}

		slices.Sort(itemNames)

		expect.Equals(t, expectedPropertyNames, itemNames)
	})

	t.Run("request body completion additionalProperties", func(t *testing.T) {
		conf.OpenapiDefPaths = []oaiPath{"./fixtures/petstore.yaml"}
		params := &protocol.CompletionParams{
			TextDocumentPositionParams: protocol.TextDocumentPositionParams{
				TextDocument: protocol.TextDocumentIdentifier{
					URI: "./fixtures/test_request_body.hurl",
				},
				Position: protocol.Position{
					Line:      16,
					Character: 3,
				},
			},
		}
		// {
		// 	"name": "Fido",
		// 	"category": {
		//
		// 	},
		//
		// 	"tags": [
		// 		{
		//
		//		"name": "test"
		// 		}
		// 	],
		//	"siblings": {
		//		"id123": {
		//			<- HERE
		//		}
		//	}
		// }
		//

		parseOpenapi()
		parseDocument(params.TextDocument.URI)

		is, err := completion(&ctx, params)
		expect.NoErr(t, err)

		items := is.([]protocol.CompletionItem)
		expectedPropertyNames := []string{"href"}
		itemNames := make([]string, 0, len(expectedPropertyNames))
		for _, item := range items {
			itemNames = append(itemNames, strings.Trim(item.Label, "\""))
		}

		slices.Sort(itemNames)

		expect.Equals(t, expectedPropertyNames, itemNames)
	})
}

func TestDiagnostics(t *testing.T) {
	t.Cleanup(func() {
		conf.OpenapiDefPaths = []oaiPath{}
	})

	tests := []struct {
		filePath string
		expected []protocol.Diagnostic
	}{
		{
			filePath: "./fixtures/test_captures.hurl",
			expected: []protocol.Diagnostic{
				{
					Range: protocol.Range{
						Start: protocol.Position{
							Line:      11,
							Character: 0,
						},
						End: protocol.Position{
							Line:      13,
							Character: 0,
						},
					},
					Message: `Missing required property "name"`,
				},
				{
					Range: protocol.Range{
						Start: protocol.Position{
							Line:      11,
							Character: 0,
						},
						End: protocol.Position{
							Line:      13,
							Character: 0,
						},
					},
					Message: `Missing required property "photoUrls"`,
				},
			},
		},
		{
			filePath: "./fixtures/test_request_body.hurl",
			expected: []protocol.Diagnostic{
				{
					Range: protocol.Range{
						Start: protocol.Position{
							Line:      1,
							Character: 0,
						},
						End: protocol.Position{
							Line:      18,
							Character: 0,
						},
					},
					Message: `Missing required property "photoUrls"`,
				},
				{
					Range: protocol.Range{
						Start: protocol.Position{
							Line:      14,
							Character: 2,
						},
						End: protocol.Position{
							Line:      16,
							Character: 2,
						},
					},
					Message: `Missing required property "siblings/id123/href"`,
				},
				{
					Range: protocol.Range{
						Start: protocol.Position{
							Line:      8,
							Character: 2,
						},
						End: protocol.Position{
							Line:      11,
							Character: 2,
						},
					},
					Message: `Missing required property "tags/0/id"`,
				},
			},
		},
	}

	ignorePointers := func(p []protocol.Diagnostic) []protocol.Diagnostic {
		ret := make([]protocol.Diagnostic, 0, len(p))
		for _, d := range p {
			d.Source = nil
			d.Severity = nil

			ret = append(ret, d)
		}

		return ret
	}

	for _, tc := range tests {
		t.Run("diagnostics for "+tc.filePath, func(t *testing.T) {
			conf.OpenapiDefPaths = []oaiPath{"./fixtures/petstore.yaml"}
			parseOpenapi()
			parseDocument(tc.filePath)

			expect.Equals(t, tc.expected, ignorePointers(runDiagnostics()))
		})
	}
}
