package openapi_test

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/ethancarlsson/hurl-lsp/expect"
	"github.com/ethancarlsson/hurl-lsp/openapi"
)

func TestParse(t *testing.T) {
	t.Run("yaml", func(t *testing.T) {
		contents, err := os.ReadFile("../fixtures/petstore.yaml")
		expect.NoErr(t, err)

		oai, err := openapi.Parse("yaml", contents)
		expect.NoErr(t, err)
		expect.Equals(t, 13, len(oai.Paths))
	})
}

func TestSchemaCombined(t *testing.T) {
	t.Run("with allOf combines all values and no existing properties", func(t *testing.T) {
		schema := openapi.Schema{
			Ref:        "#/components/schemas/Category",
			Properties: map[string]openapi.Schema{},
			AllOf: []openapi.Schema{
				{
					Type: "object",
					Properties: map[string]openapi.Schema{
						"id": {
							Type: "integer",
						},
						"name": {
							Type: "string",
						},
					},
				},
				{
					Type: "object",
					Properties: map[string]openapi.Schema{
						"rank": {
							Type: "integer",
						},
					},
				},
			},
		}

		actual := schema.Combined(map[string]json.RawMessage{})

		expect.Equals(t, actual.Type, "object")
		expect.Equals(t, actual.Properties, map[string]openapi.Schema{

			"id": {
				Type: "integer",
			},
			"name": {
				Type: "string",
			},
			"rank": {
				Type: "integer",
			},
		})
	})
}

func TestGetOp(t *testing.T) {
	petReqBody := openapi.RequestBody{
		Description: "Create a new pet in the store",
		Content: openapi.RequestBodyContent{
			Json: openapi.BodySchema{
				Schema: openapi.Schema{
					Type:     "object",
					Ref:      "#/components/schemas/Pet",
					Required: []string{"name", "photoUrls"},
					Properties: map[string]openapi.Schema{
						"category": {
							Ref:        "#/components/schemas/Category",
							Properties: map[string]openapi.Schema{},
							AllOf: []openapi.Schema{
								{
									Type: "object",
									Properties: map[string]openapi.Schema{
										"id": {
											Type: "integer",
										},
										"name": {
											Type: "string",
										},
									},
								},
								{
									Type: "object",
									Properties: map[string]openapi.Schema{
										"rank": {
											Type: "integer",
										},
									},
								},
							},
						},
						"id": {
							Type: "integer",
						},
						"name": {
							Type: "string",
						},
						"photoUrls": {
							Type: "array",
							Items: &openapi.Schema{
								Type: "string",
							},
						},
						"status": {
							Type: "string",
						},
						"tags": {
							Type: "array",
							Items: &openapi.Schema{
								Type: "object",
								Ref:  "#/components/schemas/Tag",
								Properties: map[string]openapi.Schema{
									"id": {
										Type: "integer",
									},
									"name": {
										Type: "string",
									},
								},
							},
						},
						"siblings": {
							Type: "object",
							AdditionalProperties: &openapi.Schema{
								Type:     "object",
								Required: []string{"href"},
								Properties: map[string]openapi.Schema{
									"href": {Type: "string"},
								},
							},
						},
					},
				},
			},
		},
	}

	petPutReqBody := petReqBody
	petPutReqBody.Description = "Update an existent pet in the store"
	tests := []struct {
		method, path, expectMethod, expectPath, expectSummary, expectDesc string
		expectRequestBody                                                 openapi.RequestBody
	}{
		{
			method:            "post",
			path:              "/pet",
			expectMethod:      "post",
			expectPath:        "/pet",
			expectSummary:     "Add a new pet to the store.",
			expectDesc:        "Add a new pet to the store.",
			expectRequestBody: petReqBody,
		},
		{
			method:            "put",
			path:              "/pet",
			expectMethod:      "put",
			expectPath:        "/pet",
			expectSummary:     "Update an existing pet.",
			expectDesc:        "Update an existing pet by Id.",
			expectRequestBody: petPutReqBody,
		},
		{
			method:        "get",
			path:          "/pet/findByStatus?status=available",
			expectMethod:  "get",
			expectPath:    "/pet/findByStatus",
			expectSummary: "Finds Pets by status.",
			expectDesc:    "Multiple status values can be provided with comma separated strings.",
		},
		{
			method:        "get",
			path:          "/pet/findByTags",
			expectMethod:  "get",
			expectPath:    "/pet/findByTags",
			expectSummary: "Finds Pets by tags.",
			expectDesc:    "Multiple tags can be provided with comma separated strings. Use tag1, tag2, tag3 for testing.",
		},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("../fixtures/petstore.yaml testing the correctness of %s %s", tt.method, tt.path), func(t *testing.T) {
			contents, err := os.ReadFile("../fixtures/petstore.yaml")
			expect.NoErr(t, err)

			oai, err := openapi.Parse("yaml", contents)

			expect.NoErr(t, err)
			op := oai.GetOp(tt.method, tt.path)
			expect.Equals(t, tt.expectMethod, op.Method)
			expect.Equals(t, tt.expectPath, op.Path)
			expect.Equals(t, tt.expectSummary, op.Detail.Summary)
			expect.Equals(t, tt.expectDesc, op.Detail.Description)
			expectedReq, err := json.Marshal(tt.expectRequestBody)
			expect.NoErr(t, err)
			actualReq, err := json.Marshal(op.Detail.RequestBody)
			expect.Equals(t, string(expectedReq), string(actualReq))
		})
	}
}
