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
	tests := []struct {
		method, path, expectMethod, expectPath, expectSummary, expectDesc string
	}{
		{
			method:        "post",
			path:          "/pet",
			expectMethod:  "post",
			expectPath:    "/pet",
			expectSummary: "Add a new pet to the store.",
			expectDesc:    "Add a new pet to the store.",
		},
		{
			method:        "put",
			path:          "/pet",
			expectMethod:  "put",
			expectPath:    "/pet",
			expectSummary: "Update an existing pet.",
			expectDesc:    "Update an existing pet by Id.",
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
		})
	}
}
