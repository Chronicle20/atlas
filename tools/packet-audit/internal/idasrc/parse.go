package idasrc

import (
	"regexp"
	"strings"
)

var (
	reComment = regexp.MustCompile(`//\s*(.+?)\s*$`)

	// reDecodeCB / reDecodeSB are the direction-scoped primitive matchers.
	// A CLIENTBOUND packet is one the client READS, so its reference is the
	// client's CInPacket::Decode* order; a SERVERBOUND packet is one the client
	// WRITES, so its reference is the client's COutPacket::Encode* order. Only
	// the relevant class's calls are captured — the other direction's calls
	// (e.g. response-building COutPacket writes inside a clientbound read
	// handler) are ignored entirely. parsePrim/opName treat Decode and Encode
	// identically (the audit primitive is direction-agnostic), so both yield the
	// canonical Decode{1,2,4,8,Str,Buf} op strings.
	reDecodeCB = regexp.MustCompile(`CInPacket::(Decode)(1|2|4|8|Str|Buffer)\s*\(`)
	reDecodeSB = regexp.MustCompile(`COutPacket::(Encode)(1|2|4|8|Str|Buffer)\s*\(`)

	// rePktVarCB / rePktVarSB capture the receiver/first-arg of the relevant
	// class's primitive call — the packet variable name (e.g. "a2"). EVERY match
	// across the function body seeds the packet-pointer alias set used for
	// helper-descent detection. Direction-scoped so a clientbound handler seeds
	// ONLY from CInPacket calls (its CInPacket arg), never from the COutPacket it
	// builds for the response (and vice versa for serverbound).
	rePktVarCB = regexp.MustCompile(`CInPacket::Decode\w+\s*\(\s*&?([A-Za-z_]\w*)`)
	rePktVarSB = regexp.MustCompile(`COutPacket::Encode\w+\s*\(\s*&?([A-Za-z_]\w*)`)
	// reCall captures a fully-qualified Class::method(...) call and its arg list.
	reCall = regexp.MustCompile(`([A-Za-z_]\w*(?:::[A-Za-z_]\w*)+)\s*\(([^;]*)\)`)
	// reNameCall captures a NAMED call — a fully-qualified Class::method, a
	// sub_XXXX helper, or a bare identifier — and its arg list. Used to descend
	// helpers that take a packet alias as a standalone arg. The leading `(?:^|[^.\w])`
	// avoids matching a member-access suffix as a bare call. Each `::`-qualified
	// segment may carry a `<...>` template-argument list (e.g.
	// `ZXString<char>::_Release`, `ZArray<int>::InsertBefore`) so that templated
	// callees are still recognized by the denylist instead of falling through to
	// the indirect-Unresolved bucket. Template-arg lists are assumed not to
	// themselves contain `>` (true for the single-level instantiations seen in
	// real decompile).
	reNameCall = regexp.MustCompile(`(?:^|[^.>\w:])((?:[A-Za-z_]\w*(?:<[^>]*>)?::)*[A-Za-z_]\w*(?:<[^>]*>)?)\s*\(([^;]*)\)`)
	// reTemplateArg strips `<...>` template-argument segments from a captured
	// callee name so it normalizes to its bare qualified form for denylist
	// matching (`ZXString<char>::_Release` -> `ZXString::_Release`).
	reTemplateArg = regexp.MustCompile(`<[^>]*>`)
	// reForCount captures the upper-bound var of a `< count` loop condition.
	reForCount = regexp.MustCompile(`<\s*([A-Za-z_]\w*)`)
	// reSwitch matches a `switch ( ... )` header and captures the full
	// discriminator expression (which may be a bare var or an inline read). The
	// trailing `\s*{?` tolerates a same-line opening brace (`switch ( x ) {`).
	reSwitch = regexp.MustCompile(`^\s*switch\s*\(\s*(.+?)\s*\)\s*{?\s*$`)
	// reBareVar matches a discriminator that is a single bare identifier.
	reBareVar = regexp.MustCompile(`^[A-Za-z_]\w*$`)
	// reCase captures the constant label of a `case N:` (decimal, hex, or
	// identifier). The trailing C integer suffix [uUlL]+ emitted by the upgraded
	// IDA decompiler (e.g. `case 7u:`, `case 0xAu:`, `case 9ul:`) is matched
	// outside the capture group so the emitted guard is always suffix-free
	// (`switch == 9`, not `switch == 9u`). Alternatives are tried longest-first
	// so `0x12u` captures `0x12` and not just `0`.
	reCase = regexp.MustCompile(`^\s*case\s+(0[xX][0-9A-Fa-f]+|[0-9]+|[A-Za-z_][A-Za-z0-9_]*)[uUlL]*\s*:`)
	// reIfEq matches an if / else-if header that dispatches on an equality against
	// a constant: `if ( x == 5 )`, `else if ( v7 == 0xA )`. Captures the optional
	// leading `else` (chain continuation), the discriminator, and the constant.
	// The trailing C integer suffix [uUlL]* is matched outside the capture so the
	// emitted guard is suffix-free, mirroring reCase. A condition that is not a
	// bare `ident == const` (cast, range, compound) does not match — by design,
	// such an arm gets no guard (honest unverifiable, never a fabricated `==`).
	reIfEq = regexp.MustCompile(`^\s*(?:(else)\s+)?if\s*\(\s*([A-Za-z_]\w*)\s*==\s*(0[xX][0-9A-Fa-f]+|[0-9]+)[uUlL]*\s*\)\s*{?\s*$`)
	// reElse matches a bare `else` / `else {` (no condition) — the default arm of
	// an if/else dispatch chain.
	reElse = regexp.MustCompile(`^\s*else\s*{?\s*$`)
	// reElseLed matches any `else`-led header (bare `else`, `else if (...)`, with
	// ANY condition). Used for multi-way-dispatch detection: an else-led header
	// means the chain reached a 2nd arm, regardless of whether that arm's condition
	// is a single predicate the verbatim emitter can represent. The `\b` avoids
	// matching identifiers like `else_var`.
	reElseLed = regexp.MustCompile(`^\s*else\b`)
	// reIfCond matches an if / else-if header with ANY parenthesized condition,
	// capturing the optional leading "else" and the full condition text. Used as a
	// FALLBACK to reIfEq: a single-predicate non-equality condition (e.g. "v5 < 5",
	// "v5 & 0x10") is emitted as a verbatim arm guard. Compound/indirect conditions
	// are rejected by isSinglePredicate so they bail to no-guard.
	reIfCond = regexp.MustCompile(`^\s*(?:(else)\s+)?if\s*\(\s*(.+?)\s*\)\s*{?\s*$`)
	// reAssign matches a simple `X = Y;` or `X = &Y;` assignment used for alias
	// closure (Y must be a bare identifier for the alias to propagate).
	reAssign = regexp.MustCompile(`^\s*([A-Za-z_]\w*)\s*=\s*&?([A-Za-z_]\w*)\s*;`)
	// reAssignAny matches ANY simple assignment to a bare-identifier lvalue,
	// `X = <rhs>` (rhs unconstrained), used ONLY for kill-on-reassign when the
	// RHS is not a bare identifier (those go through reAssign). The negative
	// lookahead-free `[^=]` after `=` rejects `==` comparisons; a bare leading
	// identifier (no `.`, `->`, `(`, `*`, `[`) avoids matching member-access or
	// macro lvalues (`LOBYTE(v43) =`, `*(p) =`, `a.b =`).
	reAssignAny = regexp.MustCompile(`^\s*([A-Za-z_]\w*)\s*=[^=]`)
	// reLinePrefix matches a leading Hex-Rays line annotation `/* ... */` so it
	// can be stripped before line-start-oriented detection.
	reLinePrefix = regexp.MustCompile(`^\s*/\*[^*]*\*/\s?`)
)

