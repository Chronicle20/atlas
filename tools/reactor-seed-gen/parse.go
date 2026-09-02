package main

import (
	"fmt"
	"regexp"
	"strings"
)

// sourceScript is one reactor's verbatim inventory section.
type sourceScript struct {
	Id      string // e.g. "2119000"
	Comment string // raw Source comment text, "" when the section says (no comment in source)
	HitBody string // statements inside function hit() { ... }, "" when absent or empty
	ActBody string // statements inside function act() { ... }, "" when absent or empty
}

var (
	headingPattern    = regexp.MustCompile("^### `(\\S+)`\\s*$")
	commentPattern    = regexp.MustCompile(`^\*\*Source comment:\*\* (.*)$`)
	noCommentLine     = "*(no comment in source)*"
	fenceOpen         = "```javascript"
	fenceClose        = "```"
	funcHeaderPattern = regexp.MustCompile(`function\s+(\w+)\s*\(\s*\)\s*\{`)
)

// parseInventory reads the tier-1 inventory markdown and returns one
// sourceScript per "### `<id>`" section, in file order.
func parseInventory(b []byte) ([]sourceScript, error) {
	lines := strings.Split(string(b), "\n")

	var sections [][]string
	var ids []string
	for i := 0; i < len(lines); i++ {
		m := headingPattern.FindStringSubmatch(lines[i])
		if m == nil {
			continue
		}
		ids = append(ids, m[1])
		start := i + 1
		end := len(lines)
		for j := start; j < len(lines); j++ {
			if headingPattern.MatchString(lines[j]) {
				end = j
				break
			}
		}
		sections = append(sections, lines[start:end])
		i = end - 1
	}

	scripts := make([]sourceScript, 0, len(sections))
	for idx, body := range sections {
		id := ids[idx]
		script, err := parseSection(id, body)
		if err != nil {
			return nil, err
		}
		scripts = append(scripts, script)
	}
	return scripts, nil
}

// parseSection parses the lines belonging to a single "### `<id>`" heading.
func parseSection(id string, lines []string) (sourceScript, error) {
	script := sourceScript{Id: id}

	i := 0
	for i < len(lines) && strings.TrimSpace(lines[i]) == "" {
		i++
	}
	if i >= len(lines) {
		return sourceScript{}, fmt.Errorf("reactor %s: heading with no fence", id)
	}

	line := lines[i]
	switch {
	case strings.TrimSpace(line) == noCommentLine:
		script.Comment = ""
		i++
	default:
		if m := commentPattern.FindStringSubmatch(line); m != nil {
			script.Comment = m[1]
			i++
		}
	}

	for i < len(lines) && strings.TrimSpace(lines[i]) == "" {
		i++
	}
	if i >= len(lines) || strings.TrimSpace(lines[i]) != fenceOpen {
		return sourceScript{}, fmt.Errorf("reactor %s: heading with no fence", id)
	}
	i++

	fenceEnd := -1
	for j := i; j < len(lines); j++ {
		if strings.TrimSpace(lines[j]) == fenceClose {
			fenceEnd = j
			break
		}
	}
	if fenceEnd == -1 {
		return sourceScript{}, fmt.Errorf("reactor %s: unterminated fence", id)
	}

	body := lines[i:fenceEnd]
	if err := extractFunctions(id, body, &script); err != nil {
		return sourceScript{}, err
	}

	return script, nil
}

// extractFunctions scans the raw fence body for function hit() {...} and
// function act() {...} definitions, matching braces by depth counting.
func extractFunctions(id string, lines []string, script *sourceScript) error {
	text := strings.Join(lines, "\n")

	i := 0
	for i < len(text) {
		rest := text[i:]
		m := funcHeaderPattern.FindStringSubmatchIndex(rest)
		if m == nil {
			// no more function headers in the remaining text
			break
		}
		headerEnd := i + m[1] // position just past the opening '{'
		name := rest[m[2]:m[3]]

		depth := 1
		pos := headerEnd
		bodyStart := headerEnd
		bodyEnd := -1
		for pos < len(text) {
			switch text[pos] {
			case '{':
				depth++
			case '}':
				depth--
				if depth == 0 {
					bodyEnd = pos
				}
			}
			if bodyEnd != -1 {
				break
			}
			pos++
		}
		if bodyEnd == -1 {
			return fmt.Errorf("reactor %s: unterminated function %s", id, name)
		}

		body := strings.TrimSpace(text[bodyStart:bodyEnd])

		switch name {
		case "hit":
			script.HitBody = body
		case "act":
			script.ActBody = body
		default:
			return fmt.Errorf("reactor %s: unknown function %s", id, name)
		}

		i = bodyEnd + 1
	}

	return nil
}
