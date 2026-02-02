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

func TestPathToRange(t *testing.T) {
	tests := []struct {
		name      string
		jsonLines string
		path      []string
		expected  rawjson.Range
	}{
		{
			name: "empty path",
			jsonLines: `{
				"test": {
					"id": 123
				}
			}`,
			path: []string{},
			expected: rawjson.Range{
				Start: rawjson.Pos{0, 0},
				End:   rawjson.Pos{4, 3},
			},
		},
		{
			name: "first obj",
			jsonLines: `{
				"test": {
					"id": 123,
					"inner": { "test": 123 }
				}
			}`,
			path: []string{"test"},
			expected: rawjson.Range{
				Start: rawjson.Pos{1, 4},
				End:   rawjson.Pos{4, 4},
			},
		},
		{
			name: "inner obj",
			jsonLines: `{
				"test": {
					"id": 123,
					"inner": { "test": 123 }
				}
			}`,
			path: []string{"test", "inner"},
			expected: rawjson.Range{
				Start: rawjson.Pos{3, 5},
				End:   rawjson.Pos{3, 28},
			},
		},
		{
			name: "inner obj one line",
			jsonLines: `{
				"test": { "inner": { "test": 123 } }
			}`,
			path: []string{"test", "inner"},
			expected: rawjson.Range{
				Start: rawjson.Pos{1, 14},
				End:   rawjson.Pos{1, 37},
			},
		},
		{
			name: "same start same end line",
			jsonLines: `{
				"test": { "inner": { 
					"test": 123 } }
			}`,
			path: []string{"test", "inner", "inner2"},
			expected: rawjson.Range{
				Start: rawjson.Pos{1, 14},
				End:   rawjson.Pos{2, 17},
			},
		},
		{
			name: "depth 3",
			jsonLines: `{
				"test": {
					"id": 123,
					"inner": { 
						"inner2": {
							"test": 123
						}
					}
				}
			}`,
			path: []string{"test", "inner", "inner2"},
			expected: rawjson.Range{
				Start: rawjson.Pos{4, 6},
				End:   rawjson.Pos{6, 6},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expect.Equals(t, tt.expected, rawjson.PathToRange(
				strings.Split(tt.jsonLines, "\n"),
				tt.path,
			))
		})
	}
}
