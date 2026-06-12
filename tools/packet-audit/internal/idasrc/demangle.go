package idasrc

import "strings"

// demangleQualified converts a simple MSVC member-function mangled name
// "?<name>@<scope1>@<scope2>@...@@<signature>" to the demangled qualified form
// "<scopeN>::...::<scope1>::<name>" (scopes are reversed). Returns ok=false for
// names it cannot simply demangle (templates "?$", operators "??", or a non-"?"
// name — callers use those verbatim or skip).
//
// This is a deliberately tiny demangler: the live ida-pro-mcp server returns each
// callee's MANGLED symbol, while the decompile parser sees the DEMANGLED call-site
// name (e.g. "CWvsContext::CFriend::Reset"). Address-based descent keys callees by
// their demangled form so a Delegate ref matches; only the simple member-function
// shape is needed for that match, so anything fancier (templates, operators,
// special types) is left to the caller (matched verbatim by raw Name or skipped).
func demangleQualified(mangled string) (string, bool) {
	// Must start with a single '?' (not "??" operator) and the first scope token
	// must not be a template ("?$..." → after the leading '?', a '$').
	if !strings.HasPrefix(mangled, "?") {
		return "", false
	}
	rest := mangled[1:]
	if rest == "" || rest[0] == '?' || rest[0] == '$' {
		return "", false
	}
	// Take the substring between the leading '?' and the first "@@" (signature
	// separator).
	end := strings.Index(rest, "@@")
	if end < 0 {
		return "", false
	}
	body := rest[:end]
	// Split on '@' → [name, scope1, scope2, ...].
	toks := strings.Split(body, "@")
	for _, tok := range toks {
		if tok == "" || strings.ContainsAny(tok, "$?<") {
			return "", false
		}
	}
	name := toks[0]
	scopes := toks[1:]
	if len(scopes) == 0 {
		return name, true
	}
	// Reverse scopes: [scope1, scope2, ...] → "scopeN::...::scope1".
	rev := make([]string, len(scopes))
	for i, s := range scopes {
		rev[len(scopes)-1-i] = s
	}
	return strings.Join(rev, "::") + "::" + name, true
}