// denylist: helpers that take a packet pointer but never read the wire (UI/dialog/alloc).
var helperDenylist = []string{
	"CUIFadeYesNo::", "CUIDlg", "StringPool::", "operator new", "CWnd::",
	"ZAllocEx", "ZArray", "free", "malloc",
	// Obvious non-packet noise observed in real decompile (descent-noise only;
	// descending them would be harmless but yields no reads — listing them keeps
	// the export clean).
	"ZXString", "CUtilDlg::", "CWvsContext::SetNewFadeWnd",
	// COutPacket lifecycle / transport on the SERVERBOUND path: the COutPacket
	// is the relevant packet, so its alias flows into the constructor and the
	// socket-send call — but neither reads/writes a field (the constructor takes
	// the opcode; SendPacket flushes the finished buffer). They are not Encode
	// primitives and must not be mistaken for field-reading struct helpers.
	"COutPacket::COutPacket", "SendPacket",
}

// normalizeName strips `<...>` template-argument segments from a (possibly
// qualified) callee name so denylist substring matching applies to templated
// forms (`ZXString<char>::_Release` -> `ZXString::_Release`).
func normalizeName(name string) string {
	return reTemplateArg.ReplaceAllString(name, "")
}

func isDenylisted(name string) bool {
	name = normalizeName(name)
	for _, d := range helperDenylist {
		if strings.Contains(name, d) {
			return true
		}
	}
	return false
}

// stripLinePrefix removes a leading Hex-Rays `/* line: N, address: .. */`
// annotation (or any leading `/* ... */`) so that line-start-oriented detection
// (switch/case/assignment/brace/for) operates on the actual code. The mid-line
// Decode/Encode regexes are unaffected; trailing `// reg` comments are left in
// place (captured best-effort as labels).
func stripLinePrefix(line string) string {
	return reLinePrefix.ReplaceAllString(line, "")
}

// seedPacketParams returns the set of packet-pointer identifiers to seed the
// DYNAMIC alias set with: every identifier that is ever the first-arg of a
// CInPacket::Decode* / COutPacket::Encode* call somewhere in the body. These
// are the function's packet parameter(s). Seeding them up front lets an
// alias-before-first-read assignment (`v2 = Index; ... Decode1(Index);`,
// common in real decompile where a scratch copy is taken before the switch
// discriminator is decoded) propagate correctly, while kill-on-reassign (woven
// into the main walk) still dynamically REMOVES a packet identifier the moment
// Hex-Rays recycles it as scratch (`Index = 0;`), and a later Decode re-adds it.
func seedPacketParams(lines []string, rePktVar *regexp.Regexp) map[string]bool {
	set := map[string]bool{}
	for _, line := range lines {
		for _, m := range rePktVar.FindAllStringSubmatch(line, -1) {
			set[m[1]] = true
		}
	}
	return set
}

