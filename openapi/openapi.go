package openapi

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/goccy/go-yaml"
)

var (
	paramRe = regexp.MustCompile(`\{\w*\}`)
)

type OAI struct {
	Paths       map[string]json.RawMessage `json:"paths"`
	pathRegexps map[string]*regexp.Regexp
}

func (o OAI) PathList() []string {
	if len(o.Paths) == 0 {
		return []string{}
	}

	paths := make([]string, 0, len(o.Paths))

	for p := range o.Paths {
		paths = append(paths, p)

	}

	return paths
}

type OpDetail struct {
	Summary     string `json:"summary"`
	Description string `json:"description"`
}

type Op struct {
	Method string
	Path   string
	Detail OpDetail
}

const undocumentedOpSummary = "Operation not documented"

func (o OAI) GetOp(method, path string) Op {
	// We look for the longest possible match to get the most specific match
	// So if there is /pets and /pets/{id}, /pets/1 will match both but we would
	// want the /pets/{id} path so that's the one we get the documentation of.
	longestMatching := 0
	pathInSpec := ""
	var rawPathContent json.RawMessage
	for pInSpec, content := range o.Paths {
		if len(pInSpec) < longestMatching {
			continue
		}

		reg, ok := o.pathRegexps[pInSpec]
		if !ok || reg == nil {
			continue
		}

		match := reg.MatchString(path)
		// println("reg", reg.String(), "path", path, "match", match)
		if match {
			rawPathContent = content
			longestMatching = len(pInSpec)
			pathInSpec = pInSpec
		}
	}

	if longestMatching == 0 {
		return Op{
			Path:   path,
			Method: method,
			Detail: OpDetail{
				Summary:     undocumentedOpSummary,
				Description: "Path not found in provided openapi spec",
			},
		}
	}

	op := Op{
		Path:   pathInSpec,
		Method: method,
	}

	const httpMethodsCount = 6
	pathContent := make(map[string]OpDetail, httpMethodsCount)

	if err := json.Unmarshal(rawPathContent, &pathContent); err != nil {
		op.Detail = OpDetail{
			Summary:     undocumentedOpSummary,
			Description: "Documentation of is malformed json/yaml",
		}

		return op
	}

	opDetail, ok := pathContent[strings.ToLower(method)]

	if !ok {
		op.Detail = OpDetail{
			Summary: undocumentedOpSummary,
			Description: fmt.Sprintf(
				"%s: undocumented method %s. The following methods are documented for this path %s.",
				pathInSpec,
				strings.ToUpper(method),
				strings.ToUpper(strings.Join(mapKeys(pathContent), ",")),
			),
		}

		return op
	}

	op.Detail = opDetail
	return op
}

func mapKeys[T any](m map[string]T) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}

	return keys
}

func Parse(ftype string, fContents []byte) (OAI, error) {
	fail := func(err error) error {
		return fmt.Errorf("could not parse file %w", err)
	}
	oai := OAI{}
	if ftype != "json" {
		// we transform it into JSON so that we can use json.RawMessage
		// for both file types and then only unmarshal what we need to.
		// If we delay parsing of individual paths we can allow the system
		// to work with only partially valid api schemas
		jsonContent, err := yaml.YAMLToJSON(fContents)
		if err != nil {
			return oai, fail(err)
		}
		fContents = jsonContent
	}

	if err := json.Unmarshal(fContents, &oai); err != nil {
		return oai, fail(err)
	}

	pRes := make(map[string]*regexp.Regexp, len(oai.Paths)*2)
	for p := range oai.Paths {
		re, err := regexp.Compile(string(paramRe.ReplaceAll([]byte(p), []byte(`[a-zA-Z0-9_{}]+`))))
		if err != nil {
			println("err", fail(err).Error())
			continue
		}

		pRes[p] = re
	}

	oai.pathRegexps = pRes

	return oai, nil
}
