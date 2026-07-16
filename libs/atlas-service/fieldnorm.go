package service

import (
	"sort"
	"strings"

	"github.com/sirupsen/logrus"
)

// fieldKeyNormalizerHook rewrites camelCase log field keys to snake_case at
// emit time so the fleet's ~1,500 legacy WithField call sites converge on
// one spelling without a rename sweep (CP-9).
//
// Safe to mutate entry.Data in place: logrus v1.9.4 duplicates the entry
// (including the Data map) before firing hooks (entry.Dup(), entry.go:227),
// so callers retaining a derived *Entry never observe the rewrite.
//
// Ordering caveat: keys added by hooks registered AFTER this one escape
// normalization. CreateLogger registers it last.
type fieldKeyNormalizerHook struct{}

func (fieldKeyNormalizerHook) Levels() []logrus.Level { return logrus.AllLevels }

func (fieldKeyNormalizerHook) Fire(entry *logrus.Entry) error {
	var renames [][2]string // nil for the common fully-normalized entry: zero allocation
	for k := range entry.Data {
		if nk, changed := normalizeFieldKey(k); changed {
			renames = append(renames, [2]string{k, nk})
		}
	}
	if renames == nil {
		return nil
	}
	// Sort so collision resolution is deterministic regardless of map order.
	sort.Slice(renames, func(i, j int) bool { return renames[i][0] < renames[j][0] })
	for _, r := range renames {
		v := entry.Data[r[0]]
		delete(entry.Data, r[0])
		// Collision rule: an explicitly snake_case key wins; the camelCase
		// duplicate is dropped (documented in docs/observability.md).
		if _, exists := entry.Data[r[1]]; !exists {
			entry.Data[r[1]] = v
		}
	}
	return nil
}

// normalizeFieldKey converts a camelCase ASCII key to snake_case. Keys
// containing a dot (ECS/namespaced, e.g. service.name) and keys with no
// uppercase letters pass through unchanged (changed=false, no allocation).
func normalizeFieldKey(k string) (string, bool) {
	if strings.ContainsRune(k, '.') {
		return k, false
	}
	hasUpper := false
	for i := 0; i < len(k); i++ {
		if k[i] >= 'A' && k[i] <= 'Z' {
			hasUpper = true
			break
		}
	}
	if !hasUpper {
		return k, false
	}
	isLowerOrDigit := func(c byte) bool { return (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') }
	isUpper := func(c byte) bool { return c >= 'A' && c <= 'Z' }
	var b strings.Builder
	b.Grow(len(k) + 4)
	for i := 0; i < len(k); i++ {
		c := k[i]
		if isUpper(c) {
			if i > 0 && isLowerOrDigit(k[i-1]) {
				// lower/digit → upper boundary: characterId → character_id
				b.WriteByte('_')
			} else if i > 0 && isUpper(k[i-1]) && i+1 < len(k) && k[i+1] >= 'a' && k[i+1] <= 'z' {
				// last upper of an upper-run followed by lower: HTTPServer → http_server
				b.WriteByte('_')
			}
			b.WriteByte(c + ('a' - 'A'))
		} else {
			b.WriteByte(c)
		}
	}
	nk := b.String()
	return nk, nk != k
}
