package wzdiff

import "testing"

func intNode(name, value string) Node {
	return Node{Kind: "int", Name: name, Attrs: map[string]string{"value": value}}
}

func TestDiff(t *testing.T) {
	t.Run("identical", func(t *testing.T) {
		ours := []Node{intNode("state", "1")}
		reference := []Node{intNode("state", "1")}
		deltas := Diff(ours, reference)
		if len(deltas) != 0 {
			t.Fatalf("deltas = %+v, want none", deltas)
		}
	})

	t.Run("lost subtree", func(t *testing.T) {
		ours := []Node{}
		reference := []Node{
			{
				Kind: "imgdir", Name: "event",
				Children: []Node{
					{
						Kind: "imgdir", Name: "0",
						Children: []Node{intNode("state", "1")},
					},
				},
			},
		}
		deltas := Diff(ours, reference)
		wantPaths := []string{
			"/imgdir:event",
			"/imgdir:event/imgdir:0",
			"/imgdir:event/imgdir:0/int:state",
		}
		if len(deltas) != len(wantPaths) {
			t.Fatalf("len(deltas) = %d, want %d: %+v", len(deltas), len(wantPaths), deltas)
		}
		for i, d := range deltas {
			if d.Path != wantPaths[i] {
				t.Errorf("deltas[%d].Path = %q, want %q", i, d.Path, wantPaths[i])
			}
			if d.OnlyIn != "reference" {
				t.Errorf("deltas[%d].OnlyIn = %q, want %q", i, d.OnlyIn, "reference")
			}
		}
	})

	t.Run("gained subtree", func(t *testing.T) {
		ours := []Node{
			{
				Kind: "imgdir", Name: "event",
				Children: []Node{intNode("timeOut", "2000")},
			},
		}
		reference := []Node{}
		deltas := Diff(ours, reference)
		wantPaths := []string{
			"/imgdir:event",
			"/imgdir:event/int:timeOut",
		}
		if len(deltas) != len(wantPaths) {
			t.Fatalf("len(deltas) = %d, want %d: %+v", len(deltas), len(wantPaths), deltas)
		}
		for i, d := range deltas {
			if d.Path != wantPaths[i] {
				t.Errorf("deltas[%d].Path = %q, want %q", i, d.Path, wantPaths[i])
			}
			if d.OnlyIn != "ours" {
				t.Errorf("deltas[%d].OnlyIn = %q, want %q", i, d.OnlyIn, "ours")
			}
		}
	})

	t.Run("mangled name", func(t *testing.T) {
		ours := []Node{intNode("rpeat", "1")}
		reference := []Node{intNode("repeat", "1")}
		deltas := Diff(ours, reference)
		if len(deltas) != 2 {
			t.Fatalf("len(deltas) = %d, want 2: %+v", len(deltas), deltas)
		}
		if deltas[0].Path != "/int:repeat" || deltas[0].OnlyIn != "reference" {
			t.Errorf("deltas[0] = %+v, want {/int:repeat reference}", deltas[0])
		}
		if deltas[1].Path != "/int:rpeat" || deltas[1].OnlyIn != "ours" {
			t.Errorf("deltas[1] = %+v, want {/int:rpeat ours}", deltas[1])
		}
	})

	t.Run("wrong scalar", func(t *testing.T) {
		ours := []Node{intNode("state", "1")}
		reference := []Node{intNode("state", "0")}
		deltas := Diff(ours, reference)
		if len(deltas) != 2 {
			t.Fatalf("len(deltas) = %d, want 2: %+v", len(deltas), deltas)
		}
		byDir := map[string]Delta{}
		for _, d := range deltas {
			if d.Path != "/int:state" {
				t.Errorf("delta at unexpected path: %+v", d)
			}
			byDir[d.OnlyIn] = d
		}
		if byDir["ours"].Attrs != "value=1" {
			t.Errorf("ours delta = %+v, want value=1", byDir["ours"])
		}
		if byDir["reference"].Attrs != "value=0" {
			t.Errorf("reference delta = %+v, want value=0", byDir["reference"])
		}
	})

	t.Run("collapsed canvas", func(t *testing.T) {
		ours := []Node{{Kind: "canvas", Name: "0", Attrs: map[string]string{"height": "1", "width": "1"}}}
		reference := []Node{{Kind: "canvas", Name: "0", Attrs: map[string]string{"height": "121", "width": "100"}}}
		deltas := Diff(ours, reference)
		if len(deltas) != 2 {
			t.Fatalf("len(deltas) = %d, want 2: %+v", len(deltas), deltas)
		}
		byDir := map[string]Delta{}
		for _, d := range deltas {
			if d.Path != "/canvas:0" {
				t.Errorf("delta at unexpected path: %+v", d)
			}
			byDir[d.OnlyIn] = d
		}
		if byDir["ours"].Attrs != "height=1 width=1" {
			t.Errorf("ours delta = %+v, want height=1 width=1", byDir["ours"])
		}
		if byDir["reference"].Attrs != "height=121 width=100" {
			t.Errorf("reference delta = %+v, want height=121 width=100", byDir["reference"])
		}
	})

	t.Run("uol vs canvas", func(t *testing.T) {
		ours := []Node{{Kind: "uol", Name: "0", Attrs: map[string]string{"value": "../0/0"}}}
		reference := []Node{{Kind: "canvas", Name: "0", Attrs: map[string]string{"height": "1", "width": "1"}}}
		deltas := Diff(ours, reference)
		if len(deltas) != 2 {
			t.Fatalf("len(deltas) = %d, want 2: %+v", len(deltas), deltas)
		}
		paths := map[string]string{}
		for _, d := range deltas {
			paths[d.Path] = d.OnlyIn
		}
		if paths["/canvas:0"] != "reference" {
			t.Errorf("paths = %+v, want /canvas:0 => reference", paths)
		}
		if paths["/uol:0"] != "ours" {
			t.Errorf("paths = %+v, want /uol:0 => ours", paths)
		}
	})

	t.Run("duplicate sibling names are not silently collapsed", func(t *testing.T) {
		// Two siblings share the same Kind:Name under one parent. Before the
		// flatten fix, the second write into the path->attrs map silently
		// overwrote the first, so one of the two int:state values (and its
		// Delta) vanished with no diagnostic. Both must now surface.
		ours := []Node{
			{
				Kind: "imgdir", Name: "0",
				Children: []Node{
					intNode("state", "1"),
					intNode("state", "2"),
				},
			},
		}
		reference := []Node{}
		deltas := Diff(ours, reference)
		wantPaths := map[string]bool{
			"/imgdir:0":             false,
			"/imgdir:0/int:state":   false,
			"/imgdir:0/int:state#2": false,
		}
		if len(deltas) != len(wantPaths) {
			t.Fatalf("len(deltas) = %d, want %d: %+v", len(deltas), len(wantPaths), deltas)
		}
		attrsByPath := map[string]string{}
		for _, d := range deltas {
			if _, ok := wantPaths[d.Path]; !ok {
				t.Errorf("unexpected delta path %q: %+v", d.Path, d)
				continue
			}
			wantPaths[d.Path] = true
			if d.OnlyIn != "ours" {
				t.Errorf("delta %+v, want OnlyIn=ours", d)
			}
			attrsByPath[d.Path] = d.Attrs
		}
		for path, seen := range wantPaths {
			if !seen {
				t.Errorf("missing expected delta path %q", path)
			}
		}
		if attrsByPath["/imgdir:0/int:state"] != "value=1" {
			t.Errorf("attrs at /imgdir:0/int:state = %q, want value=1", attrsByPath["/imgdir:0/int:state"])
		}
		if attrsByPath["/imgdir:0/int:state#2"] != "value=2" {
			t.Errorf("attrs at /imgdir:0/int:state#2 = %q, want value=2", attrsByPath["/imgdir:0/int:state#2"])
		}
	})

	t.Run("ordering-insensitive", func(t *testing.T) {
		ours := []Node{intNode("a", "1"), intNode("b", "2")}
		reference := []Node{intNode("b", "2"), intNode("a", "1")}
		deltas := Diff(ours, reference)
		if len(deltas) != 0 {
			t.Fatalf("deltas = %+v, want none", deltas)
		}
	})
}