// dirRegexes selects the direction-scoped primitive matcher (reDecode-style)
// and packet-var seeder (rePktVar-style). Clientbound captures the client's
// CInPacket::Decode* reads; serverbound captures its COutPacket::Encode*
// writes. The other class's calls are not matched in either case, so a
// clientbound read handler ignores any response-building COutPacket writes (and
// a serverbound Send ignores any incoming CInPacket reads).
func dirRegexes(dir Direction) (prim, pktVar *regexp.Regexp) {
	if dir == DirServerbound {
		return reDecodeSB, rePktVarSB
	}
	return reDecodeCB, rePktVarCB
}

// updateLiveAliases applies one line's alias bookkeeping to the mutable `live`
// set, in source order, BEFORE the line's helper calls are classified:
//
//   - A CInPacket::Decode* / COutPacket::Encode* call ADDS its first-arg
//     identifier (it is the packet here) — re-adding a var previously killed.
//   - A simple `X = &?Y;` assignment with bare-identifier RHS: if Y (stripped of
//     a leading `&`) is currently live, ADD X (forward alias); otherwise REMOVE
//     X (X is reassigned away from the packet, e.g. an alias copy of a now-dead
//     var).
//   - Any OTHER simple assignment to a bare identifier `X = <non-identifier>;`
//     (literal, call result, expression — e.g. `Index = 0;`,
//     `Index = CFriend::FindIndex(...);`) REMOVES X from live: the var no longer
//     holds the packet pointer. This is the kill-on-reassign that prevents a
//     recycled scratch var from staying packet-tainted forever.
func updateLiveAliases(line string, live map[string]bool, rePktVar *regexp.Regexp) {
	// Decode/Encode adds (process before assignment so a discriminator like
	// `switch ( CInPacket::Decode1(Index) )` re-seeds Index).
	for _, m := range rePktVar.FindAllStringSubmatch(line, -1) {
		live[m[1]] = true
	}
	// Alias propagation / kill via a bare-identifier RHS.
	if m := reAssign.FindStringSubmatch(line); m != nil {
		dst, src := m[1], m[2]
		if live[src] {
			live[dst] = true
		} else {
			delete(live, dst)
		}
		return
	}
	// Any other simple `X = ...;` assignment to a bare identifier kills X.
	if m := reAssignAny.FindStringSubmatch(line); m != nil {
		delete(live, m[1])
	}
}

// aliasArgMatch reports whether any alias in the set appears as a standalone,
// whole-identifier argument in the comma-separated arg list.
func aliasArgMatch(args string, aliases map[string]bool) bool {
	for alias := range aliases {
		if argHasToken(args, alias) {
			return true
		}
	}
	return false
}

// isIndirectDispatch reports whether the line is a leading-`*` indirect /
// function-pointer / vtable call of the form `(*...)(args)`. Such a call has no
// resolvable callee NAME, so it must be classified as Unresolved (never a
// Delegate) — and its cast/type tokens (`void (__thiscall **)(...)`) must not be
// mistaken for a named call.
func isIndirectDispatch(line string) bool {
	t := strings.TrimSpace(line)
	// Skip an optional `lvalue = ` assignment prefix.
	if i := strings.Index(t, "= "); i >= 0 {
		rhs := strings.TrimSpace(t[i+1:])
		if strings.HasPrefix(rhs, "(*") {
			return true
		}
	}
	return strings.HasPrefix(t, "(*")
}

// passesAnyAliasIndirect reports whether any packet alias flows into an indirect
// CALL-arg group on the line (per passesPktIndirect, evaluated per-alias).
func passesAnyAliasIndirect(line string, aliases map[string]bool) bool {
	for alias := range aliases {
		if passesPktIndirect(line, alias) {
			return true
		}
	}
	return false
}

// opName maps the raw "<width>" capture to the canonical export op string.
func opName(width string) string {
	switch width {
	case "1":
		return "Decode1"
	case "2":
		return "Decode2"
	case "4":
		return "Decode4"
	case "8":
		return "Decode8"
	case "Str":
		return "DecodeStr"
	case "Buffer":
		return "DecodeBuf"
	}
	return ""
}

// lineComment extracts the trailing `// ...` comment from a line, if any.
func lineComment(line string) string {
	if c := reComment.FindStringSubmatch(line); c != nil {
		return c[1]
	}
	return ""
}

// argHasToken reports whether tok appears as a standalone, whole-identifier
// argument in a comma-separated arg list (ignoring surrounding address-of/deref
// punctuation). It deliberately does NOT match tok as a substring of a longer
// identifier (e.g. "a2" must not match "a23" or "ba2").
func argHasToken(args, tok string) bool {
	for _, a := range strings.Split(args, ",") {
		a = strings.TrimSpace(a)
		// Strip leading address-of / dereference operators so "&a2"/"*a2" match.
		a = strings.TrimLeft(a, "&*")
		a = strings.TrimSpace(a)
		if a == tok {
			return true
		}
	}
	return false
}

