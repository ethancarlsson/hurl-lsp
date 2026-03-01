package signaturehelp

import (
	"regexp"

	"github.com/ethancarlsson/hurl-lsp/builtin"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

var reWhitespace = regexp.MustCompile("\\s")

type Lines []string

func (l Lines) SymbolAt(lineNum, col int) Signature {
	if lineNum > len(l) {
		return ""
	}

	line := reWhitespace.Split(l[lineNum], -1)
	currTotalLen := 0
	for _, sig := range line {
		posInSym := col - (currTotalLen - 1)
		if posInSym >= 0 && posInSym < len(sig) {
			return Signature(sig)
		}
		currTotalLen += len(sig) + 1 // +1 for whitespace
	}

	return ""
}

type Signature string

func (s Signature) String() string {
	return string(s)
}

func (s Signature) Description() builtin.Desc {
	if desc, ok := builtin.Filters[string(s)]; ok {
		return desc
	}

	return builtin.Desc{}
}

func ParamsFromMap(m map[string]string) []protocol.ParameterInformation {
	pmi := make([]protocol.ParameterInformation, 0, len(m))
	for label, doc := range m {
		if label == "" || doc == "" {
			continue
		}

		pmi = append(pmi, protocol.ParameterInformation{
			Label:         label,
			Documentation: doc,
		})
	}

	return pmi
}
