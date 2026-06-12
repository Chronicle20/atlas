package idasrc

import (
	"sort"
	"strconv"
	"strings"
)

// ambiguityMargin is the bestScore-vs-secondScore gap below which two cases are
// considered indistinguishable; near-ties surface as ambiguous candidates.
const ambiguityMargin = 0.1

// InferDispatch finds the single-level dispatch selector whose extracted shape best
// matches a hand-authored read list, using audit-grade equivalence (width tolerance;
// an Unresolved live read is a wildcard). Returns the best selector path, a
// confidence in [0,1], and the close-runner-up case values when ambiguous.
// (Single discriminator level — nested dispatch is handled in a later live phase.)
func InferDispatch(base Fields, hand []FieldCall) (dispatch []Selector, confidence float64, candidates []int64) {
	disc, cases := enumerateCases(base)

	type scored struct {
		c     int64
		score float64
	}
	var results []scored
	for _, c := range cases {
		ext := ExtractShape(base, []Selector{{Discriminator: disc, Case: c}})
		results = append(results, scored{c: c, score: seqScore(hand, ext)})
	}
	if len(results) == 0 {
		return nil, 0, nil
	}

	// Sort by descending score (then ascending case for deterministic ties).
	sort.Slice(results, func(i, j int) bool {
		if results[i].score != results[j].score {
			return results[i].score > results[j].score
		}
		return results[i].c < results[j].c
	})

	bestCase := results[0].c
	bestScore := results[0].score
	secondScore := 0.0
	if len(results) > 1 {
		secondScore = results[1].score
	}

	// Confidence is high only when bestScore is high AND well-separated from the
	// runner-up. The separation factor scales linearly from 0 (a dead tie) up to
	// 1.0 once the gap reaches 1.0 (e.g. a unique match against zero-scoring
	// alternatives, or only one candidate). Multiplying by bestScore keeps a
	// well-separated-but-poor match from claiming high confidence.
	sep := bestScore - secondScore
	if sep < 0 {
		sep = 0
	}
	// Map a separation gap into [0,1]: a gap at/above the ambiguity margin already
	// signals a real winner, so it saturates quickly toward 1.0.
	sepFactor := sep / ambiguityMargin
	if sepFactor > 1 {
		sepFactor = 1
	}
	confidence = bestScore * sepFactor
	if confidence < 0 {
		confidence = 0
	}
	if confidence > 1 {
		confidence = 1
	}

	// Candidates: every case scoring within the ambiguity margin of the best.
	// Length >= 2 only when the result is genuinely ambiguous (a near-tie).
	for _, res := range results {
		if bestScore-res.score < ambiguityMargin {
			candidates = append(candidates, res.c)
		}
	}
	if len(candidates) < 2 {
		candidates = nil
	}

	return []Selector{{Discriminator: disc, Case: bestCase}}, confidence, candidates
}

// EntryShape pairs an entry's FName with its hand-authored reads for joint assignment.
type EntryShape struct {
	FName string
	Hand  []FieldCall
}

// Assignment is the inferred dispatch for one entry within a joint per-base solve.
type Assignment struct {
	FName      string
	Dispatch   []Selector
	Confidence float64
	Candidates []int64 // close alternatives when the assignment is ambiguous
}