// controlKeywords are tokens that may precede a `(` group without making that
// group a call-arg list. A parenthesized group following one of these is a
// control-flow / operator construct (e.g. `if ( a2 )`, `return (a2)`), NOT an
// indirect call — so the packet var inside must not be read as Unresolved.
var controlKeywords = map[string]bool{
	"if": true, "while": true, "for": true, "switch": true,
	"return": true, "catch": true, "sizeof": true,
}

// precededByCall reports whether the `(` at line[openIdx] opens a genuine
// call-arg list. A group is a call only when the token immediately preceding
// the `(` (skipping whitespace) is either `)` — an indirect call result like
// `(*...)(args)` — or an identifier that is NOT a control keyword. A leading
// `(` with no preceding token, or one preceded by a control keyword / operator,
// is a parenthesized expression, not a call.
func precededByCall(line string, openIdx int) bool {
	i := openIdx - 1
	for i >= 0 && (line[i] == ' ' || line[i] == '\t') {
		i--
	}
	if i < 0 {
		return false
	}
	if line[i] == ')' {
		// Result of a prior parenthesized expression being called: (...)(args).
		return true
	}
	// Walk back over an identifier (incl. namespace `::` is broken by ':' which
	// is not an ident char, but the trailing segment is enough to classify).
	end := i + 1
	for i >= 0 && isIdentChar(line[i]) {
		i--
	}
	ident := line[i+1 : end]
	if ident == "" {
		// Preceded by some operator/punctuation (e.g. `+ (`, `* (`): expression.
		return false
	}
	return !controlKeywords[ident]
}

// isIdentChar reports whether b is a C identifier character.
func isIdentChar(b byte) bool {
	return b == '_' ||
		(b >= 'a' && b <= 'z') ||
		(b >= 'A' && b <= 'Z') ||
		(b >= '0' && b <= '9')
}

// passesPktIndirect reports whether pktVar appears as a standalone argument in
// some parenthesized CALL-arg group on the line. It is only consulted for lines
// that (a) are NOT Decode/Encode reads and (b) carry NO Class::method named call
// — so a true result means the packet var flows into an indirect /
// function-pointer / vtable dispatch (e.g. `(*(...)(*this + 4 * id))(this, a2)`).
// Each top-level `(...)` group is examined, but only groups that are actually
// call-arg lists (per precededByCall) count — so a control-flow condition like
// `if ( a2 )` / `while ( a2 )` is correctly rejected rather than flooding the
// export with false Unresolved gaps. This reuses argHasToken so the same
// whole-identifier matching applies (a2 must not match a23).
func passesPktIndirect(line, pktVar string) bool {
	depth := 0
	start := -1
	openIdx := -1
	for i := 0; i < len(line); i++ {
		switch line[i] {
		case '(':
			if depth == 0 {
				start = i + 1
				openIdx = i
			}
			depth++
		case ')':
			depth--
			if depth == 0 && start >= 0 {
				if precededByCall(line, openIdx) && argHasToken(line[start:i], pktVar) {
					return true
				}
				start = -1
			}
		}
	}
	return false
}

