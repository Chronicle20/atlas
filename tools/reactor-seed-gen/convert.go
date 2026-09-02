package main

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var (
	stateGuardPattern = regexp.MustCompile(`^if \(rm\.getReactor\(\)\.getState\(\) !== (\d+)\) \{\n\s*return\n\s*\}\n?`)
	nullGuardPattern  = regexp.MustCompile(`(?s)^if \(rm\.getEventInstance\(\) != null\) \{\n(.*)\n\}$`)
	eimBindPattern    = regexp.MustCompile(`^var \w+ = rm\.(?:getPlayer\(\)\.)?getEventInstance\(\);$`)
	forHeaderPattern  = regexp.MustCompile(`^for \(var \w+ = 0; \w+ < (\d+); \w+\+\+\) \{$`)

	getIntPropPattern = regexp.MustCompile(`^var (\w+) = (\w+)\.getIntProperty\("([^"]+)"\);$`)
	addPattern        = regexp.MustCompile(`^var (\w+) = (\w+) \+ (\d+);$`)
	setIntPropPattern = regexp.MustCompile(`^(\w+)\.setIntProperty\("([^"]+)", (\w+)\);$`)
	setPropPattern    = regexp.MustCompile(`^(?:\w+|rm\.getEventInstance\(\))\.setProperty\("([^"]+)", "([^"]*)"\);?$`)

	dropItemsPattern    = regexp.MustCompile(`^rm\.dropItems\(([^)]*)\);?$`)
	sprayItemsPattern   = regexp.MustCompile(`^rm\.sprayItems\(([^)]*)\);?$`)
	spawnMonsterPattern = regexp.MustCompile(`^rm\.spawnMonster\((.*)\);?$`)
	weakenPattern       = regexp.MustCompile(`^rm\.weakenAreaBoss\((-?\d+), "([^"]*)"\)\s*;?$`)

	intLiteralPattern  = regexp.MustCompile(`^-?\d+$`)
	boolLiteralPattern = regexp.MustCompile(`^(?:true|false)$`)
)

// convertScript walks a parsed inventory section against the whitelisted
// grammar. It sets ReactorId, HitRules and ActRules; Description is left
// empty and filled in by main.go from describe() (Task 10), so the grammar
// and the comment cleaner stay independently testable. Any line matching no
// whitelisted form is a hard error naming the reactor id and the line —
// there is no skip-and-report and no partial output.
func convertScript(s sourceScript) (scriptDoc, error) {
	hit, err := convertBody(s.Id, s.HitBody)
	if err != nil {
		return scriptDoc{}, err
	}
	act, err := convertBody(s.Id, s.ActBody)
	if err != nil {
		return scriptDoc{}, err
	}
	return scriptDoc{
		ReactorId: s.Id,
		HitRules:  hit,
		ActRules:  act,
	}, nil
}

// convertBody turns one function body into its rules.
//
// Every Tier-1 body produces at most one rule; a leading state guard, when
// present, becomes that rule's condition. An empty body produces an empty,
// non-nil rule slice.
func convertBody(reactorId, body string) ([]ruleDoc, error) {
	body = strings.TrimSpace(body)
	if body == "" {
		return []ruleDoc{}, nil
	}

	var conditions []condDoc
	if m := stateGuardPattern.FindStringSubmatch(body); m != nil {
		conditions = append(conditions, condDoc{Type: "reactor_state", Operator: "=", Value: m[1]})
		body = strings.TrimSpace(body[len(m[0]):])
	}

	if m := nullGuardPattern.FindStringSubmatch(body); m != nil {
		body = strings.TrimSpace(m[1])
	}

	ops, err := convertStatements(reactorId, body)
	if err != nil {
		return nil, err
	}

	return []ruleDoc{{
		Id:         ruleId(ops),
		Conditions: conditions,
		Operations: ops,
	}}, nil
}

