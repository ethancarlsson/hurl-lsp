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

type MaybeParsed[T any] struct {
	json.RawMessage

	v    T
	read bool
	err  error
}

func (j MaybeParsed[T]) MarshalJSON() ([]byte, error) {
	return json.Marshal(j.Or(*new(T)))
}

func (j MaybeParsed[T]) Read() (T, error) {
	if j.read {
		return j.v, j.err
	}

	var v T
	if err := json.Unmarshal(j.RawMessage, &v); err != nil {
		j.read = true
		j.err = err

		return v, err
	}

	j.set(v)

	return v, nil
}
func (j *MaybeParsed[T]) set(v T) {
	j.read = true
	j.v = v
}

func (j MaybeParsed[T]) OrEmpty() T {
	return j.Or(*new(T))
}

func (j MaybeParsed[T]) Or(or T) T {
	v, err := j.Read()
	if err != nil {
		return or
	}

	return v
}

func (j MaybeParsed[T]) OrLazy(or func() T) T {
	v, err := j.Read()
	if err != nil {
		return or()
	}

	return v
}

// MParsedV[alue] returns a MaybeParsed value where the value is alreay set; it will
// not need to be parsed.
func MParsedV[T any](v T) MaybeParsed[T] {
	return MaybeParsed[T]{
		v:    v,
		read: true,
	}
}

func MParsed[T any](j json.RawMessage) MaybeParsed[T] {
	return MaybeParsed[T]{RawMessage: j}
}

func MParsedS(s string) MaybeParsed[string] {
	return MaybeParsed[string]{RawMessage: json.RawMessage(`"` + s + `"`)}
}

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
	Summary     MaybeParsed[string] `json:"summary"`
	Description MaybeParsed[string] `json:"description"`
	Parameters  OpParams            `json:"parameters"`
	RequestBody RequestBody         `json:"requestBody"`
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
		m[op.Name] = fmt.Sprintf("%s: %s=%s", op.In, op.Name, op.Schema.Type.OrEmpty())
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

func (ops OpParams) Required(in string) []string {
	reqOps := make([]string, 0, len(ops))

	for _, op := range ops {
		if op.Required && op.In == in {
			reqOps = append(reqOps, op.Name)
		}
	}

	return reqOps
}

type OpParam struct {
	Name     string `json:"name"`
	In       string `json:"in"`
	Schema   Schema `json:"schema"`
	Required bool   `json:"required"`
}

type Schema struct {
	Type                 MaybeParsed[string]                         `json:"type"`
	Properties           MaybeParsed[map[string]MaybeParsed[Schema]] `json:"properties"`
	Required             MaybeParsed[[]string]                       `json:"required"`
	Ref                  MaybeParsed[string]                         `json:"$ref"`
	AdditionalProperties MaybeParsed[*Schema]                        `json:"additionalProperties"`
	Items                MaybeParsed[*Schema]                        `json:"items"`
	OneOf                MaybeParsed[[]Schema]                       `json:"oneOf"`
	AllOf                MaybeParsed[[]Schema]                       `json:"allOf"`
	AnyOf                MaybeParsed[[]Schema]                       `json:"anyOf"`
}

func (s Schema) SchemaType() string {
	var impliedType string
	if ps, err := s.Properties.Read(); len(ps) > 0 && err != nil {
		impliedType = "object"
	} else if ads, err := s.AdditionalProperties.Read(); ads != nil && err != nil {
		impliedType = "object"
	} else if items, err := s.Items.Read(); items != nil && err != nil {
		impliedType = "array"
	}

	t := s.Type.Or(impliedType)
	if t == "" {
		return impliedType
	}

	return t
}

