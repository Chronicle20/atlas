package idasrc

import "sort"

// ModeBinding is one Atlas #Mode entry's case assignment within a base handler.
type ModeBinding struct {
	FName string
	Case  int64
}

// BijectionResult: client cases with no Atlas writer (Missing) and Atlas modes
// with no client case (Extra). Both sorted deterministically.
type BijectionResult struct {
	Missing []int64       // client case labels with no bound Atlas mode
	Extra   []ModeBinding // Atlas modes whose case is absent from the client
}

// Bijection diffs a client case-label set against the bound Atlas modes. A nil
// client case-set means we have no dispatch-structure information for this base
// (e.g. the decompile yielded no labels) — we cannot claim anything is missing OR
// extra, so the result is empty. Callers should treat a nil client as "no
// completeness signal", not "everything is wrong".
func Bijection(client *CaseSet, modes []ModeBinding) BijectionResult {
	var res BijectionResult
	if client == nil {
		return res
	}
	bound := map[int64]bool{}
	for _, m := range modes {
		bound[m.Case] = true
	}
	clientHas := map[int64]bool{}
	for _, c := range client.Values() {
		clientHas[c] = true
		if !bound[c] {
			res.Missing = append(res.Missing, c)
		}
	}
	for _, m := range modes {
		if !clientHas[m.Case] {
			res.Extra = append(res.Extra, m)
		}
	}
	sort.Slice(res.Missing, func(i, j int) bool { return res.Missing[i] < res.Missing[j] })
	sort.Slice(res.Extra, func(i, j int) bool { return res.Extra[i].FName < res.Extra[j].FName })
	return res
}