// convertStatements walks the (guard-stripped) statement lines of a body and
// produces the operations they whitelist to.
func convertStatements(reactorId, body string) ([]opDoc, error) {
	lines := splitStatements(body)
	var ops []opDoc

	for i := 0; i < len(lines); i++ {
		line := lines[i]

		switch {
		case eimBindPattern.MatchString(line):
			// The event-instance binding is erased; later statements refer
			// to it by name, matched independent of scope tracking.
			continue

		case forHeaderPattern.MatchString(line):
			m := forHeaderPattern.FindStringSubmatch(line)
			factor, err := strconv.Atoi(m[1])
			if err != nil {
				return nil, fmt.Errorf("reactor %s: invalid loop bound: %s", reactorId, line)
			}
			end := indexOfClose(lines, i+1)
			if end == -1 {
				return nil, fmt.Errorf("reactor %s: unterminated loop: %s", reactorId, line)
			}
			for j := i + 1; j < end; j++ {
				inner := lines[j]
				mm := spawnMonsterPattern.FindStringSubmatch(inner)
				if mm == nil {
					return nil, fmt.Errorf("reactor %s: only spawn_monster may appear inside a loop: %s", reactorId, inner)
				}
				op, err := parseSpawnMonster(reactorId, inner, mm[1], factor)
				if err != nil {
					return nil, err
				}
				ops = append(ops, op)
			}
			i = end
			continue

		case getIntPropPattern.MatchString(line):
			op, consumed, err := parseIncrementIdiom(reactorId, lines, i)
			if err != nil {
				return nil, err
			}
			ops = append(ops, op)
			i += consumed
			continue

		case setPropPattern.MatchString(line):
			m := setPropPattern.FindStringSubmatch(line)
			ops = append(ops, opDoc{Type: "update_pq_state", Params: map[string]string{
				"updates": m[1] + "=" + m[2],
			}})
			continue

		case dropItemsPattern.MatchString(line):
			m := dropItemsPattern.FindStringSubmatch(line)
			op, err := parseDropOrSpray(reactorId, "drop_items", line, m[1])
			if err != nil {
				return nil, err
			}
			ops = append(ops, op)
			continue

		case sprayItemsPattern.MatchString(line):
			m := sprayItemsPattern.FindStringSubmatch(line)
			op, err := parseDropOrSpray(reactorId, "spray_items", line, m[1])
			if err != nil {
				return nil, err
			}
			ops = append(ops, op)
			continue

		case spawnMonsterPattern.MatchString(line):
			m := spawnMonsterPattern.FindStringSubmatch(line)
			op, err := parseSpawnMonster(reactorId, line, m[1], 1)
			if err != nil {
				return nil, err
			}
			ops = append(ops, op)
			continue

		case weakenPattern.MatchString(line):
			m := weakenPattern.FindStringSubmatch(line)
			ops = append(ops, opDoc{Type: "weaken_area_boss", Params: map[string]string{
				"monsterId": m[1],
				"message":   m[2],
			}})
			continue

		default:
			return nil, fmt.Errorf("reactor %s: unrecognized statement: %s", reactorId, line)
		}
	}

	return ops, nil
}

// parseIncrementIdiom matches the three-statement unit
// (getIntProperty(k) -> var x = <that> + 1 -> setIntProperty(k, x)) starting
// at lines[i]. It returns the resulting operation and the number of extra
// lines consumed beyond lines[i] (always 2 on success).
func parseIncrementIdiom(reactorId string, lines []string, i int) (opDoc, int, error) {
	if i+2 >= len(lines) {
		return opDoc{}, 0, fmt.Errorf("reactor %s: incomplete increment idiom: %s", reactorId, lines[i])
	}

	m1 := getIntPropPattern.FindStringSubmatch(lines[i])
	nowVar, eimVar, key := m1[1], m1[2], m1[3]

	m2 := addPattern.FindStringSubmatch(lines[i+1])
	if m2 == nil || m2[2] != nowVar {
		return opDoc{}, 0, fmt.Errorf("reactor %s: malformed increment idiom: %s", reactorId, lines[i+1])
	}
	nextVar, addend := m2[1], m2[3]
	if addend != "1" {
		return opDoc{}, 0, fmt.Errorf("reactor %s: increment idiom addend must be 1, got %s: %s", reactorId, addend, lines[i+1])
	}

	m3 := setIntPropPattern.FindStringSubmatch(lines[i+2])
	if m3 == nil || m3[1] != eimVar || m3[2] != key || m3[3] != nextVar {
		return opDoc{}, 0, fmt.Errorf("reactor %s: malformed increment idiom: %s", reactorId, lines[i+2])
	}

	return opDoc{Type: "update_pq_state", Params: map[string]string{"increments": key}}, 2, nil
}

