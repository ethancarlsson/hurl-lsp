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

func AddRespSection(items []protocol.CompletionItem) []protocol.CompletionItem {
	kind := protocol.CompletionItemKindEnumMember

	for _, section := range []string{"[Captures]", "[Asserts]"} {
		items = append(items, protocol.CompletionItem{
			Label:      section,
			Kind:       &kind,
			InsertText: &section,
		})
	}

	return items
}
