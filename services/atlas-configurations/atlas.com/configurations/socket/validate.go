// Package socket holds the validation rules for a configuration document's
// socket handler/writer tables. It is imported by both the templates and the
// tenants trees, which each contribute a thin adapter building Input from
// their own RestModel, so the rules and their tests exist exactly once.
//
// Every rule here is blocking: Validate's caller turns any returned Issue into
// a 400. There is deliberately no warn tier — task-194 decision 1.
package socket

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	opcodes "github.com/Chronicle20/atlas/libs/atlas-opcodes"
)

// OpCodePattern is the accepted wire form of an opcode: "0x" or "0X" followed
// by one to four hex digits. One digit is real (jms_185_1 carries "0x9") and so
// is three (gms_84_1 carries "0x0A5"), so a two-digit-only pattern would reject
// existing valid data.
var OpCodePattern = regexp.MustCompile(`^0[xX][0-9A-Fa-f]{1,4}$`)

// knownServices is the closed set of socket-service scopes, sourced from
// libs/atlas-opcodes so the vocabulary lives in exactly one place. Adding a
// third socket service means adding it there first.
var knownServices = []string{opcodes.ServiceLogin, opcodes.ServiceChannel}

// Issue is one blocking validation failure. Path is a JSON pointer-ish path
// into the submitted document, used verbatim as the JSON:API error meta.path.
type Issue struct {
	Path    string
	Message string
}

// Binding is one entry of socket.handlers or socket.writers, flattened to the
// fields the rules care about. Validator is empty for writers.
type Binding struct {
	Name      string
	OpCode    string
	Validator string
	Services  []string
}

// Input is a whole socket document, adapted from either tree's RestModel.
type Input struct {
	Handlers            []Binding
	Writers             []Binding
	UnsupportedHandlers []string
	UnsupportedWriters  []string
}

// ParseOpCode parses an opcode in the accepted wire form and reports whether it
// was well-formed. It is the only place a stored opcode string is interpreted.
func ParseOpCode(raw string) (int, bool) {
	if !OpCodePattern.MatchString(raw) {
		return 0, false
	}
	n, err := strconv.ParseInt(raw[2:], 16, 32)
	if err != nil {
		return 0, false
	}
	return int(n), true
}

// Validate returns every blocking issue in the document. An empty slice means
// the document may be stored.
func Validate(in Input) []Issue {
	issues := make([]Issue, 0)
	issues = append(issues, validateCollection("handlers", "handler", true, in.Handlers)...)
	issues = append(issues, validateCollection("writers", "writer", false, in.Writers)...)
	issues = append(issues, validateUnsupported("handlers", in.UnsupportedHandlers, in.Handlers)...)
	issues = append(issues, validateUnsupported("writers", in.UnsupportedWriters, in.Writers)...)
	return issues
}

// validateCollection runs the per-entry rules over one collection. group is the
// JSON key ("handlers"), nameField is the JSON key holding the implementation
// name ("handler"), and needsValidator is true only for handlers.
func validateCollection(group string, nameField string, needsValidator bool, bindings []Binding) []Issue {
	var issues []Issue
	// seen maps (name, numeric opcode) to the raw opcode of its first binding,
	// so the duplicate message can name the canonical form.
	type key struct {
		name string
		code int
	}
	seen := make(map[key]string, len(bindings))

	for i, b := range bindings {
		base := fmt.Sprintf("socket.%s[%d]", group, i)

		if strings.TrimSpace(b.Name) == "" {
			issues = append(issues, Issue{
				Path:    base + "." + nameField,
				Message: "definition name is required",
			})
		}

		code, ok := ParseOpCode(b.OpCode)
		if !ok {
			issues = append(issues, Issue{
				Path:    base + ".opCode",
				Message: fmt.Sprintf("opCode %q must match 0x followed by 1-4 hex digits", b.OpCode),
			})
		} else {
			k := key{name: b.Name, code: code}
			// A name bound to several DISTINCT opcodes is legitimate and common
			// (NoOpHandler sinks four opcodes in gms_95_1). Only the same name at
			// the same numeric opcode is a defect.
			if first, dup := seen[k]; dup {
				issues = append(issues, Issue{
					Path:    base + ".opCode",
					Message: fmt.Sprintf("%q is already bound to opcode %s", b.Name, first),
				})
			} else {
				seen[k] = b.OpCode
			}
		}

		if needsValidator && strings.TrimSpace(b.Validator) == "" {
			issues = append(issues, Issue{
				Path:    base + ".validator",
				Message: fmt.Sprintf("validator is required for handler %q", b.Name),
			})
		}

		for j, s := range b.Services {
			if !isKnownService(s) {
				issues = append(issues, Issue{
					Path:    fmt.Sprintf("%s.services[%d]", base, j),
					Message: fmt.Sprintf("unknown service %q; expected one of %s", s, strings.Join(knownServices, ", ")),
				})
			}
		}
	}
	return issues
}

// validateUnsupported enforces FR-11.3 (a name is never both defined and
// unsupported) and rejects a name listed twice.
func validateUnsupported(group string, names []string, bindings []Binding) []Issue {
	var issues []Issue
	defined := make(map[string]bool, len(bindings))
	for _, b := range bindings {
		defined[b.Name] = true
	}
	seen := make(map[string]bool, len(names))
	for i, n := range names {
		path := fmt.Sprintf("socket.unsupported.%s[%d]", group, i)
		if strings.TrimSpace(n) == "" {
			issues = append(issues, Issue{Path: path, Message: "unsupported entry name is required"})
			continue
		}
		if defined[n] {
			issues = append(issues, Issue{
				Path:    path,
				Message: fmt.Sprintf("%q is marked unsupported but is also defined in socket.%s", n, group),
			})
		}
		if seen[n] {
			issues = append(issues, Issue{
				Path:    path,
				Message: fmt.Sprintf("%q is listed more than once", n),
			})
		}
		seen[n] = true
	}
	return issues
}

func isKnownService(s string) bool {
	for _, k := range knownServices {
		if s == k {
			return true
		}
	}
	return false
}
