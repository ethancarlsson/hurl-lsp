package completions

import protocol "github.com/tliron/glsp/protocol_3_16"

func AddMethod(items []protocol.CompletionItem) []protocol.CompletionItem {
	kind := protocol.CompletionItemKindMethod
	for _, method := range []string{
		"GET", "POST", "PUT", "PATCH",
		"HEAD", "DELETE", "CONNECT", "TRACE",
		"PATCH"} {
		items = append(items, protocol.CompletionItem{
			Label:      method,
			Kind:       &kind,
			InsertText: &method,
		})

	}

	return items
}

func AddFilters(items []protocol.CompletionItem) []protocol.CompletionItem {
	kind := protocol.CompletionItemKindFunction
	filters := []struct {
		filter      string
		desctiption string
		detail      string
	}{
		{"base64Decode", "Decodes a Base64 encoded string into bytes.", "in: string, out: bytes"},
		{"base64Encode", "Encodes bytes into Base64 encoded string.", "in: bytes, out: string"},
		{"base64UrlSafeDecode", "Decodes a Base64 encoded string into bytes (using Base64 URL safe encoding).", "in: string, out: bytes"},
		{"base64UrlSafeEncode", "Encodes bytes into Base64 encoded string (using Base64 URL safe encoding).", "in: bytes, out: string"},
		{"count", "Counts the number of items in a collection.", "in: collection, out: number"},
		{"daysAfterNow", "Returns the number of days between now and a date in the future.", "in: date, out: number"},
		{"daysBeforeNow", "Returns the number of days between now and a date in the past.", "in: date, out: number"},
		{"decode", "Decodes bytes to string using encoding.", "in: bytes, out: string"},
		{"first", "Returns the first element from a collection.", "in: collection, out: any"},
		{"format", "Formats a date to a string given a specification format.", "in: date, out: string"},
		{"htmlEscape", "Converts the characters &, < and > to HTML-safe sequence.", "in: string, out: string"},
		{"htmlUnescape", "Converts all named and numeric character references (e.g. &gt;, &#62;, &#x3e;) to the corresponding Unicode characters.", "in: string, out: string"},
		{"jsonpath", "Evaluates a JSONPath expression.", "in: string, out: any"},
		{"last", "Returns the last element from a collection.", "in: collection, out: any"},
		{"location", "Returns the target location URL of a redirection.", "in: response, out: any"},
		{"nth", "Returns the element from a collection at a zero-based index, accepts negative indices for indexing from the end of the collection.", "in: collection, out: any"},
		{"regex", "Extracts regex capture group. Pattern must have at least one capture group.", "in: string, out: string"},
		{"replace", "Replaces all occurrences of old string with new string.", "in: string, out: string"},
		{"replaceRegex", "Replaces all occurrences of a pattern with new string.", "in: string, out: string"},
		{"split", "Splits to a list of strings around occurrences of the specified delimiter.", "in: string, out: string"},
		{"toDate", "Converts a string to a date given a specification format.", "in: string, out: date"},
		{"toFloat", "Converts value to float number.", "in: string|number, out: number"},
		{"toHex", "Converts bytes to hexadecimal string.", "in: bytes, out: string"},
		{"toInt", "Converts value to integer number.", "in: string|number, out: number"},
		{"toString", "Converts value to string.", "in: any, out: string"},
		{"urlDecode", "Replaces %xx escapes with their single-character equivalent.", "in: string, out: string"},
		{"urlEncode", "Percent-encodes all the characters which are not included in unreserved chars (see RFC3986) with the exception of forward slash (/).", "in: string, out: string"},
		{"urlQueryParam", "Returns the value of a query parameter in a URL.", "in: string, out: string"},
		{"xpath", "Evaluates a XPath expression.", "in: string, out: string"},
	}
	for _, filter := range filters {
		items = append(items, protocol.CompletionItem{
			Label:         filter.filter,
			Kind:          &kind,
			InsertText:    &filter.filter,
			Documentation: &filter.desctiption,
			Detail:        &filter.detail,
		})
	}

	return items
}

func AddRespSection(items []protocol.CompletionItem) []protocol.CompletionItem {
	kind := protocol.CompletionItemKindEnumMember

	for _, section := range []string{"[Captures", "[Asserts"} {
		items = append(items, protocol.CompletionItem{
			Label:      section,
			Kind:       &kind,
			InsertText: &section,
		})
	}

	return items
}

func AddVars(items []protocol.CompletionItem, vars []string) []protocol.CompletionItem {
	kind := protocol.CompletionItemKindVariable

	for _, v := range vars {
		withCurlys := "{{" + v + "}}"
		items = append(items, protocol.CompletionItem{
			Label:      v,
			Kind:       &kind,
			InsertText: &withCurlys,
		})
	}

	return items
}
