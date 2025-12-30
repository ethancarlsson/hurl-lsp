package completions

import (
	"fmt"
	"strings"

	"github.com/ethancarlsson/hurl-lsp/builtin"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

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
	for filter, desc := range builtin.Filters {
		items = append(items, protocol.CompletionItem{
			Label:         filter,
			Kind:          &kind,
			InsertText:    &filter,
			Documentation: &desc.Desctiption,
			Detail:        ptr(desc.Detail.String()),
		})
	}

	return items
}

type CompletionPath struct {
	path string
}

func AddPaths(items []protocol.CompletionItem, paths []string) []protocol.CompletionItem {
	kind := protocol.CompletionItemKindField

	for _, path := range paths {
		pathToInsert := strings.ReplaceAll(strings.ReplaceAll(path, "{", "{{"), "}", "}}")
		items = append(items, protocol.CompletionItem{
			Label:      path,
			Kind:       &kind,
			InsertText: &pathToInsert,
		})
	}

	return items
}

func AddRequestBodyProperty(items []protocol.CompletionItem, name, typ string) []protocol.CompletionItem {
	insertText := fmt.Sprintf(`"%s": `, name)
	switch typ {
	case "string":
		insertText = insertText + `"$0"`
	case "array":
		insertText = insertText + "[$0]"
	case "object":
		insertText = insertText + "{$0}"
	default:
	}
	kind := protocol.CompletionItemKindProperty
	format := protocol.InsertTextFormatSnippet
	items = append(items, protocol.CompletionItem{
		Label:            `"` + name + `"`,
		Detail:           &typ,
		Kind:             &kind,
		InsertTextFormat: &format,
		InsertText:       &insertText,
	})
	return items
}

func AddRespSection(items []protocol.CompletionItem) []protocol.CompletionItem {
	kind := protocol.CompletionItemKindEnumMember

	for _, section := range []string{"Captures", "Asserts"} {
		insertText := "[" + section + "]"
		items = append(items, protocol.CompletionItem{
			Label:      section,
			Kind:       &kind,
			InsertText: &insertText,
		})
	}

	return items
}

func AddRespVersion(items []protocol.CompletionItem) []protocol.CompletionItem {
	kind := protocol.CompletionItemKindEnumMember

	for _, version := range []string{"HTTP", "HTTP/1.0", "HTTP/1.1", "HTTP/2", "HTTP/3"} {
		insertText := version + " " // add space so that user can start input
		items = append(items, protocol.CompletionItem{
			Label:      version,
			Kind:       &kind,
			InsertText: &insertText,
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

func AddParams(items []protocol.CompletionItem, params map[string]string) []protocol.CompletionItem {
	kind := protocol.CompletionItemKindField

	for param, paramDocs := range params {
		paramText := param + "="
		items = append(items, protocol.CompletionItem{
			Label:         param,
			Kind:          &kind,
			InsertText:    &paramText,
			Documentation: paramDocs,
		})
	}

	return items
}

func ptr[T any](v T) *T {
	return &v
}
