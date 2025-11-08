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

func (hf HurlFile) OnRespSection(line, col int) bool {
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

			if col > s.Range.StartCol-1 && col < s.Range.EndCol+1 {
				return true
			}
		}
	}

	return false
}