// InferDispatchJoint assigns each entry to a DISTINCT case of the base, maximizing
// total alignment score (one-to-one). This resolves the ambiguity/conflicts that
// independent per-entry inference produces when multiple #-mode entries share a base.
// Entries beyond the available case count, or with no positive-score case, get an
// empty dispatch + low confidence. Deterministic.
//
// Independent per-entry inference (InferDispatch) lets two siblings claim the same
// case and cannot break a near-tie (the canonical OnFriendResult#Invite 8-vs-9). A
// base's N #-mode entries map one-to-one onto N distinct switch cases, so solving the
// assignment jointly under that constraint supplies the discriminating signal.
//
// Output is ordered to match the input entries slice.
func InferDispatchJoint(base Fields, entries []EntryShape) []Assignment {
	arms := enumerateArms(base)

	n := len(entries)
	out := make([]Assignment, n)
	for i := range entries {
		out[i] = Assignment{FName: entries[i].FName}
	}
	if n == 0 {
		return out
	}

	// Score matrix M[i][j] = alignment score of entries[i].Hand vs the shape of arms[j].
	m := len(arms)
	M := make([][]float64, n)
	caseShapes := make([][]FieldCall, m)
	for j := 0; j < m; j++ {
		caseShapes[j] = ExtractShape(base, []Selector{arms[j]})
	}
	for i := 0; i < n; i++ {
		M[i] = make([]float64, m)
		for j := 0; j < m; j++ {
			M[i][j] = seqScore(entries[i].Hand, caseShapes[j])
		}
	}

	// Greedy max-first one-to-one assignment: repeatedly pick the globally highest
	// unassigned (i,j) with score > 0; assign entry i -> case j; remove row i and
	// column j. Ties broken by (lower entry index, then lower case value) for
	// determinism. assignedCase[i] = column index j, or -1 if unassigned.
	rowUsed := make([]bool, n)
	colUsed := make([]bool, m)
	assignedCol := make([]int, n)
	for i := range assignedCol {
		assignedCol[i] = -1
	}
	for k := 0; k < n && k < m; k++ {
		bestI, bestJ := -1, -1
		bestScore := 0.0
		for i := 0; i < n; i++ {
			if rowUsed[i] {
				continue
			}
			for j := 0; j < m; j++ {
				if colUsed[j] {
					continue
				}
				s := M[i][j]
				if s <= 0 {
					continue
				}
				better := s > bestScore
				if !better && s == bestScore && bestI >= 0 {
					// Tie-break: lower entry index, then lower arm (column) index.
					if i < bestI || (i == bestI && j < bestJ) {
						better = true
					}
				}
				if better {
					bestScore, bestI, bestJ = s, i, j
				}
			}
		}
		if bestI < 0 {
			break // no remaining positive-score pair
		}
		rowUsed[bestI] = true
		colUsed[bestJ] = true
		assignedCol[bestI] = bestJ
	}

	// Build a reverse map: colAssignedTo[j] = the entry index that was assigned
	// to case column j, or -1 if the case is free (unassigned to any entry).
	colAssignedTo := make([]int, m)
	for j := range colAssignedTo {
		colAssignedTo[j] = -1
	}
	for i := 0; i < n; i++ {
		if assignedCol[i] >= 0 {
			colAssignedTo[assignedCol[i]] = i
		}
	}

	// Per assigned entry: build dispatch + joint-aware confidence.
	//
	// For each entry i assigned to case j, bestAvail is the highest M[i][k]
	// over all cases k != j, EXCLUDING only cases that were decisively claimed
	// by another entry (i.e., the winner's score on k exceeds i's score on k by
	// more than ambiguityMargin). A case merely tied-away from i is not excluded
	// — it was a genuine alternative, so it should still suppress confidence.
	//
	// This correctly handles two scenarios:
	//  - Distinctive joint pick: Invite scores 1.0 on case 9 and 0.5 on case 8.
	//    Update decisively claims case 8 (score 1.0 vs Invite's 0.5; gap=0.5 >
	//    ambiguityMargin). So case 8 is excluded from bestAvail for Invite →
	//    bestAvail=0 → sep=1.0 → HIGH confidence.
	//  - Ambiguous identical-shape: A and B both score 1.0 on cases 1 and 2. A
	//    wins case 1 by tie-break (score 1.0 vs B's 1.0; gap=0 ≤ ambiguityMargin).
	//    Case 1 is NOT excluded from bestAvail for B → bestAvail=1.0 → sep=0 →
	//    LOW confidence. Likewise, case 2 is a tie for A → bestAvail=1.0 → LOW.
	for i := 0; i < n; i++ {
		j := assignedCol[i]
		if j < 0 {
			continue // unassigned: empty dispatch, confidence 0
		}
		assignedScore := M[i][j]

		// bestAvail: max over alternative cases, excluding only those decisively
		// claimed by a competitor (competitor's score exceeds ours by > ambiguityMargin).
		bestAvailScore := 0.0
		for jj := 0; jj < m; jj++ {
			if jj == j {
				continue
			}
			winner := colAssignedTo[jj]
			if winner >= 0 && winner != i {
				// Case jj was assigned to another entry. Exclude it only if that
				// entry's score on jj decisively exceeds our score on jj.
				if M[winner][jj]-M[i][jj] > ambiguityMargin {
					continue
				}
			}
			if M[i][jj] > bestAvailScore {
				bestAvailScore = M[i][jj]
			}
		}

		sep := assignedScore - bestAvailScore
		if sep < 0 {
			sep = 0
		}
		sepFactor := sep / ambiguityMargin
		if sepFactor > 1 {
			sepFactor = 1
		}
		conf := assignedScore * sepFactor
		if conf < 0 {
			conf = 0
		}
		if conf > 1 {
			conf = 1
		}

		// Candidates: the equality-case values of arms (assigned or free) within the
		// ambiguity margin of the assigned score — informational near-tie signal.
		// Non-equality (verbatim) arms have no numeric case and are omitted here; a
		// near-tie count is still derived from how many arms fall within the margin.
		near := 0
		var cands []int64
		for jj := 0; jj < m; jj++ {
			if assignedScore-M[i][jj] < ambiguityMargin {
				near++
				if arms[jj].Guard == "" && !arms[jj].Default {
					cands = append(cands, arms[jj].Case)
				}
			}
		}
		sort.Slice(cands, func(a, b int) bool { return cands[a] < cands[b] })
		if near < 2 {
			cands = nil
		}

		out[i].Dispatch = []Selector{arms[j]}
		out[i].Confidence = conf
		out[i].Candidates = cands
	}

	return out
}

