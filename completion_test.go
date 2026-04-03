package main

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"pgregory.net/rapid"
)

func TestGetSegmentIdxFromPos(t *testing.T) {
	t.Run("properties", rapid.MakeCheck(func(t *rapid.T) {
		rePath := `^\/(\{\{[\w-]+\}\}/|[a-zA-Z0-9._~-]+/)+$`
		var (
			base     = rapid.StringMatching(`[^\/]*`).Draw(t, "base URL")
			uri      = rapid.StringMatching(rePath).Draw(t, "URI")
			URL      = base + uri
			posInURI = rapid.IntRange(0, len(URL)).Draw(t, "posInURI")
		)

		segmentIdx := getSegmentIdxFromPos(URL, posInURI)
		if posInURI < len(base)-1 {
			assert.Equal(t, 0, segmentIdx, "segmentIdx should be 0 when posInURI %d is less than base length %d", posInURI, len(base))
		}

		if posInURI > len(URL)-1 {
			assert.Equal(t, strings.Count(URL, "/"), segmentIdx)
		}

		assert.LessOrEqual(t, segmentIdx, strings.Count(URL, "/"))
	}))
}
