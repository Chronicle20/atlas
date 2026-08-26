package wzdiff

import (
	"fmt"
	"sort"
)

// Delta describes a single structural divergence between the "ours" and
// "reference" trees at one path.
type Delta struct {
	Path   string
	Attrs  string
	OnlyIn string // "reference" or "ours"
}

// String renders a Delta in the notation used by
// evidence-wz-parse-divergence-reactor.txt: a 6-space indent, the path,
// " | ", and the rendered attrs (empty for containers), e.g.
// "      /imgdir:0/int:state | value=1".
func (d Delta) String() string {
	return "      " + d.Path + " | " + d.Attrs
}

// Diff compares ours against reference and returns every path whose
// rendered attrs are not present, identically, on both sides. The
// comparison is a set over (path, attrs), not order-sensitive: a
// HaRepacker dump's element order is not our decode order, and the
// evidence file already compares as sets. A path present on only one side
// yields one Delta; a path present on both sides with differing attrs
// yields two Deltas, one per direction. Deltas are emitted in sorted path
// order, reference before ours at a shared path, for stable output.
func Diff(ours, reference []Node) []Delta {
	oursFlat := flatten(ours, "")
	refFlat := flatten(reference, "")

	seen := make(map[string]struct{}, len(oursFlat)+len(refFlat))
	for path := range oursFlat {
		seen[path] = struct{}{}
	}
	for path := range refFlat {
		seen[path] = struct{}{}
	}
	paths := make([]string, 0, len(seen))
	for path := range seen {
		paths = append(paths, path)
	}
	sort.Strings(paths)

	var deltas []Delta
	for _, path := range paths {
		oAttrs, oOk := oursFlat[path]
		rAttrs, rOk := refFlat[path]
		if oOk && rOk && oAttrs == rAttrs {
			continue
		}
		if rOk {
			deltas = append(deltas, Delta{Path: path, Attrs: rAttrs, OnlyIn: "reference"})
		}
		if oOk {
			deltas = append(deltas, Delta{Path: path, Attrs: oAttrs, OnlyIn: "ours"})
		}
	}
	return deltas
}

// flatten reduces a tree to a map of path -> rendered attrs.
//
// A path is normally unique per sibling group (Kind:Name), but WZ data is
// not guaranteed to keep sibling names unique, so flatten guards against a
// silent collision: the first occurrence of a given Kind:Name under a
// parent keeps the plain "/<Kind>:<Name>" path, and every later occurrence
// gets a "#N" suffix for its 2nd, 3rd, ... appearance. This never changes
// the path of a non-colliding node (the common case, and the one the
// evidence file's format is pinned to), and it makes a duplicate sibling
// visible as its own Delta instead of being silently overwritten in the
// map and dropped from the diff.
func flatten(nodes []Node, prefix string) map[string]string {
	out := map[string]string{}
	counts := map[string]int{}
	for _, n := range nodes {
		base := prefix + "/" + n.Kind + ":" + n.Name
		counts[base]++
		path := base
		if counts[base] > 1 {
			path = fmt.Sprintf("%s#%d", base, counts[base])
		}
		out[path] = renderAttrs(n.Kind, n.Attrs)
		for k, v := range flatten(n.Children, path) {
			out[k] = v
		}
	}
	return out
}
