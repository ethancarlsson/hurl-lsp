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
		{
			name: "with object in array",
			jsonLines: `{
	"name": "Fido",
	"tags": [
		{
			"id": 123,

		}

	]

}`,
			currLine:     5,
			currCol:      0,
			expectedPath: []string{"tags"},
		},
		{
			name: "with object in array",
			jsonLines: `{
	"name": "Fido",
	"tags": [
		{
			"id": 123,
			"inner": {

			}
		}

	]

}`,
			currLine:     6,
			currCol:      0,
			expectedPath: []string{"tags", "inner"},
		},
		{
			name: "under array",
			jsonLines: `{
	"name": "Fido",
	"tags": [
		{
			"id": 123,
			"inner": {

			}
		}

	]

}`,
			currLine:     11,
			currCol:      0,
			expectedPath: []string{},
		},
	}

	for _, tt := range tests {
		t.Run("test path to json object : "+tt.name, func(t *testing.T) {
			path := rawjson.PathToObj(strings.Split(tt.jsonLines, "\n"), tt.currLine, tt.currCol)

			actual := make([]string, 0, len(path))
			for _, p := range path {
				actual = append(actual, p.Name())
			}

			expect.Equals(t, tt.expectedPath, actual)
		})
	}
}

func TestIsOnLastProp(t *testing.T) {

	tests := []struct {
		name              string
		jsonLines         string
		currLine, currCol int
		expected          bool
	}{
		{
			name: "first position",
			jsonLines: `{

	"name": "Fido",
	"category": {

	},

}`,
			currLine: 1,
			currCol:  0,
			expected: false,
		},
		{
			name: "last position",
			jsonLines: `{

	"name": "Fido",
	"category": {

	},

}`,
			currLine: 6,
			currCol:  0,
			expected: true,
		},
		{
			name: "inner object position",
			jsonLines: `{

	"name": "Fido",
	"category": {

	},

}`,
			currLine: 4,
			currCol:  0,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expect.Equals(t, tt.expected, rawjson.IsOnLastProp(
				strings.Split(tt.jsonLines, "\n"),
				tt.currLine,
				tt.currCol))
		})
	}
}