// ParseDecompile extracts the ordered packet read/write primitives from one
// function's Hex-Rays decompile text. Pure (no MCP access); emits rawCall
// entries.
//
// Beyond the linear reads of task 1.2, this scanner performs:
//
//   - Struct-helper descent: a fully-qualified Class::method(...) call that is
//     passed the packet variable (and is neither a Decode/Encode primitive nor
//     a denylisted UI/alloc helper) is emitted as a Delegate referencing that
//     method. The resolver later splices the referenced FName's reads in place.
//
//   - Loop guards: reads inside a `for (... < count ...)` body carry a
//     "loop <count>" Guard. This is the loop-vs-fixed-struct disambiguation —
//     a helper that passes the packet var is a fixed struct (Delegate), whereas
//     a genuine inline loop body reads directly under a loop guard.
//
//   - Switch sub-case guards: a discriminator `switch ( <var> )` followed by
//     `case N:` labels tags reads inside each case body with "<var> == N". The
//     case guard resets at `break;`, `default:`, the next `case`, or the
//     closing brace of the switch.
//
// Guards compose: loop and case scopes both contribute a fragment, and the
// emitted Guard is the AND of all currently-active fragments in open (outer →
// inner) order, e.g. "mode == 1 && loop count".
//
// It is a brace-depth + keyword line scanner, NOT a full C parser. Label/comment
// capture is best-effort; only op width, Delegate refs, guards, and order are
// load-bearing.
func ParseDecompile(text string, dir Direction) ([]rawCall, error) {
	var out []rawCall

	// Direction-scoped matchers: clientbound reads CInPacket::Decode*,
	// serverbound reads COutPacket::Encode*. The unselected class's calls are
	// not matched at all, so response-building writes in a clientbound handler
	// (and incoming reads in a serverbound Send) never enter the read-order,
	// the alias seed set, or the Delegate/Unresolved classification.
	reDecode, rePktVar := dirRegexes(dir)

	rawLines := strings.Split(text, "\n")

	// Strip Hex-Rays `/* line: N */` prefixes up front so all subsequent
	// line-start-oriented detection (switch/case/assignment/brace/for) is robust.
	lines := make([]string, len(rawLines))
	for i, l := range rawLines {
		lines[i] = stripLinePrefix(l)
	}

	// DYNAMIC packet-pointer alias tracking. `live` starts seeded with the
	// function's packet parameter(s) (every ever-decoded first-arg identifier) so
	// an alias-before-first-read copy propagates; it is then mutated IN ORDER as
	// the walk visits each line (updateLiveAliases): a Decode/Encode re-adds its
	// packet arg, an alias assignment propagates, and any reassignment to a bare
	// identifier away from the packet KILLS that var — so a recycled scratch var
	// (`Index = 0;`) stops producing phantom Delegates/Unresolved downstream.
	live := seedPacketParams(lines, rePktVar)

	// scope is one entry on the active guard stack: a guard fragment plus the
	// brace depth at which it opened (so it can be popped when we drop below).
	type scope struct {
		depth int    // brace depth at which this scope's body opened
		frag  string // guard fragment, e.g. "loop count" or "mode == 1"
	}

	braceDepth := 0        // running { } nesting depth
	pendingLoopVar := ""   // a counted-loop header seen, awaiting its body `{`
	pendingSwitchVar := "" // a switch header seen, awaiting its body `{`

	var stack []scope // active scope-guards, outer → inner

	// sw tracks the discriminator var per switch nesting level. A switch body
	// opens at braceDepth = openDepth+1; the active case fragment for that
	// switch lives in caseIdx (an index into stack, or -1 if no case active).
	type swEntry struct {
		bodyDepth int    // brace depth of the switch body
		discrim   string // discriminator variable name
		caseIdx   int    // index into stack of the active case scope, or -1
	}
	var switches []swEntry

	// ifChain tracks one active if / else-if / else dispatch chain. discrim is the
	// shared equality variable; startDepth is the brace depth at which the chain's
	// `if` header appeared (the chain is popped only when we drop BELOW it — NOT at
	// the first arm's `}`, since if/else arms are sibling blocks at startDepth+1,
	// unlike a switch's single body brace). armIdx indexes the current arm's scope
	// on `stack`, or -1 between arms / once the chain is closed.
	type ifChainEntry struct {
		startDepth int
		discrim    string
		armIdx     int
	}
	var ifChains []ifChainEntry
	pendingArmFrag := "" // guard fragment for an arm whose body brace opens next

	// composeGuard ANDs all active scope fragments in open order.
	composeGuard := func() string {
		frags := make([]string, 0, len(stack))
		for _, s := range stack {
			frags = append(frags, s.frag)
		}
		return strings.Join(frags, " && ")
	}

	// clearActiveCase removes the active case scope of the innermost switch, if
	// any (used at break/default/next-case). The case scope is only ever cleared
	// when it is the TOP of the stack (a clean pop), preserving the depth-ordering
	// invariant. Callers must guarantee the case is topmost — at default:/next-
	// case any inner loop's `}` has already popped it, and break; checks
	// caseIsTopmost first (so a loop nested inside the case is left alone).
	clearActiveCase := func() {
		if len(switches) == 0 {
			return
		}
		sw := &switches[len(switches)-1]
		if sw.caseIdx >= 0 && sw.caseIdx == len(stack)-1 {
			// Pop the case scope from the top.
			stack = stack[:sw.caseIdx]
		}
		sw.caseIdx = -1
	}

	// caseIsTopmost reports whether the innermost switch's active case scope is
	// the topmost (innermost) scope on the stack — i.e. no loop scope is
	// interposed between the case and the current point. A `break;` is a case
	// terminator only in that situation; otherwise the break belongs to an inner
	// loop and must be ignored by the case logic.
	caseIsTopmost := func() bool {
		if len(switches) == 0 {
			return false
		}
		sw := &switches[len(switches)-1]
		return sw.caseIdx >= 0 && sw.caseIdx == len(stack)-1
	}

	// clearActiveArm removes the active arm scope of the innermost if-chain, if it
	// is topmost on the stack (mirrors clearActiveCase). Called at `else`/`else if`
	// to close the prior arm before opening the next. By the time we reach the next
	// arm header the prior arm's `}` has usually already popped its scope (and the
	// brace-exit clamp set armIdx to -1), so this is typically a no-op.
	clearActiveArm := func() {
		if len(ifChains) == 0 {
			return
		}
		ic := &ifChains[len(ifChains)-1]
		if ic.armIdx >= 0 && ic.armIdx == len(stack)-1 {
			stack = stack[:ic.armIdx]
		}
		ic.armIdx = -1
	}

	for _, line := range lines {
		// Detect a counted-loop header (for/while). The body brace usually
		// opens on the following line, so we stage it as pending and bind it
		// when the depth next increases.
		if pendingLoopVar == "" {
			if isLoopHeader(line) {
				if fc := reForCount.FindStringSubmatch(line); fc != nil {
					pendingLoopVar = fc[1]
				}
			}
		}

		// Detect a switch header; stage its discriminator to bind to the body
		// brace that opens next (which may be on a following line). The
		// discriminator may be a bare var (`switch ( mode )`) or an inline
		// expression (`switch ( CInPacket::Decode1(Index) )`). For an
		// expression we synthesize a stable label; the inline read it contains
		// is still emitted below (with an empty guard, before any case).
		if pendingSwitchVar == "" {
			if sm := reSwitch.FindStringSubmatch(line); sm != nil {
				expr := strings.TrimSpace(sm[1])
				if reBareVar.MatchString(expr) {
					pendingSwitchVar = expr
				} else {
					pendingSwitchVar = "switch"
				}
			}
		}

		// A `case N:` (or `default:`) begins a new case within the innermost
		// switch. Reset any prior active case, then open the new one.
		//
		// Known limitations (out of scope, not fixed here): (a) shared/stacked
		// `case A: case B:` labels — only the last label's guard applies; (b) a
		// fully single-line `switch(x){case 0:{...}}` may leave the body unguarded.
		if len(switches) > 0 {
			if cm := reCase.FindStringSubmatch(line); cm != nil {
				clearActiveCase()
				sw := &switches[len(switches)-1]
				stack = append(stack, scope{depth: sw.bodyDepth, frag: sw.discrim + " == " + cm[1]})
				sw.caseIdx = len(stack) - 1
			} else if strings.HasPrefix(strings.TrimSpace(line), "default:") {
				clearActiveCase()
			} else if strings.Contains(line, "break;") && caseIsTopmost() {
				// Only a `break;` whose case scope is innermost terminates the
				// case. A break inside a loop nested in the case belongs to the
				// loop (popped by brace depth at the loop's `}`) — ignore it here.
				clearActiveCase()
			}
		}

		// if / else-if / else dispatch-arm detection. An arm header stages its
		// guard fragment to bind to the body brace that opens next (mirrors the
		// pending-switch mechanism). reIfEq is checked before reElse so `else if`
		// continues the chain rather than being read as a bare `else`.
		if m := reIfEq.FindStringSubmatch(line); m != nil {
			isElse, disc, lit := m[1] == "else", m[2], m[3]
			if isElse && len(ifChains) > 0 && ifChains[len(ifChains)-1].discrim == disc {
				// `else if (disc == lit)` continues the innermost chain.
				clearActiveArm()
				pendingArmFrag = disc + " == " + lit
			} else {
				// A fresh `if (...)`, or an `else if` on a DIFFERENT discriminator
				// (treated as a new chain). Open a new chain at the current depth.
				pendingArmFrag = disc + " == " + lit
				ifChains = append(ifChains, ifChainEntry{startDepth: braceDepth, discrim: disc, armIdx: -1})
			}
		} else if m := reIfCond.FindStringSubmatch(line); m != nil && isSinglePredicate(m[2]) {
			// Non-equality single-predicate arm (e.g. `if (v5 < 5)`, `if (v5 & 0x10)`):
			// emit the VERBATIM condition as the arm guard. The condition text is the
			// chain's discriminator key (each predicate is its own arm).
			isElse, cond := m[1] == "else", strings.TrimSpace(m[2])
			if isElse && len(ifChains) > 0 && ifChains[len(ifChains)-1].discrim == cond {
				clearActiveArm()
				pendingArmFrag = cond
			} else {
				pendingArmFrag = cond
				ifChains = append(ifChains, ifChainEntry{startDepth: braceDepth, discrim: cond, armIdx: -1})
			}
		} else if reElse.MatchString(line) && len(ifChains) > 0 {
			// Bare `else` — the default arm of the innermost chain.
			clearActiveArm()
			pendingArmFrag = DefaultGuardToken
		}

		opensBrace := strings.Contains(line, "{")

		// A pending counted loop binds to the body that opens here. Handles
		// both same-line `for (...) {` and next-line `{`.
		if pendingLoopVar != "" && opensBrace {
			stack = append(stack, scope{depth: braceDepth + 1, frag: "loop " + pendingLoopVar})
			pendingLoopVar = ""
		}
		// A pending switch binds its discriminator to the body that opens here.
		if pendingSwitchVar != "" && opensBrace {
			switches = append(switches, swEntry{bodyDepth: braceDepth + 1, discrim: pendingSwitchVar, caseIdx: -1})
			pendingSwitchVar = ""
		}
		// A pending if/else arm binds to the body that opens here, pushing a guard
		// scope onto the innermost chain.
		if pendingArmFrag != "" && opensBrace && len(ifChains) > 0 {
			ic := &ifChains[len(ifChains)-1]
			stack = append(stack, scope{depth: braceDepth + 1, frag: pendingArmFrag})
			ic.armIdx = len(stack) - 1
			pendingArmFrag = ""
		}

		// Account for opening braces on this line (body level is post-open).
		braceDepth += strings.Count(line, "{")

		guard := composeGuard()

		// Apply this line's packet-alias bookkeeping (Decode re-add, alias
		// propagation, kill-on-reassign) BEFORE classifying its helper calls, so
		// a helper on the same line is judged against the CURRENT live set.
		updateLiveAliases(line, live, rePktVar)

		// Emit reads / delegates for this line under the composed guard.
		//
		// A Decode/Encode read always wins (the discriminator inline read on a
		// `switch ( CInPacket::Decode1(..) )` line is emitted here even though
		// the same line ALSO registered the switch scope above — the two paths
		// are independent and must not short-circuit each other).
		if m := reDecode.FindStringSubmatch(line); m != nil {
			if op := opName(m[2]); op != "" {
				out = append(out, rawCall{Op: op, Comment: lineComment(line), Guard: guard})
			}
		} else if len(live) > 0 {
			// A NAMED call (Class::method, sub_XXXX, or a bare identifier) that
			// passes a packet alias as a standalone arg is a resolvable helper:
			// emit a Delegate (denylist-filtered). Reserve Unresolved ONLY for a
			// true indirect `(*..)(..)` call passing an alias (no name to
			// resolve) — the anti-BuddyInvite invariant: never fabricate a read.
			emitted := false
			// A leading-`*` indirect dispatch — `(*...)(this, alias)` — has no
			// resolvable name; classify it FIRST so the cast/type tokens inside
			// (e.g. `void (__thiscall **)(...)`) are not mis-read as a call name.
			if !isIndirectDispatch(line) {
				if cm := reNameCall.FindStringSubmatch(line); cm != nil {
					name, args := cm[1], cm[2]
					// The trailing segment of a `::`-qualified name is what
					// classifies a control-flow keyword (e.g. a bare `if (...)`),
					// which is NOT a call. Class::method names never collide.
					bare := name
					if i := strings.LastIndex(name, "::"); i >= 0 {
						bare = name[i+2:]
					}
					if controlKeywords[bare] {
						// A control-flow condition (`if (alias)`, `while (alias)`)
						// — not a call. Suppress both Delegate and the indirect
						// fallback so a null-check does not flood Unresolved.
						emitted = true
					} else if aliasArgMatch(args, live) {
						emitted = true
						if !isDenylisted(name) {
							out = append(out, rawCall{Op: "Delegate", Ref: name, Comment: lineComment(line), Guard: guard})
						}
						// Denylisted named helper that took the alias: recognized
						// (emitted=true) but intentionally NOT emitted — and the
						// indirect fallback below is suppressed.
					}
				}
			}
			if !emitted && passesAnyAliasIndirect(line, live) {
				out = append(out, rawCall{Op: "Unresolved", Comment: "packet var passed to unresolved/indirect call; hand-trace", Guard: guard})
			}
		}

		// Account for closing braces on this line.
		braceDepth -= strings.Count(line, "}")

		// Pop any scopes whose body we have now exited.
		for len(stack) > 0 && braceDepth < stack[len(stack)-1].depth {
			stack = stack[:len(stack)-1]
		}
		// Pop any switches whose body we have now exited, and fix up dangling
		// case indices.
		for len(switches) > 0 && braceDepth < switches[len(switches)-1].bodyDepth {
			switches = switches[:len(switches)-1]
		}
		// A case scope is only valid while its switch is the innermost active
		// switch; if the popped scopes invalidated a caseIdx, clamp it.
		if len(switches) > 0 {
			sw := &switches[len(switches)-1]
			if sw.caseIdx >= len(stack) {
				sw.caseIdx = -1
			}
		}
		// Pop any if-chains whose enclosing scope we have now exited (braceDepth
		// dropped BELOW the chain's startDepth). Arm scopes pop on their own `}`
		// via the scope loop above; the chain entry persists across sibling arms
		// until the whole construct's enclosing block closes.
		for len(ifChains) > 0 && braceDepth < ifChains[len(ifChains)-1].startDepth {
			ifChains = ifChains[:len(ifChains)-1]
		}
		// If the popped scopes invalidated the innermost chain's armIdx (the arm
		// body just closed), clamp it so a following `else`/`else if` reopens cleanly.
		if len(ifChains) > 0 {
			ic := &ifChains[len(ifChains)-1]
			if ic.armIdx >= len(stack) {
				ic.armIdx = -1
			}
		}
	}

	return out, nil
}

