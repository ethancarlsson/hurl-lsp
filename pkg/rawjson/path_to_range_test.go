package rawjson

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"pgregory.net/rapid"
)

type nestedStruct struct {
	ID    int               `json:"id"`
	Tags  []string          `json:"tags"`
	Attrs map[string]string `json:"attrs"`
}

type deepNested struct {
	Matrix   [][]int                    `json:"matrix"`
	Lookup   map[string][]*nestedStruct `json:"lookup"`
	Optional *nestedStruct              `json:"optional"`
}

type structOfEverything struct {
	// Basic maps
	IntMap  map[string]int                 `json:"int_map"`
	StrMap  map[string]string              `json:"str_map"`
	MapMap  map[string]map[int]int         `json:"map_map"`
	ArrMap  map[string][]map[string]string `json:"arr_map"`
	BoolMap map[string]bool                `json:"bool_map"`

	// Basic values
	Str   string  `json:"str"`
	Int   int     `json:"int"`
	Float float64 `json:"float"`
	Bool  bool    `json:"bool"`

	// Deep nesting
	Deep deepNested `json:"deep"`
}

func TestPathToRange(t *testing.T) {

	rapid.Check(t, func(t *rapid.T) {
		var (
			obj = rapid.Make[structOfEverything]().Draw(t, "obj")
		)

		var allPaths = [][]string{
			// Top-level
			{"int_map"},
			{"str_map"},
			{"map_map"},
			{"arr_map"},
			{"bool_map"},
			{"str"},
			{"int"},
			{"float"},
			{"bool"},
			{"int_slice"},
			{"str_slice"},
			{"fixed_array"},
			{"slice_of_maps"},
			{"map_of_slices"},
			{"str_ptr"},
			{"int_ptr"},
			{"struct_ptr"},
			{"nested"},
			{"nested_slice"},
			{"nested_map"},
			{"deep"},
			{"map_of_struct_maps"},
			{"slice_of_deep_maps"},

			// struct_ptr (nestedStruct)
			{"struct_ptr", "id"},
			{"struct_ptr", "tags"},
			{"struct_ptr", "attrs"},

			// nested (nestedStruct)
			{"nested", "id"},
			{"nested", "tags"},
			{"nested", "attrs"},

			// deep (direct fields)
			{"deep", "matrix"},
			{"deep", "lookup"},
			{"deep", "optional"},

			// deep.optional (nestedStruct)
			{"deep", "optional", "id"},
			{"deep", "optional", "tags"},
			{"deep", "optional", "attrs"},
		}

		for _, path := range allPaths {
			t.Log(path)
			objJSON, err := json.Marshal(obj)
			require.NoError(t, err)
			objJSONStr := string(objJSON)
			// t.Log("Provided JSON", objJSONStr)
			asLines := strings.Split(objJSONStr, "\n")

			rnge, err := pathToRange(asLines, path)
			assert.NoError(t, err)
			assert.LessOrEqual(t, (rnge.Start.Col+1)*(rnge.Start.Line+1), (rnge.End.Line+1)*(rnge.End.Col+1), "end %v comes before start %v", rnge.End, rnge.Start)

			objJSON, err = json.MarshalIndent(obj, "	", "	")
			require.NoError(t, err)
			objJSONStr = string(objJSON)
			// t.Log("Provided JSON", objJSONStr)
			rnge, err = pathToRange(asLines, path)
			assert.NoError(t, err)
			assert.LessOrEqual(t, (rnge.Start.Col+1)*(rnge.Start.Line+1), (rnge.End.Line+1)*(rnge.End.Col+1), "end %v comes before start %v", rnge.End, rnge.Start)
		}
	})
}
