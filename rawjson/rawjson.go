// This package includes helpers for json strings and lines of JSON strings.
// It's useful for interpreting partially correct JSON or for translating
// document coordinates into coordinates in a JSON document (e.g. line,col to JSON path)
package rawjson

import (
	"slices"
	"strings"
	"unicode"
)

// Takes in lines of JSON and returns a slice of JSON representing the path
// through maps required to get to the line,col pos
func PathToObj(lines []string, line, col int) []string {
	path := make([]string, 0)

	expectedPathLen := 0

	for i := line; i < len(lines); i++ {
		for j := col; j < len(lines[i]); j++ {
			if lines[i][j] == '}' {
				expectedPathLen += 1
			} else if lines[i][j] == '{' {
				expectedPathLen -= 1
			}
		}

		// not on last line
		if i < len(lines)-1 {
			col = 0
		}
	}

	// used to keep track of how many of '}' character has been seen since the
	// last '{' character
	endBraceCount := 0
	for i := line; i >= 0; i-- {
		lookingForObjName := false
		objPathName := make([]byte, 0)

		for j := col; j > 0; j-- {
			currChar := lines[i][j]

			if currChar == '}' && !lookingForObjName {
				endBraceCount += 1
				continue
			}

			if currChar == '{' {
				lookingForObjName = endBraceCount == 0
				endBraceCount -= 1
				continue
			}

			if lookingForObjName && (unicode.IsSpace(rune(currChar)) || string(currChar) == ":" || string(currChar) == "\"") {
				continue
			}

			if lookingForObjName && (unicode.IsLetter(rune(currChar)) || unicode.IsNumber(rune(currChar))) {
				objPathName = append(objPathName, currChar)
				continue
			}

			lookingForObjName = false
		}

		if i > 0 {
			col = len(lines[i-1]) - 1
		}

		if len(objPathName) > 0 {
			slices.Reverse(objPathName)
			path = append(path, string(objPathName))
			endBraceCount = 0
		}
	}

	slices.Reverse(path)
	return path
}

// This goes through and removes any trailing commas in a JSON string. This allows
// the tool to work with partially broken JSON which is very common when a user might
// be adding a new property to the object or array.
func CleanTrailingCommas(jsonStr string) string {
	lookingForComma := false
	splitStr := strings.Split(jsonStr, "")
	for i := len(splitStr) - 1; i >= 0; i-- {
		if splitStr[i] == "}" || splitStr[i] == "]" {
			lookingForComma = true
			continue
		}

		if unicode.IsSpace(rune(jsonStr[i])) {
			continue
		}

		if lookingForComma == true && splitStr[i] == "," {
			splitStr[i] = ""
		}

		lookingForComma = false
	}

	return strings.Join(splitStr, "")
}