// ParseDecompileFields runs ParseDecompile and additionally collects the full
// dispatch case-label set (every switch `case N:` / `if (disc == N)` arm and
// default/else), independent of whether the arm reads. Existing callers of
// ParseDecompile are unchanged. The returned Fields carries the resolved
// (non-Delegate) reads plus the CaseLabels map.
func ParseDecompileFields(text string, dir Direction) (Fields, error) {
	calls, err := ParseDecompile(text, dir)
	if err != nil {
		return Fields{}, err
	}
	labels, multiway := collectCaseLabels(text)
	return Fields{
		Direction:           dir,
		Calls:               toFieldCalls(calls),
		CaseLabels:          labels,
		HasMultiwayDispatch: multiway,
	}, nil
}

// toFieldCalls converts parsed rawCalls to FieldCalls, resolving each op via
// parsePrim. Delegate rawCalls (named sub-function descent) are skipped — label
// collection does not descend; Unresolved markers are preserved as a known gap.
func toFieldCalls(raw []rawCall) []FieldCall {
	out := make([]FieldCall, 0, len(raw))
	for _, c := range raw {
		if c.Op == "Delegate" {
			continue
		}
		op, err := parsePrim(c.Op)
		if err != nil {
			continue
		}
		out = append(out, FieldCall{Op: op, Comment: c.Comment, Guard: c.Guard})
	}
	return out
}

