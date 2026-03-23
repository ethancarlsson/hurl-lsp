package rawjson_test

import (
	"encoding/json"
	"testing"

	"github.com/ethancarlsson/hurl-lsp/pkg/rawjson"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"pgregory.net/rapid"
)

func TestCleanVariables(t *testing.T) {
	t.Run("with a normal string", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			in := rapid.String().Draw(t, "normal string")

			actual := rawjson.CleanVariables(in)
			assert.Len(t, actual, len(in))
			assert.NotRegexp(t, rawjson.ReHurlVariable, actual)
		})
	})

	t.Run("with a variable string", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			in := rapid.StringMatching("{{\\S+}}").Draw(t, "variable string")

			actual := rawjson.CleanVariables(in)

			assert.NotEqual(t, in, actual)
			assert.Len(t, actual, len(in))
			assert.NotRegexp(t, rawjson.ReHurlVariable, actual)
		})
	})

	t.Run("with a variable string inside a JSON object", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			mapIn := rapid.Map(rapid.Make[map[string]string](), func(m map[string]string) map[string]string {
				for k := range m {
					if len(k)%2 == 0 {
						m[k] = rapid.String().Draw(t, "regular string")
					} else {
						m[k] = rapid.StringMatching("{{\\S+}}").Draw(t, "variable string")
					}
				}

				return m
			})

			in, err := json.Marshal(mapIn)
			require.NoError(t, err)

			actual := rawjson.CleanVariables(string(in))

			assert.True(t, json.Valid([]byte(actual)))
			assert.NotEqual(t, in, actual)
			assert.Len(t, actual, len(in))
			assert.NotRegexp(t, rawjson.ReHurlVariable, actual)
		})
	})

	t.Run("with variable that would break JSON does not break JSON", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			in := `{
				"test": ` + rapid.StringMatching("{{\\S+}}").Draw(t, "variable string") + `
			}`

			actual := rawjson.CleanVariables(string(in))

			assert.True(t, json.Valid([]byte(actual)), "failed asserting is valid JSON: %s", actual)
			assert.NotEqual(t, in, actual)
			assert.Len(t, actual, len(in))
			assert.NotRegexp(t, rawjson.ReHurlVariable, actual)
		})
	})

	t.Run("with variable inside string does not break JSON", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			in := `{
				"test": "` + rapid.StringMatching("{{\\S+}}").Draw(t, "variable string") + `"
			}`

			actual := rawjson.CleanVariables(string(in))

			assert.True(t, json.Valid([]byte(actual)), "failed asserting is valid JSON: %s", actual)
			assert.NotEqual(t, in, actual)
			assert.Len(t, actual, len(in))
			assert.NotRegexp(t, rawjson.ReHurlVariable, actual)
		})
	})
}