// parseDropOrSpray parses the arguments of rm.dropItems(...) / rm.sprayItems(...).
func parseDropOrSpray(reactorId, opType, line, argsStr string) (opDoc, error) {
	argsStr = strings.TrimSpace(argsStr)
	if argsStr == "" {
		return opDoc{Type: opType, Params: nil}, nil
	}

	args, err := splitLiteralArgs(argsStr)
	if err != nil {
		return opDoc{}, fmt.Errorf("reactor %s: %s: %s", reactorId, err, line)
	}
	if len(args) != 4 && len(args) != 5 {
		return opDoc{}, fmt.Errorf("reactor %s: %s takes 0, 4 or 5 arguments: %s", reactorId, opType, line)
	}
	if !boolLiteralPattern.MatchString(args[0]) {
		return opDoc{}, fmt.Errorf("reactor %s: %s meso argument is not a literal: %s", reactorId, opType, line)
	}
	for _, a := range args[1:] {
		if !intLiteralPattern.MatchString(a) {
			return opDoc{}, fmt.Errorf("reactor %s: %s argument is not a literal: %s", reactorId, opType, line)
		}
	}

	keys := []string{"meso", "mesoChance", "mesoMin", "mesoMax", "minItems"}
	params := make(map[string]string, len(args))
	for i, a := range args {
		params[keys[i]] = a
	}
	return opDoc{Type: opType, Params: params}, nil
}

// parseSpawnMonster parses the arguments of rm.spawnMonster(...). factor is
// the loop unroll multiplier (1 outside a loop); the resulting count is
// count * factor, and the count key is emitted whenever the call passed an
// explicit count or factor != 1, so a loop can never silently lose its
// multiplier.
func parseSpawnMonster(reactorId, line, argsStr string, factor int) (opDoc, error) {
	args, err := splitLiteralArgs(argsStr)
	if err != nil {
		return opDoc{}, fmt.Errorf("reactor %s: %s: %s", reactorId, err, line)
	}
	if len(args) != 1 && len(args) != 2 && len(args) != 4 {
		return opDoc{}, fmt.Errorf("reactor %s: spawnMonster takes 1, 2 or 4 arguments: %s", reactorId, line)
	}
	if !intLiteralPattern.MatchString(args[0]) {
		return opDoc{}, fmt.Errorf("reactor %s: spawnMonster monsterId is not a literal: %s", reactorId, line)
	}

	params := map[string]string{"monsterId": args[0]}

	count := 1
	hadCount := len(args) >= 2
	if hadCount {
		if !intLiteralPattern.MatchString(args[1]) {
			return opDoc{}, fmt.Errorf("reactor %s: spawnMonster count is not a literal: %s", reactorId, line)
		}
		count, _ = strconv.Atoi(args[1])
	}
	count *= factor
	if hadCount || factor != 1 {
		params["count"] = strconv.Itoa(count)
	}

	if len(args) == 4 {
		if !intLiteralPattern.MatchString(args[2]) || !intLiteralPattern.MatchString(args[3]) {
			return opDoc{}, fmt.Errorf("reactor %s: spawnMonster x/y is not a literal: %s", reactorId, line)
		}
		params["x"] = args[2]
		params["y"] = args[3]
	}

	return opDoc{Type: "spawn_monster", Params: params}, nil
}

// ruleId is the distinct operation types, in first-appearance order, joined
// with "_".
func ruleId(ops []opDoc) string {
	seen := make(map[string]bool, len(ops))
	parts := make([]string, 0, len(ops))
	for _, op := range ops {
		if seen[op.Type] {
			continue
		}
		seen[op.Type] = true
		parts = append(parts, op.Type)
	}
	return strings.Join(parts, "_")
}

// splitStatements splits a body into its non-empty, individually-trimmed
// lines. Indentation carried over from the source markdown is irrelevant to
// the grammar, so every line is trimmed independently.
func splitStatements(body string) []string {
	raw := strings.Split(body, "\n")
	lines := make([]string, 0, len(raw))
	for _, l := range raw {
		l = strings.TrimSpace(l)
		if l == "" {
			continue
		}
		lines = append(lines, l)
	}
	return lines
}

// indexOfClose returns the index of the first bare "}" line at or after
// start, or -1 if none exists.
func indexOfClose(lines []string, start int) int {
	for i := start; i < len(lines); i++ {
		if lines[i] == "}" {
			return i
		}
	}
	return -1
}

// splitLiteralArgs splits a raw argument-list string on commas and trims
// each argument. It does not handle nested strings or parens; callers that
// may see those (weakenAreaBoss's message) use a dedicated regex instead.
func splitLiteralArgs(s string) ([]string, error) {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			return nil, fmt.Errorf("empty argument")
		}
		out = append(out, p)
	}
	return out, nil
}
