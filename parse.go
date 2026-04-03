package main

import (
	"fmt"
	"os"

	"github.com/ethancarlsson/hurl-lsp/pkg/hurlfile"
	"github.com/ethancarlsson/hurl-lsp/pkg/openapi"
	"github.com/ethancarlsson/hurl-lsp/pkg/state"
	"github.com/tliron/commonlog"
)

func parseDocument(uri string) error {
	parsedLines, err := hurlfile.ParseLines(uri)
	if err != nil {
		return fmt.Errorf("Failed to parse the hurl file %w", err)
	}

	state.SetLines(parsedLines)
	state.SetHfFromLines(parsedLines)

	return nil
}

func parseOpenapi() {
	combinedOpenapiDef := openapi.New()
	for _, defPath := range conf.OpenapiDefPaths {
		fileContent, err := os.ReadFile(string(defPath))
		if err != nil {
			if m := commonlog.NewErrorMessage(0); m != nil {
				m.Set("_message", "Could not read openapi file").
					Set("err", err).Send()
			}

			continue
		}

		openAPI, err := openapi.Parse(defPath.Ft(), fileContent)
		if err != nil {
			if m := commonlog.NewErrorMessage(0); m != nil {
				m.Set("_message", "Could not parse openapi file").
					Set("err", err).Send()
			}

			continue
		}

		combinedOpenapiDef = combinedOpenapiDef.Merge(openAPI)
	}

	state.SetOAI(&combinedOpenapiDef)
}
