package rawjson_test

import (
	"strings"
	"testing"

	"github.com/ethancarlsson/hurl-lsp/expect"
	"github.com/ethancarlsson/hurl-lsp/rawjson"
)

func TestPathToObj(t *testing.T) {
	tests := []struct {
		name              string
		jsonLines         string
		currLine, currCol int
		expectedPath      []string
	}{
		{
			name: "highest level",
			jsonLines: `{
	"name": "Fido",
	"category": {

	},

}`,
			currLine:     0,
			currCol:      0,
			expectedPath: []string{},
		},
		{
			name: "nested",
			jsonLines: `{
	"name": "Fido",
	"category": {

	},

}`,
			currLine:     3,
			currCol:      0,
			expectedPath: []string{"category"},
		},
		{
			name: "deeply nested",
			jsonLines: `{
	"name": "Fido",
	"category": {
		"inner1": {
			"inner2": {
				"inner3": {

				}
			}
		}
	}
}`,
			currLine:     6,
			currCol:      0,
			expectedPath: []string{"category", "inner1", "inner2", "inner3"},
		},
		{
			name: "partial nesting",
			jsonLines: `{
	"name": "Fido",
	"category": {
		"inner1": {

			"inner2": {
				"inner3": {

				}
			}
		}
	}
}`,
			currLine:     4,
			currCol:      0,
			expectedPath: []string{"category", "inner1"},
		},
		{
			name: "highest level at the bottom",
			jsonLines: `{
	"name": "Fido",
	"category": {

	},

}`,
			currLine:     5,
			currCol:      0,
			expectedPath: []string{},
		},
		{
			name: "with sibling objects",
			jsonLines: `{
	"name": "Fido",
	"1sib": {},
	"2sib": {},
	"3sib": {},
	"category": {

	},

}`,
			currLine:     6,
			currCol:      0,
			expectedPath: []string{"category"},
		},
	}

	for _, tt := range tests {
		t.Run("test path to json object : "+tt.name, func(t *testing.T) {
			actual := rawjson.PathToObj(strings.Split(tt.jsonLines, "\n"), tt.currLine, tt.currCol)

			expect.Equals(t, tt.expectedPath, actual)
		})
	}
}
