package openapi

import (
	"encoding/json"
	"fmt"
	"maps"
	"regexp"
	"slices"
	"strings"

	"github.com/goccy/go-yaml"
)

var (
	paramRe = regexp.MustCompile(`\{\w*\}`)
)

type OAI struct {
	Paths       map[string]json.RawMessage `json:"paths"`
	Components  Components                 `json:"components"`
	pathRegexps map[string]*regexp.Regexp
}

type Components struct {
	Schemas map[string]Schema `json:"schemas"`
}

func New() OAI {
	return OAI{
		Paths: make(map[string]json.RawMessage),
		Components: Components{
			Schemas: make(map[string]Schema),
		},
		pathRegexps: make(map[string]*regexp.Regexp),
	}
}

func (o OAI) Merge(other OAI) OAI {
	maps.Copy(o.Paths, other.Paths)
	maps.Copy(o.pathRegexps, other.pathRegexps)
	maps.Copy(o.Components.Schemas, other.Components.Schemas)

	return o
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

func (o OAI) ChildPaths(op Op) []string {
	if len(o.Paths) == 0 {
		return []string{}
	}

	paths := make([]string, 0, len(o.Paths)-1)

	for p := range o.Paths {
		child := strings.TrimPrefix(p, op.Path)
		if child != "" && child != p {
			paths = append(paths, child)
		}
	}

	return paths
}

type OpDetail struct {
	Summary     string      `json:"summary"`
	Description string      `json:"description"`
	Parameters  OpParams    `json:"parameters"`
	RequestBody RequestBody `json:"requestBody"`
}

type RequestBody struct {
	Description string             `json:"description"`
	Content     RequestBodyContent `json:"content"`
}

type RequestBodyContent struct {
	Json BodySchema `json:"application/json"`
}

type BodySchema struct {
	Schema Schema `json:"schema"`
}

type OpParams []OpParam

func (ops OpParams) ToDocMap() map[string]string {
	m := make(map[string]string, len(ops))
	for _, op := range ops {
		m[op.Name] = fmt.Sprintf("%s: %s=%s", op.In, op.Name, op.Schema.Type)
	}

	return m
}

func (ops OpParams) In(in string) OpParams {
	params := make(OpParams, 0, len(ops))
	for _, op := range ops {
		if op.In == in {
			params = append(params, op)
		}
	}

	return params
}

type OpParam struct {
	Name   string `json:"name"`
	In     string `json:"in"`
	Schema Schema `json:"schema"`
}

type Schema struct {
	Type       string            `json:"type"`
	Properties map[string]Schema `json:"properties"`
	Ref        string            `json:"$ref"`
	Items      *Schema           `json:"items"`
}

func (s Schema) ToString(indent int) string {
	res := strings.Repeat(" ", indent) + s.Type
	for k, p := range s.Properties {
		res += "\n" + strings.Repeat(" ", indent+2) + "- " + k + ": " + p.ToString(indent+2)
	}

	return res
}

type Op struct {
	Method string
	Path   string
	Detail OpDetail
}

const undocumentedOpSummary = "Operation not documented"

func (o Op) NotDocumented() bool {
	return o.Detail.Summary == undocumentedOpSummary
}

func (o OAI) resolveSchemaRef(schema Schema) Schema {
	if schema.Items != nil {
		schemaItems := o.resolveSchemaRef(*schema.Items)
		schema.Items = &schemaItems
	}

	if schema.Ref == "" {
		return schema
	}

	splitRefPtr := strings.Split(schema.Ref, "/")

	schemasI := slices.Index(splitRefPtr, "schemas")
	if schemasI == -1 {
		return schema
	}

	// Don't try and reach cases where the ref looks like "#/components/schemas/"
	// i.e. with no actual schema
	if len(splitRefPtr) < schemasI {
		return schema
	}

	schemaKey := splitRefPtr[schemasI+1]
	refedSchema, ok := o.Components.Schemas[schemaKey]
	if !ok {
		return schema
	}

	if schema.Type == "" {
		schema.Type = refedSchema.Type
	}

	if schema.Properties == nil {
		schema.Properties = make(map[string]Schema)
	}
	if refedSchema.Properties == nil {
		refedSchema.Properties = make(map[string]Schema)
	}

	maps.Copy(schema.Properties, refedSchema.Properties)

	for k, prop := range schema.Properties {
		schema.Properties[k] = o.resolveSchemaRef(prop)
	}

	return schema
}

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

	const httpMethodsCount = 9
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

	opDetail.RequestBody.Content.Json.Schema = o.resolveSchemaRef(opDetail.RequestBody.Content.Json.Schema)
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
