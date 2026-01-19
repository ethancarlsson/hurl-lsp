package diagnose

import (
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/agnivade/levenshtein"
)

var (
	variableRe = regexp.MustCompile(`\{.*\}`)
)

type Diagnosis struct {
	HasProblem bool
	Message    string
}

var validMethods = []string{"GET", "HEAD", "POST", "PUT",
	"DELETE", "CONNECT", "OPTIONS", "TRACE", "PATCH"}

func HTTPMethod(method string) Diagnosis {
	if slices.Contains(validMethods, method) {
		return Diagnosis{
			HasProblem: false,
		}
	}

	closest, shortestDist := closestMethodName(method)
	diagnosis := Diagnosis{
		Message:    "Undefined HTTP method used.",
		HasProblem: true,
	}

	if shortestDist <= 3 {
		diagnosis.Message += fmt.Sprintf(" Did you mean %s?", closest)
	}

	return diagnosis
}

func closestMethodName(method string) (closest string, shortestDist int) {
	shortestDist = 99999999
	closest = ""

	for _, validM := range validMethods {
		if computedDist := levenshtein.ComputeDistance(validM, method); computedDist < shortestDist {
			shortestDist = computedDist
			closest = validM
		}
	}

	return closest, shortestDist
}

func QueryParams(path string, requiredParams []string) Diagnosis {
	split := strings.Split(path, "?")
	if len(split) < 2 {
		return Diagnosis{
			HasProblem: len(requiredParams) > 0,
			Message:    fmt.Sprintf("Missing required params: %v", requiredParams),
		}
	}

	params := strings.Split(split[1], "&")
	providedParams := make([]string, 0, len(params))
	for _, param := range params {
		kv := strings.Split(param, "=")
		providedParams = append(providedParams, kv[0])
	}

	missingParams := make([]string, 0, len(requiredParams))

	for _, requiredParam := range requiredParams {
		if !slices.Contains(providedParams, requiredParam) {
			missingParams = append(missingParams, requiredParam)
		}
	}

	if len(missingParams) == 0 {
		return Diagnosis{
			HasProblem: false,
		}
	}

	return Diagnosis{
		HasProblem: true,
		Message:    fmt.Sprintf("Missing required params: %v", missingParams),
	}
}