// enumerateArms collects the distinct dispatch arms present across the guards of
// base.Calls as Selectors, in first-seen order: an equality clause "disc == N"
// yields {Discriminator: disc, Case: N}; the default token yields {Default: true};
// any other single non-loop clause yields a verbatim {Guard: clause}. Loop clauses
// ("loop ...") are not dispatch arms and are skipped. This is the superset of
// enumerateCases used by the inference (which now proposes verbatim selectors for
// non-equality dispatch as well as numeric cases).
func enumerateArms(base Fields) []Selector {
	seen := map[string]bool{}
	var arms []Selector
	for _, call := range base.Calls {
		g := strings.TrimSpace(call.Guard)
		if g == "" {
			continue
		}
		if g == DefaultGuardToken {
			if !seen["default"] {
				seen["default"] = true
				arms = append(arms, Selector{Default: true})
			}
			continue
		}
		for _, clause := range strings.Split(g, "&&") {
			clause = strings.TrimSpace(clause)
			clause = strings.TrimPrefix(clause, "(")
			clause = strings.TrimSuffix(clause, ")")
			clause = strings.TrimSpace(clause)
			if clause == "" || strings.HasPrefix(clause, "loop ") {
				continue
			}
			if parts := strings.SplitN(clause, "==", 2); len(parts) == 2 {
				if v, ok := parseIntLit(strings.TrimSpace(parts[1])); ok {
					disc := strings.TrimSpace(parts[0])
					key := "eq:" + disc + ":" + strconv.FormatInt(v, 10)
					if !seen[key] {
						seen[key] = true
						arms = append(arms, Selector{Discriminator: disc, Case: v})
					}
					continue
				}
			}
			key := "g:" + clause
			if !seen[key] {
				seen[key] = true
				arms = append(arms, Selector{Guard: clause})
			}
		}
	}
	return arms
}

// enumerateCases collects the distinct integer case values present across the
// guards of base.Calls, and picks the discriminator name to use for the returned
// selector (the most common left-token of an "X == V" clause; default "switch").
func enumerateCases(base Fields) (disc string, cases []int64) {
	seenCase := map[int64]bool{}
	var order []int64
	leftCounts := map[string]int{}

	for _, call := range base.Calls {
		for _, clause := range strings.Split(call.Guard, "&&") {
			clause = strings.TrimSpace(clause)
			clause = strings.TrimPrefix(clause, "(")
			clause = strings.TrimSuffix(clause, ")")
			clause = strings.TrimSpace(clause)
			parts := strings.SplitN(clause, "==", 2)
			if len(parts) != 2 {
				continue
			}
			left := strings.TrimSpace(parts[0])
			val, ok := parseIntLit(strings.TrimSpace(parts[1]))
			if !ok {
				continue
			}
			leftCounts[left]++
			if !seenCase[val] {
				seenCase[val] = true
				order = append(order, val)
			}
		}
	}

	disc = "switch"
	best := 0
	// Deterministic: highest count wins; ties broken by lexical token order.
	var lefts []string
	for tok := range leftCounts {
		lefts = append(lefts, tok)
	}
	sort.Strings(lefts)
	for _, tok := range lefts {
		if leftCounts[tok] > best {
			best = leftCounts[tok]
			disc = tok
		}
	}

	return disc, order
}

