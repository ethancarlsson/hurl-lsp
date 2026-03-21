package openapi_test

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/ethancarlsson/hurl-lsp/pkg/expect"
	"github.com/ethancarlsson/hurl-lsp/pkg/openapi"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	expectedPetSchema := openapi.Schema{
		Required: openapi.MParsedV([]string{"name", "photoUrls"}),
		Type:     openapi.MParsedS("object"),
		Properties: openapi.MParsedV(map[string]openapi.MaybeParsed[openapi.Schema]{
			"id": openapi.MParsedV(openapi.Schema{}),
			"name": openapi.MParsedV(openapi.Schema{
				Type: openapi.MParsedS("string"),
			}),
		}),
	}
	t.Run("yaml", func(t *testing.T) {
		contents, err := os.ReadFile("../../fixtures/petstore.yaml")
		expect.NoErr(t, err)

		oai, err := openapi.Parse("yaml", contents)
		expect.NoErr(t, err)
		expect.Equals(t, 13, len(oai.Paths))
	})

	tests := []struct {
		name        string
		in          string
		expected    openapi.OAI
		expectedOps []openapi.Op
	}{
		{
			name: "empty",
			in:   ``,
		},
		{
			name: "op not a struct",
			in: `
paths:
  /pet: this op is a string`,
			expectedOps: []openapi.Op{
				{
					Method: "put",
					Path:   "/pet",
					Detail: openapi.OpDetail{
						Summary:     openapi.MParsedS("Operation not documented"),
						Description: openapi.MParsedS("Documentation is malformed json/yaml"),
					},
				},
			},
		},
		{
			name: "invalid description type",
			in: `
paths:
  /pet/findByStatus:
    get:
      summary: Finds Pets by status.
      description: 
        - an array for some reason
      operationId: findPetsByStatus
      responses:
        '200':
          description: successful operation
          content:
            application/json:
              schema:
                type: array
                items:
                  $ref: '#/components/schemas/Pet'
            application/xml:
              schema:
                type: array
                items:
                  $ref: '#/components/schemas/Pet'
        '400':
          description: Invalid status value
        default:
          description: Unexpected error
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/Error"`,
			expectedOps: []openapi.Op{
				{
					Method: "get",
					Path:   "/pet/findByStatus",
					Detail: openapi.OpDetail{
						Summary:     openapi.MParsedS("Finds Pets by status."),
						Description: openapi.MParsedS("ERROR"),
					},
				},
			},
		},
		{
			name: "invalid property type does not break on reading other props",
			in: `
paths:
  /pet:
    post:
      tags:
        - pet
      summary: Add a new pet to the store.
      description: Add a new pet to the store.
      operationid: addPet
      requestbody:
        description: create a new pet in the store
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/pet'
        required: true
components:
  schemas:
    pet:
      required:
        - name
        - photoUrls
      type: object
      properties:
        id: invalid property schema
        name:
          type: string
          example: doggie
        category:
          $ref: '#/components/schemas/category'
        photoUrls:
          type: array
          xml:
            wrapped: true
          items:
            type: string
            xml:
              name: photoUrl
        tags:
          type: array
          xml:
            wrapped: true
          items:
            $ref: '#/components/schemas/tag'
        status:
          type: string
          description: pet status in the store
          enum:
            - available
            - pending
            - sold
        siblings:
          type: object
          additionalproperties:
            type: object
            required:
              - href
            properties:
              href:
                type: string`,
			expectedOps: []openapi.Op{
				{
					Method: "post",
					Path:   "/pet",
					Detail: openapi.OpDetail{
						Summary:     openapi.MParsedS("Add a new pet to the store."),
						Description: openapi.MParsedS("Add a new pet to the store."),
						RequestBody: openapi.RequestBody{
							Content: openapi.RequestBodyContent{
								Json: openapi.BodySchema{
									Schema: expectedPetSchema,
								},
							},
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("partially valid values can be read - %s", tt.name), func(t *testing.T) {
			oai, err := openapi.Parse("yaml", []byte(tt.in))
			require.NoError(t, err)
			for _, expectedOp := range tt.expectedOps {
				op := oai.GetOp(expectedOp.Method, expectedOp.Path)
				assert.Equal(t, expectedOp.Method, op.Method)
				assert.Equal(t, expectedOp.Path, op.Path)
				assert.Equal(t, expectedOp.Detail.Description.Or("ERROR"), op.Detail.Description.Or("ERROR"))
				sum, err := expectedOp.Detail.Summary.Read()
				require.NoError(t, err)
				actualSum, err := op.Detail.Summary.Read()
				assert.Equal(t, sum, actualSum)
				expectedSchema := expectedOp.Detail.RequestBody.Content.Json.Schema
				actualSchema := op.Detail.RequestBody.Content.Json.Schema
				assert.Equal(t, expectedSchema.Required, actualSchema.Required)

				actualProps := actualSchema.Properties.Or(map[string]openapi.MaybeParsed[openapi.Schema]{})
				expectedProps := expectedSchema.Properties.Or(map[string]openapi.MaybeParsed[openapi.Schema]{})
				for k, expectedProp := range expectedProps {
					actualProp, ok := actualProps[k]
					assert.True(t, ok, `expected prop "%s" but not found in %v`, k, actualProps)
					p, err := actualProp.Read()
					expectedP, expectedErr := expectedProp.Read()
					assert.Equal(t, expectedP.Type.OrEmpty(), p.Type.Or(""))
					assert.Equal(t, err, expectedErr)
				}
			}
		})
	}
}

func TestSchemaCombined(t *testing.T) {
	t.Run("with allOf combines all values and no existing properties", func(t *testing.T) {
		schema := openapi.Schema{
			Ref:        openapi.MParsedS("#/components/schemas/Category"),
			Properties: openapi.MParsedV(map[string]openapi.MaybeParsed[openapi.Schema]{}),
			AllOf: openapi.MParsedV([]openapi.Schema{
				{
					Type: openapi.MParsedV("object"),
					Properties: openapi.MParsedV(map[string]openapi.MaybeParsed[openapi.Schema]{
						"id": openapi.MParsedV(openapi.Schema{
							Type: openapi.MParsedV("integer"),
						}),
						"name": openapi.MParsedV(openapi.Schema{
							Type: openapi.MParsedV("string"),
						}),
					}),
				},
				{
					Type: openapi.MParsedV("object"),
					Properties: openapi.MParsedV(map[string]openapi.MaybeParsed[openapi.Schema]{
						"rank": openapi.MParsedV(openapi.Schema{
							Type: openapi.MParsedV("integer"),
						}),
					}),
				},
			}),
		}

		actual := schema.Combined(map[string]json.RawMessage{})

		expect.Equals(t, actual.Type.Or("-"), "object")
		expect.Equals(t, actual.Properties.Or(map[string]openapi.MaybeParsed[openapi.Schema]{}), map[string]openapi.MaybeParsed[openapi.Schema]{
			"id": openapi.MParsedV(openapi.Schema{
				Type: openapi.MParsedV("integer"),
			}),
			"name": openapi.MParsedV(openapi.Schema{
				Type: openapi.MParsedV("string"),
			}),
			"rank": openapi.MParsedV(openapi.Schema{
				Type: openapi.MParsedV("integer"),
			}),
		})
	})
}
