// This package includes helpers for json strings and lines of JSON strings.
// It's useful for interpreting partially correct JSON or for translating
// document coordinates into coordinates in a JSON document (e.g. line,col to JSON path)
package rawjson

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
	"unicode"
)

type JPath struct {
	string
}

func (j JPath) IsArr() bool {
	return strings.HasSuffix(j.string, `""`)
}

func (j JPath) Name() string {
	if j.IsArr() {
		return j.string[:len(j.string)-2]
	}

	return j.string
}

// Takes in lines of JSON and returns a slice of JSON representing the path
// through maps required to get to the line,col pos
func PathToObj(lines []string, line, col int) []JPath {
	path := make([]JPath, 0)

	// ignore col for now
	col = 0

	// used to keep track of how many of '}' character has been seen since the
	// last '{' character
	endBraceCount := 0
	endBracketCount := 0

	for i := line; i >= 0; i-- {
		lookingForObjName := false
		isObjInArr := false
		objPathName := make([]byte, 0)

		for j := col; j > 0; j-- {
			currChar := lines[i][j]

			// only add 1 if not looking for the name. If we're looking
			// for the name of the property we might run into a case
			// like "propnamewith{curlybraces}"
			if currChar == '}' && !lookingForObjName {
				endBraceCount += 1
				continue
			}

			if currChar == '{' {
				lookingForObjName = endBraceCount == 0
				endBraceCount -= 1
				continue
			}

			if currChar == ']' {
				endBracketCount -= 1
				continue
			}

			if currChar == '[' {
				isObjInArr = endBracketCount == 0
				lookingForObjName = endBracketCount == 0
				endBracketCount += 1
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
			if isObjInArr {
				objPathName = append(objPathName, '"', '"')
			}
			path = append(path, JPath{string(objPathName)})
			endBraceCount = 0
		}
	}

	slices.Reverse(path)
	return path
}

type Range struct {
	Start Pos
	End   Pos
}

type Pos struct {
	Line int
	Col  int
}

func PathToRange(lines []string, path []string) Range {
	lenLines := len(lines)
	lastCol := 0
	isInArr := false

	if lenLines > 0 {
		lastCol = len(lines[lenLines-1]) - 1
		curly := strings.Index(lines[0], "{")
		square := strings.Index(lines[0], "[")
		if square != -1 && (curly > square || curly == -1) {
			isInArr = true
		}
	}

	if len(path) == 0 {
		return Range{
			Start: Pos{
				Line: 0,
				Col:  0,
			},
			End: Pos{
				Line: lenLines - 1,
				Col:  lastCol,
			},
		}
	}

	nextPropertyName := path[0]

	// Wrapped in function to take advantage of early return
	getPathStart := func() Pos {
		pathStart := Pos{}

		for i, line := range lines {
			pPointer := 0
			for j, char := range line {
				if pPointer >= len(nextPropertyName) {
					pathStart.Line = i
					pathStart.Col = j - len(nextPropertyName) - 1
					return pathStart
				}

				if rune(nextPropertyName[pPointer]) == char {
					pPointer += 1
					continue
				}

				if rune(nextPropertyName[pPointer]) != char {
					pPointer = 0
					continue
				}
			}
		}

		return pathStart
	}

	getArrIndexPathStart := func() Pos {
		pathStart := Pos{}
		currentIndex := 0
		onNextChar := false
		startLooking := false

		for i, line := range lines {
			for j, char := range line {
				if char == '[' {
					startLooking = true
				}

				if !startLooking {
					continue
				}

				if onNextChar && !unicode.IsSpace(char) {
					pathStart.Line = i
					pathStart.Col = j
					return pathStart
				}

				if char == '[' && nextPropertyName == "0" {
					onNextChar = true
					continue
				}

				if char == ',' {
					currentIndex += 1
				}

				if fmt.Sprintf("%d", currentIndex) == nextPropertyName {
					onNextChar = true
					continue
				}
			}
		}

		return pathStart
	}

	var pathStart Pos
	if isInArr {
		pathStart = getArrIndexPathStart()
	} else {
		pathStart = getPathStart()
	}

	pathEnd := func() Pos {
		pathEnd := Pos{}
		started := false
		openCount := 0
		entityOpener := '{'
		entityCloser := '}'

		for i, line := range lines {
			if i < pathStart.Line {
				continue
			}

			for j, char := range line {
				if !started && j < pathStart.Col {
					continue
				}

				// If we haven't found the first '{' and the char
				// is a '[' we know it's an array and we can treat.
				// the whole thing as an array.
				if openCount == 0 && char == '[' {
					entityOpener = '['
					entityCloser = ']'
				}

				started = true

				if char == entityOpener {
					openCount += 1
				}

				if char == entityCloser {
					openCount -= 1
				}

				if char == entityCloser && openCount == 0 {
					pathEnd.Line = i
					pathEnd.Col = j
					return pathEnd
				}
			}
		}

		return pathEnd
	}()

	if len(path) == 1 {
		return Range{pathStart, pathEnd}
	}

	lines[pathEnd.Line] = lines[pathEnd.Line][:pathEnd.Col+1]
	lines[pathStart.Line] = lines[pathStart.Line][pathStart.Col:]
	lines = lines[pathStart.Line : pathEnd.Line+1]

	innerPathRange := PathToRange(lines, path[1:])
	startCol := innerPathRange.Start.Col
	endCol := innerPathRange.End.Col

	// If it starts at 0 that means that the innerPath is on the same
	// line as the outerPath. This means that the start column needs to be
	// considered as part of the inner position.
	if innerPathRange.Start.Line == 0 {
		startCol = startCol + pathStart.Col
	}

	if innerPathRange.End.Line == 0 {
		endCol = endCol + pathStart.Col
	}

	return Range{
		Start: Pos{
			Line: pathStart.Line + innerPathRange.Start.Line,
			Col:  startCol,
		},
		End: Pos{
			Line: pathStart.Line + innerPathRange.End.Line,
			Col:  endCol,
		},
	}
}

// Accepts lines, line, and col and returns true if adding a property at that
// position will make it the last property in the JSON object.
// It returns true if there are no characters between this position and the first
// '}' character other than whitespace.
func IsOnLastProp(lines []string, line, col int) bool {
	for i := line; i < len(lines); i++ {
		for j := col; j < len(lines[i]); j++ {
			if unicode.IsSpace(rune(lines[i][j])) {
				continue
			}

			return lines[i][j] == '}'
		}

		// move col to the front for the next line
		col = 0
	}

	return false
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

var reHurlVariable = regexp.MustCompile("{{\\S+}}")

// Takes in a string and replaces all hurl variables with "{}" this makes it possible
// to parse the string as JSON. "{}" is chosen because it will be valid JSON even
// in scenarios where the variable is wrapped in a string, e.g. both of these will
// be replaced in a way that can be parsed as json:
// - "id": {{id}}	-> "id": {}
// - "name": "{{name}}"	-> "name": "{}"
func CleanVariables(str string) string {
	return string(reHurlVariable.ReplaceAll([]byte(str), []byte("{}")))
}
