package evidence

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
)

// FunctionHash computes the canonical sha256 of one function's record in an
// IDA export JSON. Canonical form: unmarshal the function entry to
// map[string]any and re-marshal with encoding/json (sorted keys). Errors when
// the function is absent — the caller renders "citation unresolvable".
func FunctionHash(exportPath, fname string) (string, error) {
	raw, err := os.ReadFile(exportPath)
	if err != nil {
		return "", err
	}
	var file struct {
		Functions map[string]json.RawMessage `json:"functions"`
	}
	if err := json.Unmarshal(raw, &file); err != nil {
		return "", fmt.Errorf("%s: %w", exportPath, err)
	}
	entry, ok := file.Functions[fname]
	if !ok {
		return "", fmt.Errorf("%s: function %q not in export (citation unresolvable)", exportPath, fname)
	}
	var v any
	if err := json.Unmarshal(entry, &v); err != nil {
		return "", err
	}
	canon, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", sha256.Sum256(canon)), nil
}
