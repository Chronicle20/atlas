// Package wzdiff compares a WZ property tree, as parsed by this repository,
// against a HaRepacker-style ".img.xml" dump of the same image. Both sides
// are reduced to the same []Node shape so the structural differ never has
// to know which side came from which source.
package wzdiff

import (
	"strings"

	"github.com/Chronicle20/atlas/libs/atlas-wz/wz/wzxml"
)

// Node is the common tree shape both sides of the comparison are reduced
// to. Kind is the XML local name (imgdir, int, canvas, vector, uol,
// string, short, long, float, double, sound, extended, null).
type Node struct {
	Kind     string
	Name     string
	Attrs    map[string]string
	Children []Node
}

// pathOf renders the path contributed by a chain of nodes from an implicit
// root, in the /<Kind>:<Name> notation used by the evidence file, e.g.
// pathOf(imgdir:0, imgdir:event, int:state) => "/imgdir:0/imgdir:event/int:state".
func pathOf(chain ...Node) string {
	var b strings.Builder
	for _, n := range chain {
		b.WriteByte('/')
		b.WriteString(n.Kind)
		b.WriteByte(':')
		b.WriteString(n.Name)
	}
	return b.String()
}

// renderAttrs renders a node's Attrs map into the evidence file's
// attribute notation for the given Kind: "value=X" for value-bearing
// kinds, "height=H width=W" for canvas, "x=X y=Y" for vector, and empty
// for containers (imgdir, extended) and value-less leaves (null, sound).
func renderAttrs(kind string, attrs map[string]string) string {
	switch kind {
	case "canvas":
		return "height=" + attrs["height"] + " width=" + attrs["width"]
	case "vector":
		return "x=" + attrs["x"] + " y=" + attrs["y"]
	case "imgdir", "extended":
		return ""
	default:
		if v, ok := attrs["value"]; ok {
			return "value=" + v
		}
		return ""
	}
}

// FromElements converts a slice of wzxml.Element (the in-memory parse's
// XML-shaped output) into the common []Node tree, preserving order.
func FromElements(els []wzxml.Element) []Node {
	if len(els) == 0 {
		return nil
	}
	out := make([]Node, 0, len(els))
	for _, e := range els {
		out = append(out, fromElement(e))
	}
	return out
}

func fromElement(e wzxml.Element) Node {
	n := Node{
		Kind: e.XMLName.Local,
		Name: e.Name,
	}
	switch n.Kind {
	case "canvas":
		n.Attrs = map[string]string{"width": e.Width, "height": e.Height}
	case "vector":
		n.Attrs = map[string]string{"x": e.X, "y": e.Y}
	case "imgdir", "extended", "null", "sound":
		// container or value-less leaf: no attrs
	default:
		n.Attrs = map[string]string{"value": e.Value}
	}
	n.Children = FromElements(e.Children)
	return n
}
