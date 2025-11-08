package hurlfile

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"unicode"
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

type Section struct {
	Name      Ranged[string]
	KeyValues map[string]string
	Range     SourceRange
	RawLines  []string
}

type Ranged[T any] struct {
	Value T
	Range SourceRange
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
		Name:      Ranged[string]{Value: name, Range: p.computeStringRange(name, countLeadingWhitespace(line))},
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

	sec.Range.EndLine = p.i - 1
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

func (p *Parser) computeStringRange(s string, leadingChars int) SourceRange {
	startLine := p.i - 1
	return SourceRange{
		StartLine: startLine,
		StartCol:  leadingChars + 1,
		EndLine:   startLine,
		EndCol:    len(s),
	}
}

func countLeadingWhitespace(s string) int {
	cnt := 0
	for _, char := range s {
		if unicode.IsSpace(char) {
			cnt += 1
		} else {
			break
		}
	}

	return cnt
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
