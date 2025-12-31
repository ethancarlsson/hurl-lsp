package openapi_test

import (
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

func TestGetOp(t *testing.T) {
	petReqBody := openapi.RequestBody{
		Description: "Create a new pet in the store",
		Content: openapi.RequestBodyContent{
			Json: openapi.BodySchema{
				Schema: openapi.Schema{
					Type: "object",
					Ref:  "#/components/schemas/Pet",
					Properties: map[string]openapi.Schema{
						"category": {
							Ref:  "#/components/schemas/Category",
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
						"id": {
							Type: "integer",
						},
						"name": {
							Type: "string",
						},
						"photoUrls": {
							Type: "array",
						},
						"status": {
							Type: "string",
						},
						"tags": {
							Type: "array",
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
			expect.Equals(t, tt.expectRequestBody, op.Detail.RequestBody)
		})
	}
}
