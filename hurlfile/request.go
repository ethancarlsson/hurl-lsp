package hurlfile

import (
	"fmt"
	"strings"
)

type Method struct {
	Name  string
	Range SourceRange
}

type Target struct {
	Target string
	Range  SourceRange
}

type Request struct {
	Method   Method
	Target   Target
	Headers  Ranged[map[string]string]
	Sections []Section
	Body     Ranged[[]string]
	Range    SourceRange
}

func (p *Parser) parseRequest() (*Request, error) {
	// Expect method line
	untrimmedLine := p.next()
	line := strings.TrimSpace(untrimmedLine)
	// method is first token
	parts := strings.Fields(line)
	if len(parts) == 0 {
		return nil, fmt.Errorf("expected method line but got empty")
	}
	method := parts[0]
	target := ""
	if len(parts) > 1 {
		target = strings.TrimSpace(line[len(method):])
	}

	leadingWhitespace := countLeadingWhitespace(untrimmedLine)

	startLine := p.i - 1
	headers := make(map[string]string)
	req := &Request{
		Method: Method{
			Name: method,
			Range: SourceRange{
				StartLine: startLine,
				StartCol:  leadingWhitespace,
				EndCol:    leadingWhitespace + len(method),
				EndLine:   startLine,
			},
		},
		Target: Target{
			Target: target,
			Range: SourceRange{
				StartLine: startLine,
				// Could be more than +1, but we can consider all whitespace after the first to be part of the target
				StartCol: leadingWhitespace + len(method) + 1,
				EndCol:   len(untrimmedLine),
				EndLine:  startLine,
			},
		},
		Headers: Ranged[map[string]string]{
			Value: headers,
		},
		Range: computeLineRange(line, startLine),
	}

	defer func() {
		req.Range.EndLine = p.i - 1
		req.Range.EndCol = len(p.peek())
		req.Body.Range.EndLine = p.i - 1
		req.Body.Range.EndCol = len(p.peek())
	}()

	for !p.eof() {
		// Peek but do NOT skip comments/empty here because empty lines may indicate body begins
		raw := p.peek()
		trim := strings.TrimSpace(raw)

		// If response or next request begins, break
		if reResponseLine.MatchString(trim) {
			break
		}

		if reMethodLine.MatchString(trim) {
			// If a new method is starting, this request has no body (unless previously detected)
			break
		}

		// If section start
		if matches := reSectionLine.FindStringSubmatch(raw); matches != nil {
			sec, err := p.parseSection()
			if err != nil {
				return nil, err
			}
			req.Sections = append(req.Sections, *sec)
			continue
		}

		// Header line?
		if reHeaderLine.MatchString(raw) {
			// It should never be zero unless there are no headers
			if req.Headers.Range.StartLine == 0 {
				req.Headers.Range.StartLine = p.i
				req.Headers.Range.EndLine = p.i
			}

			p.i++
			k, v := splitHeader(raw)
			req.Headers.Value[k] = v
			req.Headers.Range.EndCol = len(raw) - 1
			continue
		}

		// consume empty or comment line
		if trim == "" || strings.HasPrefix(trim, "#") {
			p.i++
			continue
		}

		req.Body.Value = append(req.Body.Value, raw)
		if req.Body.Range.StartLine == 0 {
			req.Body.Range.StartLine = p.i
			req.Body.Range.StartCol = 0
		}

		req.Body.Range.EndCol = len(raw) - 1
		req.Body.Range.EndLine = p.i - 1
		p.i++
	}

	return req, nil
}
