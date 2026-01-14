package diagnose

import (
	"fmt"
	"regexp"
	"slices"

	"github.com/agnivade/levenshtein"
)

var (
	paramRe = regexp.MustCompile(`\{\.*\}`)
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

func Path(path string, availablePaths []string) Diagnosis {
	path = string(paramRe.ReplaceAll([]byte(path), []byte{}))

	shortestDist := 99999999
	closest := ""

	for _, availableP := range availablePaths {
		availableP = string(paramRe.ReplaceAll([]byte(availableP), []byte{}))
		if computedDist := levenshtein.ComputeDistance(availableP, path); computedDist < shortestDist {
			shortestDist = computedDist
			closest = availableP
		}
	}

	diagnosis := Diagnosis{
		Message:    "Undocumented uri." + fmt.Sprintf(" Did you mean %s?", closest),
		HasProblem: true,
	}

	return diagnosis
}
