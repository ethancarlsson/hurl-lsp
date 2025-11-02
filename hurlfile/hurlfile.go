package hurlfile

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"
)

func Parse(uri string) (*HurlFile, error) {
	f, err := os.OpenFile(strings.Replace(uri, "file://", "", 1), os.O_RDONLY, os.ModePerm)
	if err != nil {
		return &HurlFile{}, fmt.Errorf("couldn't open file: %w", err)
	}

	parser, err := NewParser(f)
	if err != nil {
		return &HurlFile{}, fmt.Errorf("couldn't create parser: %w", err)
	}

	hurlFile, err := parser.Parse()
	if err != nil {
		return &HurlFile{}, fmt.Errorf("parse file: %w", err)
	}

	return hurlFile, nil
}

// AST structures
type HurlFile struct {
	Entries []Entry
	Range   SourceRange
}

type SourceRange struct {
	StartLine int
	StartCol  int
	EndLine   int
	EndCol    int
}

type Entry struct {
	Request  Request
	Response *Response
	Range    SourceRange
}

type Request struct {
	Method   string
	Target   string
	Headers  map[string]string
	Sections []Section
	Body     Body
	Range    SourceRange
}

type Body struct {
	Raw   string
	Range SourceRange
}

type Response struct {
	Version  string
	Status   int
	Headers  map[string]string
	Sections []Section
	Body     string
	Range    SourceRange
}

type Section struct {
	Name      string
	KeyValues map[string]string
	Range     SourceRange
	RawLines  []string
}

// Parser
type Parser struct {
	lines []string
	i     int
	len   int
}

