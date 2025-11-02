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
		expect.Equals(t, len(hf.Entries), 1)
		expect.Equals(t, hf.Entries[0].Request.Method, "POST")
		expect.Equals(t, hf.Entries[0].Request.Target, "{{url}}/people")
	})

	t.Run("file doesn't exist", func(t *testing.T) {
		_, err := hurlfile.Parse("file:///doesntexist")

		expect.Err(t, err)
		expect.ErrContains(t, "couldn't open file", err)
	})
}
