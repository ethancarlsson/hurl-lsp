package main

import (
	"encoding/json"
	"slices"
	"strings"

	"github.com/ethancarlsson/hurl-lsp/pkg/completions"
	"github.com/ethancarlsson/hurl-lsp/pkg/hurlfile"
	"github.com/ethancarlsson/hurl-lsp/pkg/openapi"
	"github.com/ethancarlsson/hurl-lsp/pkg/rawjson"
	"github.com/ethancarlsson/hurl-lsp/pkg/state"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func completion(context *glsp.Context, params *protocol.CompletionParams) (any, error) {
	items := make([]protocol.CompletionItem, 0)
	hf := state.Hf()

	if hf == nil {
		return items, nil
	}

	line := int(params.Position.Line)
	col := int(params.Position.Character) - 1 // zero base

	if hf.OnRespSectionName(line, col) {
		items = completions.AddRespSection(items)
	}

	if hf.OnEmptyResponse(line, col) {
		items = completions.AddRespVersion(items)
	}

	if caps := hf.Captures().Before(line); len(caps) > 0 {
		items = completions.AddVars(items, caps.Variables())
	}

	if hf.CanUseFilter(line, col) {
		items = completions.AddFilters(items)
	}

	items = addSpecAwareCompletions(hf, items, line, col)

	return items, nil
}

func addSpecAwareCompletions(hf *hurlfile.HurlFile, items []protocol.CompletionItem, line, col int) []protocol.CompletionItem {
	req := hf.GetReq(line, col)
	op := state.OAI().GetOp(req.Method.Name, strings.ReplaceAll(strings.ReplaceAll(req.Target.Target, "{{", "{"), "}}", "}"))

	items = addUriCompletions(hf, items, op, line, col)
	items = addBodyCompletions(hf, items, req, op, line, col)

	return items
}

func addBodyCompletions(hf *hurlfile.HurlFile, items []protocol.CompletionItem, req hurlfile.Request, op openapi.Op, line, col int) []protocol.CompletionItem {
	if !hf.OnRequestBody(line, col) {
		return items
	}

	currentSchema := op.Detail.RequestBody.Content.Json.Schema

	if len(currentSchema.Properties.OrEmpty()) == 0 {
		return items
	}

	currLine := line - req.Body.Range.StartLine
	if len(req.Body.Value)-1 < currLine {
		return items
	}

	var alreadyProvidedProps map[string]json.RawMessage
	bodyLines := slices.Clone(req.Body.Value)

	reqBody := rawjson.CleanTrailingCommas(strings.Join(bodyLines, "\n"))
	reqBody = rawjson.CleanVariables(reqBody)

	if err := json.Unmarshal([]byte(reqBody), &alreadyProvidedProps); err != nil {
		return items
	}

	schemaPath := rawjson.PathToObj(req.Body.Value, currLine, col)
	for _, path := range schemaPath {
		innerSchema, ok := currentSchema.Properties.OrEmpty()[path.Name()]
		currentSchemaAdditionalProps := currentSchema.AdditionalProperties.Or(nil)
		if !ok && currentSchemaAdditionalProps != nil {
			innerSchema = openapi.MParsedV(*currentSchemaAdditionalProps)
		} else if !ok {
			currentSchema = openapi.Schema{}
			break
		}

		var innerPropMap map[string]json.RawMessage
		if path.IsArr() {
			var innerPropMapList []map[string]json.RawMessage

			innerJSON, ok := alreadyProvidedProps[path.Name()]

			if ok {
				if err := json.Unmarshal(innerJSON, &innerPropMapList); err != nil {
					break
				}
			}

			if len(innerPropMapList) == 0 {
				break
			}

			innerPropMap = innerPropMapList[0]
		} else {
			innerJSON, ok := alreadyProvidedProps[path.Name()]

			if ok {
				if err := json.Unmarshal(innerJSON, &innerPropMap); err != nil {
					break
				}
			}
		}

		alreadyProvidedProps = innerPropMap
		if path.IsArr() {
			innerSchemaItems := innerSchema.OrEmpty().Items.OrEmpty()
			currentSchema = *innerSchemaItems
		} else {
			currentSchema = innerSchema.OrEmpty()
		}
	}

	isOnLastProp := rawjson.IsOnLastProp(bodyLines, currLine, col)

	for name, schema := range currentSchema.Combined(alreadyProvidedProps).Properties.OrEmpty() {
		if _, isAlreadyThere := alreadyProvidedProps[name]; isAlreadyThere {
			continue
		}

		if isOnLastProp {
			items = completions.AddRequestBodyProperty(items, name, schema.OrEmpty().Type.OrEmpty(), "")
		} else {
			items = completions.AddRequestBodyProperty(items, name, schema.OrEmpty().Type.OrEmpty(), ",")
		}
	}

	return items
}

func addUriCompletions(hf *hurlfile.HurlFile, items []protocol.CompletionItem, op openapi.Op, line, col int) []protocol.CompletionItem {
	if !hf.OnUri(line, col) {
		return items
	}

	if op.NotDocumented() {
		items = completions.AddPaths(items, state.OAI().PathList())

		// nothing else can be done with this if it's not documented
		return items
	} else {
		items = completions.AddPaths(items, state.OAI().ChildPaths(op))
	}

	items = completions.AddParams(items, op.Detail.Parameters.In("query").ToDocMap())

	return items
}
