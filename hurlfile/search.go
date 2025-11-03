package hurlfile

func (hf HurlFile) IsOnMethod(line, col int) bool {
	if line == 0 && col == 1 {
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
