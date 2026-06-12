package idasrc

import "context"

// ResolveLive decompiles the function at `address`, parses it in `dir`, descends the
// discovered packet-reading helpers BY ADDRESS, and returns the fully resolved
// Fields. A decompilation soft-fail on the BASE returns the error verbatim (caller
// checks IsDecompilationFailed -> unverifiable). A soft-fail on a DESCENDED helper
// becomes an Unresolved entry (a known gap, never a false verdict).
//
// Address-based descent (the task-081 fix): the new ida-pro-mcp server returns
// "Not found" from GetFunctionByName for DEMANGLED Class::Method helper names, so
// name-based descent bottomed out in Unresolved. Instead we descend purely by
// ADDRESS using each parent's `callees` (which carry the callee's address + its
// MANGLED symbol). We key the callee map by BOTH the raw callee Name (so a
// sub_XXXX ref matches verbatim) AND its demangleQualified form (so a Delegate ref
// like "CWvsContext::CFriend::Reset" matches the mangled
// "?Reset@CFriend@CWvsContext@@..."). A Delegate ref with no matching callee is
// neutralized to Unresolved so the export stays self-consistent (the resolver
// never hard-errors on a ref pointing at a missing entry).
func ResolveLive(ctx context.Context, c MCPClient, address string, dir Direction, opts HarvestOpts) (Fields, error) {
	depthBound := opts.DescentDepth
	if depthBound == 0 {
		depthBound = 6
	}
	dirStr := "clientbound"
	if dir == DirServerbound {
		dirStr = "serverbound"
	}

	out := exportFile{
		Binary:      opts.Binary,
		MD5:         opts.MD5,
		GeneratedAt: opts.GeneratedAt,
		Functions:   map[string]exportFn{},
	}

	const rootName = "__base__"

	// baseText captures the BASE function's raw decompile so the dispatch
	// case-label set can be collected (for the case<->mode bijection check). Only
	// the base's structure matters — descended helpers are linear reads.
	var baseText string

	// keyFor assigns a stable export key per (addr). The root uses rootName; each
	// descended helper is keyed by the Delegate ref that reached it (the parser's
	// demangled call-site name or sub_XXXX), which is exactly what the Delegate ref
	// in the parent's Calls points at — keeping the export self-consistent.
	visited := map[string]bool{} // addr -> already exported (cycle guard)

	// descend exports the function at addr under key `name`, recursing into its
	// packet-reading helpers via callees. rootErr is returned non-nil ONLY for a
	// base (depth 0) soft-fail so the caller maps it to unverifiable; descended
	// soft-fails are recorded inline as Unresolved.
	var descend func(addr, name string, depth int) error
	descend = func(addr, name string, depth int) error {
		visited[addr] = true
		text, err := c.DecompileFunction(ctx, addr)
		if err != nil {
			if depth == 0 {
				// Base soft-fail (or hard error): surface verbatim so the caller's
				// IsDecompilationFailed check maps it to unverifiable.
				return err
			}
			if IsDecompilationFailed(err) {
				out.Functions[name] = exportFn{Address: addr, Direction: dirStr, Unresolved: true,
					Calls: []rawCall{{Op: "Unresolved", Comment: "decompilation failed; hand-trace"}}}
				return nil
			}
			return err
		}
		if depth == 0 {
			baseText = text
		}
		calls, err := ParseDecompile(text, dir)
		if err != nil {
			return err
		}

		// Build the callee lookup: raw Name → addr AND demangled Name → addr.
		// Tolerate a GetCallees error (treat as no callees). First key wins on
		// collision.
		var cm map[string]string
		if cs, cerr := c.GetCallees(ctx, addr); cerr == nil {
			cm = make(map[string]string, len(cs)*2)
			for _, ce := range cs {
				if ce.Name != "" {
					if _, ok := cm[ce.Name]; !ok {
						cm[ce.Name] = ce.Addr
					}
				}
				if dem, ok := demangleQualified(ce.Name); ok {
					if _, exists := cm[dem]; !exists {
						cm[dem] = ce.Addr
					}
				}
			}
		}

		// Resolve each Delegate ref to a child address via the callee map, descend
		// by address, and neutralize any ref we cannot reach (missing callee or
		// past the depth bound) so the export never points at a missing entry.
		for i := range calls {
			cl := calls[i]
			if cl.Op != "Delegate" || cl.Ref == "" {
				continue
			}
			childAddr, ok := cm[cl.Ref]
			if !ok || childAddr == "" || depth+1 > depthBound {
				// Unreachable helper: neutralize to keep the export self-consistent.
				comment := "delegate ref not found in callees; hand-trace"
				if ok && childAddr != "" {
					comment = "descent depth exceeded; hand-trace"
				}
				calls[i] = rawCall{Op: "Unresolved", Comment: comment, Guard: cl.Guard}
				continue
			}
			if !visited[childAddr] {
				if err := descend(childAddr, cl.Ref, depth+1); err != nil {
					return err
				}
			}
		}

		out.Functions[name] = exportFn{Address: addr, Direction: dirStr, Calls: calls}
		return nil
	}

	if err := descend(address, rootName, 0); err != nil {
		return Fields{}, err
	}
	f, err := newExportSourceFromFile(out).Resolve(ctx, rootName)
	if err != nil {
		return Fields{}, err
	}
	labels, multiway := collectCaseLabels(baseText)
	f.CaseLabels = labels
	f.HasMultiwayDispatch = multiway
	return f, nil
}
