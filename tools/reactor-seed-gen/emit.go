package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// seedDirs is the literal fan-out list. Keeping it a literal here (rather
// than globbing deploy/seed) is deliberate: adopting the shared catalog root
// later (design §9) is then a change to this one variable plus a git rm.
var seedDirs = []string{
	"gms/12_1", "gms/48_1", "gms/61_1", "gms/72_1", "gms/79_1",
	"gms/83_1", "gms/84_1", "gms/87_1", "gms/92_1", "gms/95_1",
	"jms/185_1",
}

// marshalEnvelope renders one script as its JSON:API seed file. Nested
// map[string]any (not structs) so encoding/json's key sorting reproduces the
// existing corpus's alphabetical ordering byte for byte.
func marshalEnvelope(d scriptDoc) ([]byte, error) {
	envelope := map[string]any{
		"data": map[string]any{
			"id":   d.ReactorId,
			"type": "reactor-action",
			"attributes": map[string]any{
				"reactorId":   d.ReactorId,
				"description": d.Description,
				"hitRules":    rulesToAny(d.HitRules),
				"actRules":    rulesToAny(d.ActRules),
			},
		},
	}

	b, err := json.MarshalIndent(envelope, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshalling reactor %s: %w", d.ReactorId, err)
	}
	b = append(b, '\n')
	return b, nil
}

// rulesToAny converts a rule slice into []any so a nil or empty slice
// marshals as [] rather than null.
func rulesToAny(rules []ruleDoc) []any {
	out := make([]any, 0, len(rules))
	for _, r := range rules {
		out = append(out, ruleToAny(r))
	}
	return out
}

func ruleToAny(r ruleDoc) map[string]any {
	conditions := make([]any, 0, len(r.Conditions))
	for _, c := range r.Conditions {
		conditions = append(conditions, condToAny(c))
	}
	operations := make([]any, 0, len(r.Operations))
	for _, op := range r.Operations {
		operations = append(operations, opToAny(op))
	}
	return map[string]any{
		"id":         r.Id,
		"conditions": conditions,
		"operations": operations,
	}
}

// condToAny contributes only "operator", "type" and "value" when Step is
// empty; "step" is added only for pq_custom_data conditions that carry one.
func condToAny(c condDoc) map[string]any {
	out := map[string]any{
		"type":     c.Type,
		"operator": c.Operator,
		"value":    c.Value,
	}
	if c.Step != "" {
		out["step"] = c.Step
	}
	return out
}

// opToAny contributes only "type" when Params is nil.
func opToAny(op opDoc) map[string]any {
	out := map[string]any{
		"type": op.Type,
	}
	if op.Params != nil {
		out["params"] = op.Params
	}
	return out
}

// fanOut writes the same bytes to all eleven tenant directories, so
// byte-identity is structural rather than something checked afterwards. It
// does not create the tenant directories - all eleven already exist, and a
// missing one is a real error worth surfacing.
func fanOut(seedRoot, reactorId string, b []byte) error {
	for _, dir := range seedDirs {
		path := filepath.Join(seedRoot, dir, "reactor-actions/reactors", "reactor-"+reactorId+".json")
		if err := os.WriteFile(path, b, 0o644); err != nil {
			return fmt.Errorf("writing reactor %s to %s: %w", reactorId, path, err)
		}
	}
	return nil
}
