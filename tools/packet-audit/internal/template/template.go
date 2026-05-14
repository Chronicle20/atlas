package template

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Template holds the decoded socket handler/writer table from a configuration
// template JSON file (e.g. template_gms_95_1.json).
//
// NOTE: The real template can assign multiple writer names to the same opCode
// (e.g. 0x00 → AuthSuccess, AuthTemporaryBan, AuthPermanentBan). The writers
// map here stores the last-seen name for each opcode, which is sufficient for
// Phase A opcode-presence checks. Task 12 (reverse writer→opcode lookup) will
// need to iterate the raw slice instead of using this map.
type Template struct {
	Region       string
	MajorVersion uint16
	MinorVersion uint16

	handlers map[int]string
	writers  map[int]string
}

type rawTemplate struct {
	Region       string `json:"region"`
	MajorVersion uint16 `json:"majorVersion"`
	MinorVersion uint16 `json:"minorVersion"`
	Socket       struct {
		Handlers []struct {
			OpCode  string `json:"opCode"`
			Handler string `json:"handler"`
		} `json:"handlers"`
		Writers []struct {
			OpCode string `json:"opCode"`
			Writer string `json:"writer"`
		} `json:"writers"`
	} `json:"socket"`
}

// Load reads and parses a template JSON file at path.
func Load(path string) (*Template, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var r rawTemplate
	if err := json.Unmarshal(b, &r); err != nil {
		return nil, err
	}
	t := &Template{
		Region:       r.Region,
		MajorVersion: r.MajorVersion,
		MinorVersion: r.MinorVersion,
		handlers:     make(map[int]string),
		writers:      make(map[int]string),
	}
	for _, h := range r.Socket.Handlers {
		op, err := parseOp(h.OpCode)
		if err != nil {
			return nil, fmt.Errorf("handler %s: %w", h.Handler, err)
		}
		t.handlers[op] = h.Handler
	}
	for _, w := range r.Socket.Writers {
		op, err := parseOp(w.OpCode)
		if err != nil {
			return nil, fmt.Errorf("writer %s: %w", w.Writer, err)
		}
		t.writers[op] = w.Writer
	}
	return t, nil
}

// Handler returns the handler name registered for op, and whether one exists.
func (t *Template) Handler(op int) (string, bool) {
	v, ok := t.handlers[op]
	return v, ok
}

// Writer returns the writer name registered for op, and whether one exists.
func (t *Template) Writer(op int) (string, bool) {
	v, ok := t.writers[op]
	return v, ok
}

// Writers returns a copy of the opcode→writer-name map.
func (t *Template) Writers() map[int]string { return t.writers }

// Handlers returns a copy of the opcode→handler-name map.
func (t *Template) Handlers() map[int]string { return t.handlers }

func parseOp(s string) (int, error) {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "0x") || strings.HasPrefix(s, "0X") {
		n, err := strconv.ParseInt(s[2:], 16, 32)
		return int(n), err
	}
	n, err := strconv.ParseInt(s, 10, 32)
	return int(n), err
}