func (s Schema) Combined(existingProps map[string]json.RawMessage) Schema {
	for _, schema := range s.AnyOf.OrEmpty() {
		if t := schema.Type.OrEmpty(); t != "" {
			s.Type.set(t)
		}

		if len(schema.Properties.OrEmpty()) > 0 {
			props := s.Properties.OrEmpty()
			maps.Copy(props, schema.Properties.OrEmpty())
			s.Properties.set(props)
		}
	}

	for _, schema := range s.AllOf.OrEmpty() {
		if t := schema.Type.OrEmpty(); t != "" {
			s.Type.set(t)
		}

		if len(schema.Properties.OrEmpty()) > 0 {
			props := s.Properties.OrEmpty()
			maps.Copy(props, schema.Properties.OrEmpty())
			s.Properties.set(props)
		}
	}

	if len(s.OneOf.OrEmpty()) == 0 {
		return s
	}

	bestMatchPropCount := 0
	matchingPropCounts := make([]int, len(s.OneOf.OrEmpty()))
	for i, schema := range s.OneOf.OrEmpty() {
		for name := range schema.Properties.OrEmpty() {
			if _, ok := existingProps[name]; ok {
				matchingPropCounts[i] += 1
			}
		}

		if matchingPropCounts[i] > bestMatchPropCount {
			bestMatchPropCount = matchingPropCounts[i]
		}
	}

	oneOfsToMerge := make([]Schema, 0, len(s.OneOf.OrEmpty()))
	for i, schema := range s.OneOf.OrEmpty() {
		if matchingPropCounts[i] == bestMatchPropCount {
			oneOfsToMerge = append(oneOfsToMerge, schema)
		}
	}

	for _, schema := range oneOfsToMerge {
		if t := schema.Type.OrEmpty(); t != "" {
			s.Type.set(t)
		}

		if len(schema.Properties.OrEmpty()) > 0 {
			props := s.Properties.OrEmpty()
			maps.Copy(props, schema.Properties.OrEmpty())
			s.Properties.set(props)
		}
	}

	return s
}