// seqScore aligns the hand-authored read list against an extracted candidate
// shape with audit-grade width tolerance AND Unresolved-run absorption, then
// normalizes the matched-hand count.
//
// An Unresolved live read represents an undecompilable helper (e.g. the GW_Friend
// 39-byte struct, which expands by hand into several concrete reads). A strict
// position-by-position compare would credit such an Unresolved with matching only
// a single hand field, depressing the score of the correct case below a simpler
// competing case. Instead, an Unresolved live read absorbs a RUN of hand reads:
//   - If a concrete live read follows, the hand cursor advances (each absorbed
//     read counts as matched) until the hand read equivalences that next concrete
//     read (the re-anchor) or the hand list ends. Absorbing zero reads is allowed.
//   - If the Unresolved is the LAST live read, it absorbs ALL remaining hand reads.
//
// Score = matchedHandPositions / max(len(hand), len(concreteLiveReads)). A full
// absorb-and-anchor alignment yields ~1.0; partial alignments score lower, and
// concrete live reads that go unmatched (overlong shape, or hand exhausted) are
// penalized via the max() denominator.
func seqScore(hand, ext []FieldCall) float64 {
	concrete := 0
	for _, e := range ext {
		if e.Op != Unresolved {
			concrete++
		}
	}
	denom := len(hand)
	if concrete > denom {
		denom = concrete
	}
	if denom == 0 {
		return 1
	}
	return float64(alignMatches(hand, ext)) / float64(denom)
}

// alignMatches returns the maximum number of hand positions that a candidate live
// shape can account for, via a Needleman–Wunsch-style dynamic program. The
// Unresolved-as-absorbing-gap rule (see seqScore) is encoded as a transition that
// consumes a hand read for free (counting it as matched) without advancing the
// live cursor — so a single Unresolved can swallow a whole run of hand reads and a
// following concrete live read re-anchors on the next equivalent hand read. A DP
// (rather than a greedy walk) is used so the absorb length is chosen to maximize
// the overall alignment, avoiding mis-anchoring when an absorbed read happens to
// also equivalence the re-anchor target.
func alignMatches(hand, ext []FieldCall) int {
	h, e := len(hand), len(ext)
	// memo[hi][li] = best matched-hand count for hand[hi:] vs ext[li:]; -1 = unset.
	memo := make([][]int, h+1)
	for i := range memo {
		memo[i] = make([]int, e+1)
		for j := range memo[i] {
			memo[i][j] = -1
		}
	}
	var best func(hi, li int) int
	best = func(hi, li int) int {
		if li == e {
			// No live reads left: remaining hand reads are unmatched (no credit).
			return 0
		}
		if memo[hi][li] >= 0 {
			return memo[hi][li]
		}
		res := 0
		if ext[li].Op == Unresolved {
			// Absorb zero hand reads and advance the live cursor...
			res = best(hi, li+1)
			// ...or absorb one more hand read (counts as matched) and stay on the
			// same Unresolved so it can keep swallowing the run.
			if hi < h {
				if v := 1 + best(hi+1, li); v > res {
					res = v
				}
			}
		} else {
			// Concrete live read. Either skip it unmatched (the shape is longer
			// than the hand list, or this read has no hand counterpart here)...
			res = best(hi, li+1)
			// ...or match it against the current hand read in place. Only an
			// Unresolved may skip hand reads for free; a concrete read must align
			// positionally, so unrelated shapes cannot match out of order.
			if hi < h && fieldEquivalent(hand[hi].Op, ext[li].Op) {
				if v := 1 + best(hi+1, li+1); v > res {
					res = v
				}
			}
		}
		memo[hi][li] = res
		return res
	}
	return best(0, 0)
}

// FieldEquivalent reports whether two primitives are equivalent under audit-grade
// width tolerance:
//   - Unresolved on either side is a wildcard (the undecompilable bulk reads).
//   - equal byte width matches.
//   - a fixed width (>0) vs an opaque DecodeBuf matches (no declared length).
//
// Exported so callers outside the package (e.g. cmd/decompose) can perform
// position-by-position prefix comparisons without duplicating the tolerance logic.
func FieldEquivalent(a, b Primitive) bool {
	if a == Unresolved || b == Unresolved {
		return true
	}
	aw := idaW(a)
	bw := idaW(b)
	if aw == bw {
		return true
	}
	if aw > 0 && b == DecodeBuf {
		return true
	}
	if a == DecodeBuf && bw > 0 {
		return true
	}
	return false
}

// fieldEquivalent is the package-internal alias used by ValidateShape and the
// alignment DP (alignMatches). Keeps existing call sites unchanged.
func fieldEquivalent(a, b Primitive) bool { return FieldEquivalent(a, b) }

// idaW maps an idasrc primitive to a byte width: fixed widths positive, string
// and buffer to distinct negative sentinels (so they never width-match each
// other by accident).
func idaW(p Primitive) int {
	switch p {
	case Decode1:
		return 1
	case Decode2:
		return 2
	case Decode4:
		return 4
	case Decode8:
		return 8
	case DecodeStr:
		return -1
	case DecodeBuf:
		return -2
	}
	return 0
}
