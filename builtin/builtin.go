package builtin

import "fmt"

type Desc struct {
	Desctiption string
	Detail      InOut
}

type InOut struct {
	In  string
	Out string
}

func (i InOut) String() string {
	return fmt.Sprintf("in: %s, out %s", i.In, i.Out)
}

var Filters = map[string]Desc{
	"base64Decode":        {"Decodes a Base64 encoded string into bytes.", InOut{"string", "bytes"}},
	"base64Encode":        {"Encodes bytes into Base64 encoded string.", InOut{"bytes", "string"}},
	"base64UrlSafeDecode": {"Decodes a Base64 encoded string into bytes (using Base64 URL safe encoding).", InOut{"string", "bytes"}},
	"base64UrlSafeEncode": {"Encodes bytes into Base64 encoded string (using Base64 URL safe encoding).", InOut{"bytes", "string"}},
	"count":               {"Counts the number of items in a collection.", InOut{"collection", "number"}},
	"daysAfterNow":        {"Returns the number of days between now and a date in the future.", InOut{"date", "number"}},
	"daysBeforeNow":       {"Returns the number of days between now and a date in the past.", InOut{"date", "number"}},
	"decode":              {"Decodes bytes to string using encoding.", InOut{"bytes", "string"}},
	"first":               {"Returns the first element from a collection.", InOut{"collection", "any"}},
	"format":              {"Formats a date to a string given a specification format.", InOut{"date", "string"}},
	"htmlEscape":          {"Converts the characters &, < and > to HTML-safe sequence.", InOut{"string", "string"}},
	"htmlUnescape":        {"Converts all named and numeric character references (e.g. &gt;, &#62;, &#x3e;) to the corresponding Unicode characters.", InOut{"string", "string"}},
	"jsonpath":            {"Evaluates a JSONPath expression.", InOut{"string", "any"}},
	"last":                {"Returns the last element from a collection.", InOut{"collection", "any"}},
	"location":            {"Returns the target location URL of a redirection.", InOut{"response", "any"}},
	"nth":                 {"Returns the element from a collection at a zero-based index, accepts negative indices for indexing from the end of the collection.", InOut{"collection", "any"}},
	"regex":               {"Extracts regex capture group. Pattern must have at least one capture group.", InOut{"string", "string"}},
	"replace":             {"Replaces all occurrences of old string with new string.", InOut{"string", "string"}},
	"replaceRegex":        {"Replaces all occurrences of a pattern with new string.", InOut{"string", "string"}},
	"split":               {"Splits to a list of strings around occurrences of the specified delimiter.", InOut{"string", "string"}},
	"toDate":              {"Converts a string to a date given a specification format.", InOut{"string", "date"}},
	"toFloat":             {"Converts value to float number.", InOut{"string|number", "number"}},
	"toHex":               {"Converts bytes to hexadecimal string.", InOut{"bytes", "string"}},
	"toInt":               {"Converts value to integer number.", InOut{"string|number", "number"}},
	"toString":            {"Converts value to string.", InOut{"any", "string"}},
	"urlDecode":           {"Replaces %xx escapes with their single-character equivalent.", InOut{"string", "string"}},
	"urlEncode":           {"Percent-encodes all the characters which are not included in unreserved chars (see RFC3986) with the exception of forward slash (/).", InOut{"string", "string"}},
	"urlQueryParam":       {"Returns the value of a query parameter in a URL.", InOut{"string", "string"}},
	"xpath":               {"Evaluates a XPath expression.", InOut{"string", "string"}},
}
