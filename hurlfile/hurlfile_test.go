package hurlfile_test

import (
	"slices"
	"testing"

	"github.com/ethancarlsson/hurl-lsp/expect"
	"github.com/ethancarlsson/hurl-lsp/hurlfile"
)

func TestParse(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		lines, err := hurlfile.ParseLines("file://../fixtures/test.hurl")
		expect.NoErr(t, err)

		hf, err := hurlfile.Parse(lines)
		expect.NoErr(t, err)

		expect.Equals(t, 1, len(hf.Entries))
		expect.Equals(t, "POST", hf.Entries[0].Request.Method.Name)
		expect.Equals(t, 0, hf.Entries[0].Request.Method.Range.StartCol)
		expect.Equals(t, 4, hf.Entries[0].Request.Method.Range.EndCol)
		expect.Equals(t, "{{url}}/people", hf.Entries[0].Request.Target.Target)
		expect.Equals(t, 0, hf.Range.StartLine)
		expect.Equals(t, 12, hf.Range.EndLine)

		expect.Equals(t, 7, hf.Entries[0].Response.Range.StartLine)
		expect.Equals(t, 8, hf.Entries[0].Response.Sections[0].Range.StartLine)
		expect.Equals(t, 10, hf.Entries[0].Response.Sections[0].Range.EndLine)

		expect.Equals(t, false, hf.OnRespSectionName(7, 0))
		expect.Equals(t, true, hf.OnRespSectionName(8, 0))
		expect.Equals(t, false, hf.OnRespSectionName(9, 0))
		expect.Equals(t, 1, len(hf.Captures()))
		vars := hf.Captures()[0].Variables
		slices.Sort(vars) // order not guaranteed
		expect.Equals(t, []string{"id", "test"}, vars)

		expect.Equals(t, 10, hf.Captures()[0].UseAfter)

		// [Captures] // 8
		// id: jsonpath "$.id" // 9
		// test: "hello" // 10
		// [Asserts] // 11
		// count "$.list" == 2 // 12
		expect.Equals(t, false, hf.CanUseFilter(8, 0))
		expect.Equals(t, true, hf.CanUseFilter(9, 5))
		expect.Equals(t, false, hf.CanUseFilter(9, 2))
		expect.Equals(t, true, hf.CanUseFilter(9, 3))

		// Captures should be able to use filters after : but not before
		expect.Equals(t, false, hf.CanUseFilter(10, 0))
		expect.Equals(t, true, hf.CanUseFilter(10, 5))
		expect.Equals(t, false, hf.CanUseFilter(10, 4))

		// In the quoted area
		expect.Equals(t, false, hf.CanUseFilter(10, 7))
		expect.Equals(t, false, hf.CanUseFilter(9, 14))

		// out of the quoted area
		expect.Equals(t, true, hf.CanUseFilter(9, 12))

		// Shouldn't be available on name
		expect.Equals(t, false, hf.CanUseFilter(11, 7))
		// Should be available at any part of the string
		expect.Equals(t, true, hf.CanUseFilter(12, 0))
		expect.Equals(t, true, hf.CanUseFilter(12, 5))
	})

	t.Run("partial request", func(t *testing.T) {
		lines, err := hurlfile.ParseLines("file://../fixtures/test_partial_req.hurl")
		expect.NoErr(t, err)

		hf, err := hurlfile.Parse(lines)
		expect.NoErr(t, err)

		expect.Equals(t, 2, len(hf.Entries))

		expect.Equals(t, "PATCH", hf.Entries[0].Request.Method.Name)
		expect.Equals(t, "GET", hf.Entries[1].Request.Method.Name)
		expect.Equals(t, 3, hf.Entries[1].Request.Method.Range.StartCol)
		expect.Equals(t, 6, hf.Entries[1].Request.Method.Range.EndCol)

		expect.Equals(t, "", hf.Entries[0].Request.Target.Target)
		expect.Equals(
			t,
			[]string{"{", `	"hello": "test"`, "}"},
			hf.Entries[0].Request.Body.Value)

		expect.Equals(t, 0, hf.Range.StartLine)
		expect.Equals(t, 8, hf.Range.EndLine)

		expect.Equals(t, 1, hf.Entries[0].Request.Range.StartLine)
		expect.Equals(t, 5, hf.Entries[0].Request.Range.EndLine)

		expect.Equals(t, 3, hf.Entries[0].Request.Body.Range.StartLine)
		expect.Equals(t, 0, hf.Entries[0].Request.Body.Range.StartCol)
		expect.Equals(t, 5, hf.Entries[0].Request.Body.Range.EndLine)

		expect.Equals(t, 2, hf.Entries[0].Request.Headers.Range.StartLine)
		expect.Equals(t, 2, hf.Entries[0].Request.Headers.Range.EndLine)

		expect.Equals(t, 6, hf.Entries[1].Range.StartLine)
		expect.Equals(t, 8, hf.Entries[1].Range.EndLine)

		expect.Equals(t, 7, hf.Entries[1].Response.Range.StartLine)
		expect.Equals(t, 8, hf.Entries[1].Response.Range.EndLine)
		expect.Equals(t, 8, hf.Entries[1].Response.Sections[0].Range.StartLine)
		expect.Equals(t, 8, hf.Entries[1].Response.Sections[0].Range.EndLine)

		expect.Equals(t, 1, hf.Entries[1].Response.Sections[0].Name.Range.StartCol)
		expect.Equals(t, 3, hf.Entries[1].Response.Sections[0].Name.Range.EndCol)
		expect.Equals(t, 8, hf.Entries[1].Response.Sections[0].Name.Range.StartLine)
		expect.Equals(t, 8, hf.Entries[1].Response.Sections[0].Range.EndLine)
	})

	t.Run("file doesn't exist", func(t *testing.T) {
		_, err := hurlfile.ParseLines("file://../fixtures/notexists.hurl")
		expect.Err(t, err)
		expect.ErrContains(t, "couldn't open file", err)
	})
}
