package hurlfile_test

import (
	"testing"

	"github.com/ethancarlsson/hurl-lsp/expect"
	"github.com/ethancarlsson/hurl-lsp/hurlfile"
)

func TestParse(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		hf, err := hurlfile.Parse("file://../fixtures/test.hurl")

		expect.NoErr(t, err)
		expect.Equals(t, 1, len(hf.Entries))
		expect.Equals(t, "POST", hf.Entries[0].Request.Method.Name)
		expect.Equals(t, 0, hf.Entries[0].Request.Method.Range.StartCol)
		expect.Equals(t, 4, hf.Entries[0].Request.Method.Range.EndCol)
		expect.Equals(t, "{{url}}/people", hf.Entries[0].Request.Target)
		expect.Equals(t, 0, hf.Range.StartLine)
		expect.Equals(t, 8, hf.Range.EndLine)
	})

	t.Run("partial request", func(t *testing.T) {
		hf, err := hurlfile.Parse("file://../fixtures/test_partial_req.hurl")

		expect.NoErr(t, err)
		expect.Equals(t, 2, len(hf.Entries))

		expect.Equals(t, "PATCH", hf.Entries[0].Request.Method.Name)
		expect.Equals(t, "GET", hf.Entries[1].Request.Method.Name)
		expect.Equals(t, 3, hf.Entries[1].Request.Method.Range.StartCol)
		expect.Equals(t, 6, hf.Entries[1].Request.Method.Range.EndCol)

		expect.Equals(t, "", hf.Entries[0].Request.Target)
		expect.Equals(t, `{ 
	"hello": "test" 
}`,
			hf.Entries[0].Request.Body.Raw)

		expect.Equals(t, 0, hf.Range.StartLine)
		expect.Equals(t, 7, hf.Range.EndLine)

		expect.Equals(t, 1, hf.Entries[0].Request.Range.StartLine)
		expect.Equals(t, 5, hf.Entries[0].Request.Range.EndLine)

		expect.Equals(t, 3, hf.Entries[0].Request.Body.Range.StartLine)
		expect.Equals(t, 0, hf.Entries[0].Request.Body.Range.StartCol)
		expect.Equals(t, 5, hf.Entries[0].Request.Body.Range.EndLine)

		expect.Equals(t, 6, hf.Entries[1].Range.StartLine)
		expect.Equals(t, 6, hf.Entries[1].Range.EndLine)

	})

	t.Run("file doesn't exist", func(t *testing.T) {
		_, err := hurlfile.Parse("file:///doesntexist")

		expect.Err(t, err)
		expect.ErrContains(t, "couldn't open file", err)
	})
}
