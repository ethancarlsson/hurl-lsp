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