func NewParser(r io.Reader) (*Parser, error) {
	scanner := bufio.NewScanner(r)
	// Keep line endings content but trim trailing \r if present
	var lines []string
	for scanner.Scan() {
		line := scanner.Text()
		lines = append(lines, line)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return &Parser{
		lines: lines,
		i:     0,
		len:   len(lines),
	}, nil
}

func (p *Parser) eof() bool { return p.i >= p.len }

func (p *Parser) peek() string {
	if p.eof() {
		return ""
	}
	return p.lines[p.i]
}

func (p *Parser) next() string {
	if p.eof() {
		return ""
	}
	l := p.lines[p.i]
	p.i++
	return l
}

func (p *Parser) skipCommentsAndEmpty() {
	for !p.eof() {
		line := strings.TrimSpace(p.peek())
		if line == "" || strings.HasPrefix(line, "#") {
			p.i++
			continue
		}
		break
	}
}

// Recognizers
var reMethodLine = regexp.MustCompile(`^[A-Z]+\b(?:\s+.+)?$`)
var reResponseLine = regexp.MustCompile(`^HTTP/\d+(?:\.\d+)?\s+\d+`) // e.g. HTTP/1.1 200
var reHeaderLine = regexp.MustCompile(`^[^:\s][^:]*\s*:\s*.*$`)
var reSectionLine = regexp.MustCompile(`^\s*\[([A-Za-z0-9_-]+)\]\s*$`)

func (p *Parser) Parse() (*HurlFile, error) {
	h := &HurlFile{}
	for {
		p.skipCommentsAndEmpty()
		if p.eof() {
			break
		}
		// Expect a request line (METHOD ...)
		// If not a method line, but maybe stray comments/lines - treat as error or skip
		line := strings.TrimSpace(p.peek())
		if reMethodLine.MatchString(line) {
			entry, err := p.parseEntry()
			if err != nil {
				return nil, err
			}
			h.Entries = append(h.Entries, *entry)
			continue
		}
		// If we find something else (like a response without request), skip it safely
		// But in general Hurl files start with a request.
		// We'll skip unexpected lines to be forgiving.
		p.i++
	}

	h.Range.EndCol = len(p.peek())
	h.Range.EndLine = len(p.lines)

	return h, nil
}

func (p *Parser) parseEntry() (*Entry, error) {
	req, err := p.parseRequest()
	if err != nil {
		return nil, err
	}

	// After request, there may be an immediate response block
	p.skipCommentsAndEmpty()
	var resp *Response
	if !p.eof() {
		ln := strings.TrimSpace(p.peek())
		if reResponseLine.MatchString(ln) {
			r, err := p.parseResponse()
			if err != nil {
				return nil, err
			}
			resp = r
		}
	}
	entry := &Entry{
		Request:  *req,
		Response: resp,
		Range: SourceRange{
			StartLine: req.Range.StartLine,
			StartCol:  req.Range.StartCol,
			EndCol:    req.Range.EndCol,
			EndLine:   req.Range.EndLine,
		},
	}

	if resp != nil {
		entry.Range.EndCol = resp.Range.EndCol
		entry.Range.EndLine = resp.Range.EndLine
	}

	return entry, nil
}

func (p *Parser) parseRequest() (*Request, error) {
	// Expect method line
	line := strings.TrimSpace(p.next())
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

	startLine := p.i - 1
	req := &Request{
		Method:  method,
		Target:  target,
		Headers: map[string]string{},
		Range:   computeLineRange(line, startLine),
	}

	defer func() {
		req.Range.EndLine = p.i - 1
		req.Range.EndCol = len(p.peek())
		req.Body.Range.EndLine = p.i - 1
		req.Body.Range.EndCol = len(p.peek())
	}()
	// Now parse headers, sections, and body
	// We'll read until we hit:
	// - blank line followed by something that doesn't look like a header/section (body start)
	// - a line that starts a section [Name]
	// - a response line (HTTP/...)
	// - a next request method line (start of next entry)
	// - EOF
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
			kv := raw
			p.i++
			k, v := splitHeader(kv)
			req.Headers[k] = v
			continue
		}

		// If line is empty or comment: consume and continue (could be separator before body)
		if trim == "" || strings.HasPrefix(trim, "#") {
			// consume it
			p.i++
			// peek next non-comment non-empty line to see if it's a header/section/response/method
			// If the next meaningful line is not header/section/response/method, parse the rest as body
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
				// EOF: nothing more — no body
				p.i = save
				continue
			}
			nextLine := strings.TrimSpace(p.lines[nextMeaningful])
			// If the next line is not a header/section/response/method, treat everything from nextMeaningful as body
			req.Body.Range.StartLine = p.i
			if !(reHeaderLine.MatchString(nextLine) || reSectionLine.MatchString(nextLine) || reResponseLine.MatchString(nextLine) || reMethodLine.MatchString(nextLine)) {
				// collect body from nextMeaningful until we hit a line that is a method/response that starts a new entry
				bodyLines := []string{}
				for j := nextMeaningful; j < p.len; j++ {
					t := strings.TrimSpace(p.lines[j])
					if reMethodLine.MatchString(t) || reResponseLine.MatchString(t) {
						break
					}
					// Stop body if we encounter a section that would belong to next entry? in Hurl, sections are part of request/response; but we're conservative
					// We treat everything until blank line preceding a method/response as body.
					// Add line
					bodyLines = append(bodyLines, p.lines[j])
					p.i = j + 1
				}
				req.Body.Raw = strings.Join(bodyLines, "\n")

				return req, nil
			}
			// else it's still part of headers/sections - continue loop
			continue
		}

		// If line looks like start of a body (starts with `{` or `[` or backtick or triple-backtick), treat as body directly
		trimL := strings.TrimLeft(raw, " \t")
		if len(trimL) > 0 && (strings.HasPrefix(trimL, "{") || strings.HasPrefix(trimL, "[") || strings.HasPrefix(trimL, "`") || strings.HasPrefix(trimL, "```")) {
			bodyLines := []string{}

			req.Body.Range.StartLine = p.i
			for j := p.i; j < p.len; j++ {
				t := strings.TrimSpace(p.lines[j])
				if reMethodLine.MatchString(t) || reResponseLine.MatchString(t) {
					break
				}
				// stop if we hit a section that clearly belongs to next entry? (rare)
				if reSectionLine.MatchString(p.lines[j]) {
					// assume section is not part of body: break
					break
				}
				bodyLines = append(bodyLines, p.lines[j])
				p.i = j + 1
			}

			req.Body.Raw = strings.Join(bodyLines, "\n")

			return req, nil
		}

		// If none of the above matched and it's not a header, treat it conservatively as "unknown line" and consume it (to avoid infinite loop)
		// but we also try: if it's something like json-object start but not recognized, capture as body
		if !reHeaderLine.MatchString(raw) {
			// Put back one line and parse rest as body (safe fallback)
			// Actually we already are at the start of such a line; treat from here as body until next method/response.
			bodyLines := []string{}
			for j := p.i; j < p.len; j++ {
				t := strings.TrimSpace(p.lines[j])
				if reMethodLine.MatchString(t) || reResponseLine.MatchString(t) {
					break
				}
				// Also break if we find a section that may belong to next entry
				if reSectionLine.MatchString(p.lines[j]) {
					break
				}
				bodyLines = append(bodyLines, p.lines[j])
				p.i = j + 1
			}
			req.Body.Raw = strings.Join(bodyLines, "\n")
			return req, nil
		}
	}

	return req, nil
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

