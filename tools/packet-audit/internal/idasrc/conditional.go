package idasrc

import "strings"

// HasRepeatingRun reports whether the primitive slice contains a loop/array
// pattern that the decompiler UNROLLED — i.e. the same block of reads appears
// three or more consecutive times.
//
// Formally: returns true if there exist a block length L >= 1 and a start index
// i such that ops[i:i+L] == ops[i+L:i+2L] == ops[i+2L:i+3L] AND L*3 >= 6
// (so a bare triple of a SINGLE read, e.g. [D1,D1,D1], does NOT qualify — only
// a run of six or more reads does; a [D4,D2,D2]×3 = 9 reads run qualifies).
//
// The L*3 >= 6 guard prevents false positives on trivially repeated singleton
// reads that may just be coincidental repetitions in a flat handler.
//
// Unresolved entries are included in the comparison (the block must match
// exactly, position-for-position, including any Unresolved entries).
func HasRepeatingRun(ops []Primitive) bool {
	n := len(ops)
	// Maximum block length L is at most n/3 (need three consecutive copies).
	for L := 1; L <= n/3; L++ {
		if L*3 < 6 {
			// Block too short; would be trivially coincidental (e.g. [D1]×3 = 3 reads).
			continue
		}
		// Slide the window: start index i from 0 to n-3L.
		for i := 0; i+3*L <= n; i++ {
			if slicesEqual(ops[i:i+L], ops[i+L:i+2*L]) &&
				slicesEqual(ops[i:i+L], ops[i+2*L:i+3*L]) {
				return true
			}
		}
	}
	return false
}

// slicesEqual reports whether two Primitive slices are element-wise identical.
func slicesEqual(a, b []Primitive) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// ReadsAreConditional reports whether the handler's packet reads BRANCH on an
// earlier value — i.e. at least one direction-relevant primitive read
// (CInPacket::Decode* clientbound / COutPacket::Encode* serverbound) occurs
// INSIDE a conditional or loop block (switch / if / else / for / while / case /
// do), nested below the function's top-level body.
//
// Why this matters (the load-bearing reason): the flat whole-function export
// read-order is the UNION of every branch. Atlas writes ONE branch. When the
// client handler branches on an early read (e.g. DropDestroy's
// `if ( leaveType == 2 ) { Decode4(...); }`), comparing the flat union against
// a single Atlas branch yields a FALSE width/length mismatch. Such a packet must
// be triaged as per-mode-branch (verify per-branch), NOT flagged as a real bug.
//
// Detection is a brace-depth + keyword line scanner (the same shape used by
// ParseDecompile, NOT a full C parser):
//   - Hex-Rays `/* line: N */` prefixes are stripped first (stripLinePrefix).
//   - We track the conditional/loop nesting depth: a block opens when a line is a
//     switch/if/else/for/while/do header or a `case`/`default:` label; it closes
//     by brace depth.
//   - A relevant read seen while that nesting depth > 0 → conditional (true).
//   - Reads only at nesting depth 0 (the function body top level) → false.
//
// Bias: when unsure, prefer CONDITIONAL — a false per-mode-branch classification
// only defers a packet for per-branch verification, whereas a false
// non-conditional classification would let a flat-union compare cry "real bug".
func ReadsAreConditional(decompileText string, dir Direction) bool {
	reDecode, _ := dirRegexes(dir)

	rawLines := strings.Split(decompileText, "\n")
	lines := make([]string, len(rawLines))
	for i, l := range rawLines {
		lines[i] = stripLinePrefix(l)
	}

	// condStack holds the brace depth at which each currently-open conditional /
	// loop block's BODY began. A read is conditional iff this stack is non-empty
	// when the read is seen. We also track case/label scopes, which open at a
	// switch body's brace depth (no extra brace of their own) and are bounded by
	// the enclosing switch body.
	var condDepths []int // brace depths at which open conditional/loop bodies began
	// caseActive tracks whether a `case`/`default` label scope is currently open
	// at a given switch-body brace depth. A label scope is itself a conditional
	// context (its body only runs for that discriminator value), so a read under
	// it counts even when no inner brace opened.
	caseDepths := map[int]bool{} // brace depth of switch body -> case label open

	braceDepth := 0
	pendingBlock := false // a conditional/loop header seen, awaiting its body `{`

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// A `case N:` / `default:` label opens a conditional label scope at the
		// CURRENT brace depth (the switch body). Reads after it (until the case
		// ends by brace) branch on the discriminator.
		if isCaseLabel(trimmed) {
			caseDepths[braceDepth] = true
		}

		// Detect a conditional/loop header. Its body brace usually opens on this
		// line or the next; stage it so it binds at the next depth increase.
		if isConditionalHeader(trimmed) {
			pendingBlock = true
		}

		opensBrace := strings.Contains(line, "{")
		if pendingBlock && opensBrace {
			condDepths = append(condDepths, braceDepth+1)
			pendingBlock = false
		}

		braceDepth += strings.Count(line, "{")

		// A relevant read: conditional iff any conditional/loop body is open OR a
		// case-label scope is active at any shallower switch-body depth.
		if reDecode.MatchString(line) {
			if len(condDepths) > 0 || anyCaseActive(caseDepths, braceDepth) {
				return true
			}
		}

		braceDepth -= strings.Count(line, "}")

		// Pop conditional/loop scopes whose body we have exited.
		for len(condDepths) > 0 && braceDepth < condDepths[len(condDepths)-1] {
			condDepths = condDepths[:len(condDepths)-1]
		}
		// Drop case-label scopes whose switch body we have exited.
		for d := range caseDepths {
			if braceDepth < d {
				delete(caseDepths, d)
			}
		}
	}
	return false
}

// isConditionalHeader reports whether a (trimmed) line begins a conditional or
// loop block: switch/if/else/for/while/do. `else if` is covered by the `if`
// and `else` prefixes; a bare `do` (loop) is included.
func isConditionalHeader(trimmed string) bool {
	if isLoopHeader(trimmed) { // for / while / do
		return true
	}
	switch {
	case strings.HasPrefix(trimmed, "switch ") || strings.HasPrefix(trimmed, "switch("):
		return true
	case strings.HasPrefix(trimmed, "if ") || strings.HasPrefix(trimmed, "if("):
		return true
	case trimmed == "else" || strings.HasPrefix(trimmed, "else ") || strings.HasPrefix(trimmed, "else{"):
		return true
	}
	return false
}

// isCaseLabel reports whether a (trimmed) line is a `case N:` or `default:`
// switch label.
func isCaseLabel(trimmed string) bool {
	return strings.HasPrefix(trimmed, "case ") || strings.HasPrefix(trimmed, "default:") || trimmed == "default :"
}

// anyCaseActive reports whether a case-label scope is open at some switch-body
// brace depth at or below the current depth (i.e. we are textually inside a
// case body that has not yet been closed by a brace).
func anyCaseActive(caseDepths map[int]bool, braceDepth int) bool {
	for d := range caseDepths {
		if braceDepth >= d {
			return true
		}
	}
	return false
}
