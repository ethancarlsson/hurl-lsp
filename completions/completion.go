package completions

import protocol "github.com/tliron/glsp/protocol_3_16"

func AddMethodCompletions(items []protocol.CompletionItem) []protocol.CompletionItem {
	methodKind := protocol.CompletionItemKindMethod
	for _, method := range []string{
		"GET", "POST", "PUT", "PATCH",
		"HEAD", "DELETE", "CONNECT", "TRACE",
		"PATCH"} {
		items = append(items, protocol.CompletionItem{
			Label:      method,
			Kind:       &methodKind,
			InsertText: &method,
		})

	}

	return items
}
