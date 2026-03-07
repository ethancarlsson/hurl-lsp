package main

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/ethancarlsson/hurl-lsp/pkg/expect"
	"github.com/ethancarlsson/hurl-lsp/pkg/openapi"
	"github.com/ethancarlsson/hurl-lsp/pkg/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
	"pgregory.net/rapid"
)

func TestCompletion(t *testing.T) {
	ctx := glsp.Context{}

	t.Cleanup(func() {
		conf.OpenapiDefPaths = []oaiPath{}
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
		runDiagnostics()
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
		runDiagnostics()
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
			_, ok := state.OAI().Paths[item.Label]
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
		runDiagnostics()
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
		runDiagnostics()

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
		runDiagnostics()

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
		runDiagnostics()

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

	ignorePointers := func(p []protocol.Diagnostic) []protocol.Diagnostic {
		ret := make([]protocol.Diagnostic, 0, len(p))
		for _, d := range p {
			d.Source = nil
			d.Severity = nil

			ret = append(ret, d)
		}

		return ret
	}

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
							Character: 1,
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
							Character: 1,
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
							Character: 1,
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
							Character: 3,
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
							Character: 3,
						},
					},
					Message: `Missing required property "tags/0/id"`,
				},
			},
		},
		{
			filePath: "./fixtures/test_partial_req.hurl",
			expected: []protocol.Diagnostic{
				{
					Range: protocol.Range{
						Start: protocol.Position{
							Line:      4,
							Character: 11,
						},
						End: protocol.Position{
							Line:      4,
							Character: 16,
						},
					},
					Message: `Could not parse JSON invalid character '"' after object key`,
				},
			},
		},
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

type nestedStruct struct {
	ID    int               `json:"id"`
	Tags  []string          `json:"tags"`
	Attrs map[string]string `json:"attrs"`
}

type deepNested struct {
	Matrix   [][]int                    `json:"matrix"`
	Lookup   map[string][]*nestedStruct `json:"lookup"`
	Optional *nestedStruct              `json:"optional"`
}

type structOfEverything struct {
	// Basic maps
	IntMap  map[string]int                 `json:"int_map"`
	StrMap  map[string]string              `json:"str_map"`
	MapMap  map[string]map[int]int         `json:"map_map"`
	ArrMap  map[string][]map[string]string `json:"arr_map"`
	BoolMap map[string]bool                `json:"bool_map"`

	// Basic values
	Str   string  `json:"str"`
	Int   int     `json:"int"`
	Float float64 `json:"float"`
	Bool  bool    `json:"bool"`

	// Deep nesting
	Deep deepNested `json:"deep"`
}

func TestProperties(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		var (
			name     = rapid.String().Draw(t, "name")
			category = rapid.Make[structOfEverything]().Draw(t, "category")
			tags     = rapid.SliceOf(rapid.MapOf(rapid.String(), rapid.String())).Draw(t, "tags")
			siblings = rapid.MapOf(rapid.String(), rapid.MapOf(rapid.String(), rapid.String())).Draw(t, "siblings")
		)
		categoryJSON, _ := json.Marshal(category)
		tagsJSON, _ := json.Marshal(tags)
		siblingsJSON, _ := json.Marshal(siblings)
		contents := fmt.Sprintf(`POST {{url}}/pet
{

	"name": "%s",
	"category": %s,

	"tags": %s,
	"siblings": %s
}`, name, string(categoryJSON), string(tagsJSON), string(siblingsJSON))
		oaiContents := `
{
  "paths": {
    "/pet": {
      "post": {
        "tags": [
          "pet"
        ],
        "summary": "add a new pet to the store.",
        "description": "add a new pet to the store.",
        "operationid": "addpet",
        "requestbody": {
          "description": "create a new pet in the store",
          "content": {
            "application/json": {
              "schema": {
                "$ref": "#/components/schemas/pet"
              }
            }
          },
          "required": true
        }
      }
    }
  },
  "components": {
    "schemas": {
      "pet": {
        "required": [
          "name",
          "photourls"
        ],
        "type": "object",
        "properties": {
          "id": "invalid property schema",
          "name": {
            "type": "string",
            "example": "doggie"
          },
          "category": {
            "$ref": "#/components/schemas/category"
          },
          "photourls": {
            "type": "array",
            "xml": {
              "wrapped": true
            },
            "items": {
              "type": "string",
              "xml": {
                "name": "photourl"
              }
            }
          },
          "tags": {
            "type": "array",
            "xml": {
              "wrapped": true
            },
            "items": {
              "$ref": "#/components/schemas/tag"
            }
          },
          "status": {
            "type": "string",
            "description": "pet status in the store",
            "enum": [
              "available",
              "pending",
              "sold"
            ]
          },
          "siblings": {
            "type": "object",
            "additionalproperties": {
              "type": "object",
              "required": [
                "href"
              ],
              "properties": {
                "href": {
                  "type": "string"
                }
              }
            }
          }
        }
      }
    }
  }
}`

		t.Log("Hurl file contents:", contents)

		var err error
		oai, err := openapi.Parse("json", []byte(oaiContents))
		state.SetOAI(&oai)

		require.NoError(t, err)

		lines := strings.Split(contents, "\n")
		state.SetLines(lines)
		state.SetHfFromLines(lines)
		completions, err := completion(&glsp.Context{}, &protocol.CompletionParams{
			TextDocumentPositionParams: protocol.TextDocumentPositionParams{
				Position: protocol.Position{
					Line:      2,
					Character: 1,
				},
			}})

		require.NoError(t, err)
		assert.IsType(t, completions, []protocol.CompletionItem{})
		_, ok := completions.([]protocol.CompletionItem)
		assert.True(t, ok)

		runDiagnostics()
	})
}
