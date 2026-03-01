package hurlfile

const Capture = "Captures"

type CaptureVars struct {
	UseAfter  int
	Variables []string
}

type Captures []CaptureVars

func (c Captures) Before(line int) Captures {
	newCaps := make(Captures, 0, len(c))
	for _, capture := range c {
		if capture.UseAfter < line {
			newCaps = append(newCaps, capture)
		}
	}

	return newCaps
}

func (c Captures) Variables() []string {
	vars := make([]string, 0)

	for _, capture := range c {
		vars = append(vars, capture.Variables...)
	}

	return vars
}

func (hf *HurlFile) Captures() Captures {
	caps := make([]CaptureVars, 0, len(hf.Entries))
	for _, entry := range hf.Entries {
		if entry.Response == nil {
			continue
		}

		for _, section := range entry.Response.Sections {
			if section.Name.Value != Capture {
				continue
			}

			vars := make([]string, 0, len(section.KeyValues))
			for k := range section.KeyValues {
				vars = append(vars, k)
			}

			caps = append(caps, CaptureVars{
				UseAfter:  section.Range.EndLine,
				Variables: vars,
			})
		}
	}

	return caps
}
