package drift

import "encoding/json"

// Canonicalize marshals v, drops the Excluded keys, and prunes every
// empty value, returning the comparable document.
//
// Pruning recursively removes any object key whose value is null, an
// empty array, or an empty object AFTER recursing into it. This collapses
// the whole nil-vs-empty false-positive class in one generic rule rather
// than one normalizer per slice field: `npcs: null` (a seed file that
// omits the key) is the same document as `npcs: []` (what the UI sends
// after a round trip through `?? []`), at any depth, without naming the
// field.
//
// Pruning cannot hide a real divergence: if one side has content and the
// other is empty, the non-empty side keeps its key and the hashes differ.
// It only erases distinctions between WAYS OF WRITING NOTHING. false, 0
// and "" are NOT pruned -- they are values, not absences.
func Canonicalize(v any) (Doc, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	var top map[string]json.RawMessage
	if err := json.Unmarshal(b, &top); err != nil {
		return nil, err
	}

	out := make(Doc, len(top))
	for k, raw := range top {
		if isExcluded(k) {
			continue
		}
		pruned, empty, err := prune(raw)
		if err != nil {
			return nil, err
		}
		if empty {
			continue
		}
		out[k] = pruned
	}
	return out, nil
}

func isExcluded(k string) bool {
	for _, e := range Excluded {
		if e == k {
			return true
		}
	}
	return false
}

// prune returns raw with empty descendants removed, and reports whether
// the result is itself empty (null, [], or {}).
func prune(raw json.RawMessage) (json.RawMessage, bool, error) {
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return nil, false, err
	}
	switch t := v.(type) {
	case nil:
		return nil, true, nil
	case map[string]any:
		obj := make(map[string]json.RawMessage, len(t))
		for k, child := range t {
			cb, err := json.Marshal(child)
			if err != nil {
				return nil, false, err
			}
			pruned, empty, err := prune(cb)
			if err != nil {
				return nil, false, err
			}
			if empty {
				continue
			}
			obj[k] = pruned
		}
		if len(obj) == 0 {
			return nil, true, nil
		}
		b, err := json.Marshal(obj)
		return b, false, err
	case []any:
		if len(t) == 0 {
			return nil, true, nil
		}
		arr := make([]json.RawMessage, 0, len(t))
		for _, child := range t {
			cb, err := json.Marshal(child)
			if err != nil {
				return nil, false, err
			}
			pruned, empty, err := prune(cb)
			if err != nil {
				return nil, false, err
			}
			if empty {
				// An element that prunes to nothing is still an
				// element: the array's LENGTH is content. Keep it
				// as the empty object it canonicalizes to.
				pruned = json.RawMessage("{}")
			}
			arr = append(arr, pruned)
		}
		b, err := json.Marshal(arr)
		return b, false, err
	default:
		// A scalar: false, 0 and "" are values, not absences.
		return raw, false, nil
	}
}