func (s Schema) ToString(indent int) string {
	res := s.Type.OrEmpty()
	const indentIncrement = 2

	if len(s.OneOf.OrEmpty()) > 0 {
		res += "oneOf"
		for _, schema := range s.OneOf.OrEmpty() {
			res += "\n" + strings.Repeat(" ", indent+indentIncrement) + "| " + schema.ToString(indent+indentIncrement)
		}

		return res
	}

	props := s.Combined(make(map[string]json.RawMessage)).Properties.OrEmpty()
	items := s.Items.OrEmpty()
	if s.Type.OrEmpty() == "array" && items != nil {
		res += "<" + items.Type.OrEmpty() + ">"
		props = items.Properties.OrEmpty()
	}

	additionalProps := s.AdditionalProperties.OrEmpty()
	if additionalProps != nil {
		res += "<additionalProperties>"
		props = additionalProps.Properties.OrEmpty()
	}

	for k, p := range props {
		res += "\n" + strings.Repeat(" ", indent+indentIncrement) + "- " + k + ": " + p.OrEmpty().ToString(indent+indentIncrement)
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
	return o.Detail.Summary.OrEmpty() == undocumentedOpSummary
}

func (o OAI) resolveSchemaRef(schema Schema) Schema {
	schema = o.resolveSchemaCombiners(schema)
	items := schema.Items.OrEmpty()
	if items != nil {
		schemaItems := o.resolveSchemaRef(*items)
		schema.Items.set(&schemaItems)
	}

	additionalProps := schema.AdditionalProperties.OrEmpty()

	if additionalProps != nil {
		schemaAdditionalProperties := o.resolveSchemaRef(*additionalProps)
		schema.AdditionalProperties.set(&schemaAdditionalProperties)
	}

	if schema.Ref.OrEmpty() == "" {
		return schema
	}

	splitRefPtr := strings.Split(schema.Ref.OrEmpty(), "/")

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

	if schema.Type.OrEmpty() == "" {
		schema.Type.set(refedSchema.Type.OrEmpty())
	}

	if schema.Properties.Or(nil) == nil {
		schema.Properties.set(map[string]MaybeParsed[Schema]{})
	}
	if refedSchema.Properties.Or(nil) == nil {
		refedSchema.Properties.set(map[string]MaybeParsed[Schema]{})
	}

	refedSchema = o.resolveSchemaCombiners(refedSchema)
	if len(refedSchema.AllOf.Or([]Schema{})) > 0 {
		schema.AllOf.set(refedSchema.AllOf.Or([]Schema{}))
	}
	if len(refedSchema.OneOf.Or([]Schema{})) > 0 {
		schema.OneOf.set(refedSchema.OneOf.Or([]Schema{}))
	}
	if len(refedSchema.AnyOf.Or([]Schema{})) > 0 {
		schema.AnyOf.set(refedSchema.AnyOf.Or([]Schema{}))
	}

	props := schema.Properties.Or(map[string]MaybeParsed[Schema]{})
	maps.Copy(props, refedSchema.Properties.Or(map[string]MaybeParsed[Schema]{}))
	schema.Required.set(refedSchema.Required.Or([]string{}))

	for k, prop := range props {
		props[k] = MParsedV(o.resolveSchemaRef(prop.Or(Schema{})))
	}

	schema.Properties.set(props)

	return schema
}

func (o OAI) resolveSchemaCombiners(schema Schema) Schema {
	anyOf := schema.AnyOf.Or([]Schema{})
	for i, s := range anyOf {
		anyOf[i] = o.resolveSchemaRef(s)
	}
	schema.AnyOf.set(anyOf)

	allOf := schema.AllOf.Or([]Schema{})
	for i, s := range allOf {
		allOf[i] = o.resolveSchemaRef(s)
	}
	schema.AllOf.set(allOf)

	oneOf := schema.OneOf.Or([]Schema{})
	for i, s := range oneOf {
		oneOf[i] = o.resolveSchemaRef(s)
	}
	schema.OneOf.set(oneOf)

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
				Summary:     MParsedS(undocumentedOpSummary),
				Description: MParsedS("Path not found in provided openapi spec"),
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
		op.Path = path
		op.Method = method
		op.Detail = OpDetail{
			Summary:     MParsedS(undocumentedOpSummary),
			Description: MParsedS("Documentation is malformed json/yaml"),
		}

		return op
	}

	opDetail, ok := pathContent[strings.ToLower(method)]

	if !ok {
		op.Detail = OpDetail{
			Summary: MParsedS(undocumentedOpSummary),
			Description: MParsedS(fmt.Sprintf(
				"%s: undocumented method %s. The following methods are documented for this path %s.",
				pathInSpec,
				strings.ToUpper(method),
				strings.ToUpper(strings.Join(mapKeys(pathContent), ",")),
			)),
		}

		return op
	}

	opDetail.RequestBody.Content.Json.Schema = o.resolveSchemaRef(opDetail.RequestBody.Content.Json.Schema)
	op.Detail = opDetail

	return op
}

// Accepts a path and returns the availableMethods for the path most closely
// matching the path provided. usedURI is also returned which represents the path
// provided with query params and location data removed as well as anything that
// prefixes assumedPath from the documentation (URL or versioning data for example,
// https://example.com, http://versioned/v1 would be removed for example).
func (o OAI) GetPathMethods(path string) []string {
	longestMatching := 0
	var rawPathContent json.RawMessage
	var regUsed *regexp.Regexp

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
			regUsed = reg
		}
	}

	if regUsed == nil {
		return []string{}
	}

	const httpMethodsCount = 9
	methods := make(map[string]struct{}, httpMethodsCount)

	if err := json.Unmarshal(rawPathContent, &methods); err != nil {
		return []string{}
	}

	res := make([]string, 0, len(methods))
	for method := range methods {
		res = append(res, method)
	}

	pathLoc := regUsed.FindIndex([]byte(path))
	if len(pathLoc) < 1 {
		return res
	}

	return res
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
			continue
		}

		pRes[p] = re
	}

	oai.pathRegexps = pRes

	return oai, nil
}
