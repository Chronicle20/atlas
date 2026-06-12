package idasrc

import (
	"context"
	"fmt"
)

type HarvestOpts struct {
	DescentDepth int    // max helper recursion depth; 0 => default 6
	Binary       string // provenance
	MD5          string
	GeneratedAt  string
	// DirOf resolves a roster function's packet direction so each function is
	// parsed against the correct class (clientbound → CInPacket::Decode*,
	// serverbound → COutPacket::Encode*). nil → all functions default to
	// DirClientbound. A discovered helper INHERITS its discoverer's direction
	// (DirOf is consulted only for roster roots; descendants reuse the parent's
	// direction so a clientbound handler's helpers read CInPacket and a
	// serverbound Send-function's helpers write COutPacket).
	DirOf func(name string) Direction
}

// Harvest drives the MCP client over the roster, parsing each function and
// enqueuing every discovered packet-reading helper (Delegate ref) as its own
// export entry. Cycle-guarded (visited set) and depth-bounded; a function past
// the depth bound on a still-descending path is marked Unresolved rather than
// silently truncated.
func Harvest(ctx context.Context, c MCPClient, roster []string, opts HarvestOpts) (exportFile, error) {
	if opts.DescentDepth == 0 {
		opts.DescentDepth = 6
	}
	out := exportFile{Binary: opts.Binary, MD5: opts.MD5,
		GeneratedAt: opts.GeneratedAt, Functions: map[string]exportFn{}}
	// dirFor resolves a roster ROOT's direction (DirClientbound default when no
	// DirOf is supplied). Descendants do not consult DirOf — they inherit the
	// parent's direction via the queue item below.
	dirFor := func(name string) Direction {
		if opts.DirOf == nil {
			return DirClientbound
		}
		return opts.DirOf(name)
	}
	type item struct {
		name  string
		depth int
		dir   Direction // direction to parse this function in (root: DirOf; helper: inherited)
	}
	queue := make([]item, 0, len(roster))
	for _, n := range roster {
		queue = append(queue, item{n, 0, dirFor(n)})
	}
	visited := map[string]bool{}
	for len(queue) > 0 {
		it := queue[0]
		queue = queue[1:]
		if visited[it.name] {
			continue
		}
		visited[it.name] = true
		addr, ok, err := c.GetFunctionByName(ctx, it.name)
		if err != nil {
			return out, fmt.Errorf("harvest %s: %w", it.name, err)
		}
		if !ok {
			out.Functions[it.name] = exportFn{Unresolved: true,
				Calls: []rawCall{{Op: "Unresolved", Comment: "function not found in IDB"}}}
			continue
		}
		text, err := c.DecompileFunction(ctx, addr)
		if err != nil {
			if IsDecompilationFailed(err) {
				out.Functions[it.name] = exportFn{Unresolved: true,
					Calls: []rawCall{{Op: "Unresolved", Comment: "decompilation failed; hand-trace"}}}
				continue
			}
			return out, fmt.Errorf("harvest %s decompile: %w", it.name, err)
		}
		calls, err := ParseDecompile(text, it.dir)
		if err != nil {
			return out, fmt.Errorf("harvest %s parse: %w", it.name, err)
		}
		fn := exportFn{Address: addr, Calls: calls}
		for i := range calls {
			cl := calls[i]
			if cl.Op == "Delegate" && cl.Ref != "" && !visited[cl.Ref] {
				if it.depth+1 > opts.DescentDepth {
					// descent too deep to prove: neutralize the dangling
					// Delegate ref to an Unresolved op so the export stays
					// self-consistent (resolver emits a known gap instead of
					// hard-erroring on a ref that was never harvested).
					fn.Calls[i] = rawCall{Op: "Unresolved",
						Comment: "descent depth exceeded; hand-trace", Guard: cl.Guard}
					fn.Unresolved = true
					continue
				}
				// Helper inherits the discoverer's direction.
				queue = append(queue, item{cl.Ref, it.depth + 1, it.dir})
			}
		}
		out.Functions[it.name] = fn
	}
	return out, nil
}