// parseSection assumes current line is [Name]
func (p *Parser) parseSection() (*Section, error) {
	line := p.next()
	m := reSectionLine.FindStringSubmatch(line)
	name := ""
	if len(m) > 1 {
		name = m[1]
	}

	startLine := p.i - 1
	sec := &Section{
		Name:      name,
		KeyValues: map[string]string{},
		Range:     computeLineRange(line, startLine),
	}
	// Collect following key-value lines until blank or another section / request/response starts
	for !p.eof() {
		raw := p.peek()
		trim := strings.TrimSpace(raw)
		if trim == "" || strings.HasPrefix(trim, "#") {
			// consume and continue
			p.i++
			continue
		}
		// stop if next is another section or request/response start
		if reSectionLine.MatchString(raw) || reMethodLine.MatchString(trim) || reResponseLine.MatchString(trim) {
			break
		}
		// parse key-value: expect "key : value" or "key: value"
		if reHeaderLine.MatchString(raw) {
			p.i++
			k, v := splitHeader(raw)
			sec.KeyValues[k] = v
			continue
		}
		// if not a key-value line, treat as raw line included in section raw content and consume
		sec.RawLines = append(sec.RawLines, raw)
		p.i++
	}

	sec.Range.EndLine = p.i
	sec.Range.EndCol = len(p.peek())

	return sec, nil
}

func splitHeader(line string) (string, string) {
	// split at first ':'
	idx := strings.Index(line, ":")
	if idx < 0 {
		return strings.TrimSpace(line), ""
	}
	k := strings.TrimSpace(line[:idx])
	v := strings.TrimSpace(line[idx+1:])
	return k, v
}

func (p *Parser) currentLineNum() int {
	// 1-based line numbers
	return p.i + 1
}

func computeLineRange(line string, lineNum int) SourceRange {
	start := strings.IndexFunc(line, func(r rune) bool { return !isSpace(r) })
	if start == -1 {
		start = 0
	}
	end := len(line)
	return SourceRange{
		StartLine: lineNum,
		StartCol:  start + 1,
		EndLine:   lineNum,
		EndCol:    end,
	}
}

func mergeRanges(r1, r2 SourceRange) SourceRange {
	if r1.StartLine == 0 {
		return r2
	}
	if r2.StartLine == 0 {
		return r1
	}
	if r2.StartLine < r1.StartLine {
		r1.StartLine = r2.StartLine
		r1.StartCol = r2.StartCol
	}
	if r2.EndLine > r1.EndLine || (r2.EndLine == r1.EndLine && r2.EndCol > r1.EndCol) {
		r1.EndLine = r2.EndLine
		r1.EndCol = r2.EndCol
	}
	return r1
}

func isSpace(r rune) bool {
	return r == ' ' || r == '\t'
}
