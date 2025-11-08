package hurlfile

import (
	"fmt"
	"strconv"
	"strings"
)

type Response struct {
	Version  string
	Status   int
	Headers  map[string]string
	Sections []Section
	Body     string
	Range    SourceRange
}

// parseResponse expects current line is response line (HTTP/.. status)
func (p *Parser) parseResponse() (*Response, error) {
	line := strings.TrimSpace(p.next())
	parts := strings.Fields(line)
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid response status line: %q", line)
	}
	version := parts[0]
	statusNum, err := strconv.Atoi(parts[1])
	if err != nil {
		statusNum = 0
	}

	startLine := p.i - 1
	resp := &Response{
		Version: version,
		Status:  statusNum,
		Headers: map[string]string{},
		Range:   computeLineRange(line, startLine),
	}

	defer func() {
		resp.Range.EndLine = p.i - 1
		resp.Range.EndCol = len(p.peek())
	}()

	// parse headers, sections, body similar to request
	for !p.eof() {
		raw := p.peek()
		trim := strings.TrimSpace(raw)
		// If new request begins, stop
		if reMethodLine.MatchString(trim) {
			break
		}
		// If section start
		if matches := reSectionLine.FindStringSubmatch(raw); matches != nil {
			sec, err := p.parseSection()
			if err != nil {
				return nil, err
			}
			resp.Sections = append(resp.Sections, *sec)
			continue
		}

		// Header?
		if reHeaderLine.MatchString(raw) {
			p.i++
			k, v := splitHeader(raw)
			resp.Headers[k] = v
			continue
		}

		// Empty line or comment - may be a separator before body
		if trim == "" || strings.HasPrefix(trim, "#") {
			p.i++
			// check next meaningful line
			save := p.i
			nextMeaningful := -1
			for j := p.i; j < p.len; j++ {
				t := strings.TrimSpace(p.lines[j])
				if t == "" || strings.HasPrefix(t, "#") {
					continue
				}
				nextMeaningful = j
				break
			}
			if nextMeaningful == -1 {
				p.i = save
				continue
			}
			nextLine := strings.TrimSpace(p.lines[nextMeaningful])
			if !(reHeaderLine.MatchString(nextLine) || reSectionLine.MatchString(nextLine) || reMethodLine.MatchString(nextLine)) {
				// parse body as everything until next method
				bodyLines := []string{}
				for j := nextMeaningful; j < p.len; j++ {
					t := strings.TrimSpace(p.lines[j])
					if reMethodLine.MatchString(t) {
						break
					}
					bodyLines = append(bodyLines, p.lines[j])
					p.i = j + 1
				}
				resp.Body = strings.Join(bodyLines, "\n")
				return resp, nil
			}
			continue
		}

		// If line looks like body start
		trimL := strings.TrimLeft(raw, " \t")
		if len(trimL) > 0 && (strings.HasPrefix(trimL, "{") || strings.HasPrefix(trimL, "[") || strings.HasPrefix(trimL, "`") || strings.HasPrefix(trimL, "```")) {
			bodyLines := []string{}
			for j := p.i; j < p.len; j++ {
				t := strings.TrimSpace(p.lines[j])
				if reMethodLine.MatchString(t) {
					break
				}
				bodyLines = append(bodyLines, p.lines[j])
				p.i = j + 1
			}
			resp.Body = strings.Join(bodyLines, "\n")
			return resp, nil
		}
		// fallback: treat rest as body
		bodyLines := []string{}
		for j := p.i; j < p.len; j++ {
			t := strings.TrimSpace(p.lines[j])
			if reMethodLine.MatchString(t) {
				break
			}
			if reSectionLine.MatchString(p.lines[j]) {
				break
			}
			bodyLines = append(bodyLines, p.lines[j])
			p.i = j + 1
		}
		resp.Body = strings.Join(bodyLines, "\n")
		return resp, nil
	}
	return resp, nil
}