// collectCaseLabels is a focused second pass over the decompile text that records
// every dispatch case label per discriminator (switch `case N:` and `if/else if
// (disc == N)` arms) plus whether a default/else arm exists — WITHOUT emitting
// reads. It reuses the same header regexes and brace-depth/switch-nesting
// bookkeeping as ParseDecompile, kept separate so ParseDecompile's load-bearing
// read-emission logic is untouched.
func collectCaseLabels(text string) (map[string]*CaseSet, bool) {
	labels := map[string]*CaseSet{}
	get := func(disc string) *CaseSet {
		if labels[disc] == nil {
			labels[disc] = &CaseSet{}
		}
		return labels[disc]
	}
	// multiway: set on any `else`/`else if` (a chain reaching a 2nd arm), and below
	// for any switch discriminator with >=2 cases (or >=1 case + default).
	multiway := false

	rawLines := strings.Split(text, "\n")
	lines := make([]string, len(rawLines))
	for i, l := range rawLines {
		lines[i] = stripLinePrefix(l)
	}

	braceDepth := 0
	pendingSwitchVar := ""
	pendingChainDisc := ""
	type swLbl struct {
		bodyDepth int
		discrim   string
	}
	var switches []swLbl
	type icLbl struct {
		startDepth int
		discrim    string
	}
	var chains []icLbl

	for _, line := range lines {
		if pendingSwitchVar == "" {
			if sm := reSwitch.FindStringSubmatch(line); sm != nil {
				expr := strings.TrimSpace(sm[1])
				if reBareVar.MatchString(expr) {
					pendingSwitchVar = expr
				} else {
					pendingSwitchVar = "switch"
				}
			}
		}

		// Any `else`-led header (`else`, `else if (...)` with ANY condition —
		// including a compound `&&`/`||` predicate the verbatim emitter cannot
		// represent) means the chain reached a 2nd arm: a multi-way dispatch. A lone
		// `if` (no else) is not multi-way.
		if reElseLed.MatchString(line) {
			multiway = true
		}

		// switch case / default attribution to the innermost switch discriminator.
		if len(switches) > 0 {
			if cm := reCase.FindStringSubmatch(line); cm != nil {
				if v, ok := parseIntLit(cm[1]); ok {
					get(switches[len(switches)-1].discrim).add(v)
				}
			} else if strings.HasPrefix(strings.TrimSpace(line), "default:") {
				get(switches[len(switches)-1].discrim).Default = true
			}
		}

		// if / else-if / else arm labels. The discriminator is captured directly
		// for == arms; a new chain is staged so a later bare `else` attributes its
		// default to the right discriminator.
		if m := reIfEq.FindStringSubmatch(line); m != nil {
			isElse, disc, lit := m[1] == "else", m[2], m[3]
			if v, ok := parseIntLit(lit); ok {
				get(disc).add(v)
			}
			continuation := isElse && len(chains) > 0 && chains[len(chains)-1].discrim == disc
			if !continuation {
				pendingChainDisc = disc
			}
		} else if reElse.MatchString(line) && len(chains) > 0 {
			get(chains[len(chains)-1].discrim).Default = true
		}

		opensBrace := strings.Contains(line, "{")
		if pendingSwitchVar != "" && opensBrace {
			switches = append(switches, swLbl{bodyDepth: braceDepth + 1, discrim: pendingSwitchVar})
			pendingSwitchVar = ""
		}
		if pendingChainDisc != "" && opensBrace {
			chains = append(chains, icLbl{startDepth: braceDepth, discrim: pendingChainDisc})
			pendingChainDisc = ""
		}

		braceDepth += strings.Count(line, "{")
		braceDepth -= strings.Count(line, "}")

		for len(switches) > 0 && braceDepth < switches[len(switches)-1].bodyDepth {
			switches = switches[:len(switches)-1]
		}
		for len(chains) > 0 && braceDepth < chains[len(chains)-1].startDepth {
			chains = chains[:len(chains)-1]
		}
	}

	// A switch with >=2 case labels (or >=1 case + a default) is multi-way.
	for _, cs := range labels {
		if len(cs.Values()) >= 2 || (len(cs.Values()) >= 1 && cs.Default) {
			multiway = true
		}
	}
	if len(labels) == 0 {
		return nil, multiway
	}
	return labels, multiway
}

// isSinglePredicate reports whether a condition is a single readable predicate
// suitable for a verbatim arm guard: no boolean combinator (&&/||) and not an
// indirect/function-pointer expression. Equality "x == N" is handled by reIfEq
// before this is consulted, so it never reaches here.
func isSinglePredicate(cond string) bool {
	if strings.Contains(cond, "&&") || strings.Contains(cond, "||") {
		return false
	}
	if strings.Contains(cond, "(*") || strings.HasPrefix(strings.TrimSpace(cond), "*") {
		return false
	}
	return strings.TrimSpace(cond) != ""
}

// isLoopHeader reports whether a line begins a for/while/do loop.
func isLoopHeader(line string) bool {
	t := strings.TrimSpace(line)
	return strings.HasPrefix(t, "for ") || strings.HasPrefix(t, "for(") ||
		strings.HasPrefix(t, "while ") || strings.HasPrefix(t, "while(") ||
		strings.HasPrefix(t, "do ") || t == "do" || strings.HasPrefix(t, "do{")
}
