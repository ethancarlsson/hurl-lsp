package hurlfile

func (hf HurlFile) OnMethod(line, col int) bool {
	// 3 is the length of the smallest method
	if line == 0 && col <= 3 {
		return true
	}

	for _, entry := range hf.Entries {
		reqRange := entry.Request.Range
		if reqRange.StartLine != line {
			continue
		}

		return true
	}

	return false
}

func (hf HurlFile) GetReq(line, col int) Request {
	for _, entry := range hf.Entries {
		if line >= entry.Range.StartLine && line < entry.Range.EndLine {
			return entry.Request
		}
	}

	return Request{}
}

func (hf HurlFile) OnUri(line, col int) bool {
	for _, entry := range hf.Entries {
		req := entry.Request
		if line != req.Range.StartLine {
			continue
		}

		if col > req.Method.Range.EndCol {
			return true
		}
	}

	return false
}

func (hf HurlFile) OnRespSectionName(line, col int) bool {
	for _, entry := range hf.Entries {
		if entry.Response == nil {
			continue
		}

		respRange := entry.Response.Range
		if line == respRange.StartLine+1 && col <= 1 {
			return true
		}

		for _, s := range entry.Response.Sections {
			if line != s.Range.StartLine {
				continue
			}

			if col > s.Name.Range.StartCol-1 && col < s.Name.Range.EndCol+1 {
				return true
			}
		}
	}

	return false
}

func (hf HurlFile) CanUseFilter(line, col int) bool {
	for _, entry := range hf.Entries {
		if entry.Response == nil {
			continue
		}

		if line <= entry.Response.Range.StartLine {
			continue
		}

		sections := entry.Response.Sections
		if len(sections) == 0 {
			continue
		}

		if line > sections[len(sections)-1].Range.EndLine {
			continue
		}

		for _, sec := range sections {
			if line > sec.Range.StartLine && line <= sec.Range.EndLine {
				return canUseFilterOnLine(sec, line, col)
			}
		}
	}

	return false
}

func canUseFilterOnLine(sec Section, line, col int) bool {
	if sec.Name.Value != Capture {
		return true
	}
	linesFromSecStart := line - (sec.Range.StartLine + 1) // +1 because we don't include the name
	if len(sec.RawLines) < linesFromSecStart {
		return false
	}

	rawLine := sec.RawLines[linesFromSecStart]

	k, _ := splitHeader(rawLine)

	if sec.Name.Value == Capture && col <= len(k) {
		return false
	}

	inQuote := false
	for i, char := range rawLine {
		if inQuote && col == i {
			return false
		}

		if char == '"' {
			inQuote = !inQuote
		}
	}

	return true
}

// Used to check if user is starting to write the response
func (hf *HurlFile) OnEmptyResponse(line, col int) bool {
	if len(hf.Entries) == 0 {
		return false
	}

	if line == hf.Range.EndLine-1 && hf.Entries[len(hf.Entries)-1].Response == nil {
		return true
	}

	for _, entry := range hf.Entries {
		if len(entry.Request.Body.Value) == 0 && line == entry.Request.Range.EndLine && entry.Response == nil {
			return true
		}

		bodyRange := entry.Request.Body.Range
		if line > bodyRange.EndLine &&
			entry.Response == nil &&
			line < entry.Range.EndLine {
			return true
		}
	}

	return false
}

func (hf *HurlFile) OnRequestBody(line, col int) bool {
	if len(hf.Entries) == 0 {
		return false
	}

	for _, entry := range hf.Entries {
		bodyRange := entry.Request.Body.Range
		if line >= bodyRange.StartLine && line <= bodyRange.EndLine {
			return true
		}
	}

	return false
}
