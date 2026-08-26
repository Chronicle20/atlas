package wzdiff

import (
	"os"
	"path/filepath"
	"testing"
)

const sampleImageXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<imgdir name="2406000.img">
  <imgdir name="info">
    <string name="name" value="x"/>
    <int name="activateByTouch" value="1"/>
  </imgdir>
  <imgdir name="0">
    <canvas name="0" width="100" height="121">
      <vector name="origin" x="49" y="121"/>
    </canvas>
  </imgdir>
</imgdir>
`

func writeSample(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "2406000.img.xml")
	if err := os.WriteFile(path, []byte(sampleImageXML), 0o644); err != nil {
		t.Fatalf("os.WriteFile: %v", err)
	}
	return path
}

func TestLoadImageXML(t *testing.T) {
	path := writeSample(t)

	nodes, err := LoadImageXML(path)
	if err != nil {
		t.Fatalf("LoadImageXML: %v", err)
	}

	if len(nodes) != 2 {
		t.Fatalf("len(nodes) = %d, want 2", len(nodes))
	}

	info := nodes[0]
	if info.Kind != "imgdir" || info.Name != "info" {
		t.Fatalf("nodes[0] = %+v, want imgdir:info", info)
	}
	if len(info.Children) != 2 {
		t.Fatalf("len(info.Children) = %d, want 2", len(info.Children))
	}
	if info.Children[0].Kind != "string" || info.Children[1].Kind != "int" {
		t.Fatalf("info.Children kinds = [%q %q], want [string int]",
			info.Children[0].Kind, info.Children[1].Kind)
	}

	zero := nodes[1]
	if zero.Kind != "imgdir" || zero.Name != "0" {
		t.Fatalf("nodes[1] = %+v, want imgdir:0", zero)
	}
	if len(zero.Children) != 1 {
		t.Fatalf("len(zero.Children) = %d, want 1", len(zero.Children))
	}
	canvas := zero.Children[0]
	if canvas.Kind != "canvas" || canvas.Name != "0" {
		t.Fatalf("zero.Children[0] = %+v, want canvas:0", canvas)
	}
	if canvas.Attrs["width"] != "100" || canvas.Attrs["height"] != "121" {
		t.Errorf("canvas.Attrs = %+v, want width=100 height=121", canvas.Attrs)
	}
	if len(canvas.Children) != 1 {
		t.Fatalf("len(canvas.Children) = %d, want 1", len(canvas.Children))
	}
	vector := canvas.Children[0]
	if vector.Kind != "vector" || vector.Name != "origin" {
		t.Fatalf("canvas.Children[0] = %+v, want vector:origin", vector)
	}
	if vector.Attrs["x"] != "49" || vector.Attrs["y"] != "121" {
		t.Errorf("vector.Attrs = %+v, want x=49 y=121", vector.Attrs)
	}
}

const scalarKindsXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<imgdir name="scalars.img">
  <imgdir name="values">
    <short name="s" value="1"/>
    <long name="l" value="2"/>
    <float name="f" value="1.5"/>
    <double name="d" value="2.5"/>
    <null name="n"/>
    <sound name="snd" value="0"/>
  </imgdir>
</imgdir>
`

// TestLoadImageXML_ScalarElementKinds covers the scalar leaf Kinds
// (short, long, float, double, null, sound) that decodeElement handles
// generically alongside int/string. Kind and value=-bearing attrs must
// come through untouched for each of them.
func TestLoadImageXML_ScalarElementKinds(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "scalars.img.xml")
	if err := os.WriteFile(path, []byte(scalarKindsXML), 0o644); err != nil {
		t.Fatalf("os.WriteFile: %v", err)
	}

	nodes, err := LoadImageXML(path)
	if err != nil {
		t.Fatalf("LoadImageXML: %v", err)
	}
	if len(nodes) != 1 || nodes[0].Kind != "imgdir" || nodes[0].Name != "values" {
		t.Fatalf("nodes = %+v, want single imgdir:values", nodes)
	}

	children := nodes[0].Children
	want := []struct {
		kind, name, value string
		hasValue          bool
	}{
		{"short", "s", "1", true},
		{"long", "l", "2", true},
		{"float", "f", "1.5", true},
		{"double", "d", "2.5", true},
		{"null", "n", "", false},
		{"sound", "snd", "0", true},
	}
	if len(children) != len(want) {
		t.Fatalf("len(children) = %d, want %d: %+v", len(children), len(want), children)
	}
	for i, w := range want {
		c := children[i]
		if c.Kind != w.kind || c.Name != w.name {
			t.Errorf("children[%d] = %+v, want Kind=%q Name=%q", i, c, w.kind, w.name)
		}
		v, ok := c.Attrs["value"]
		if ok != w.hasValue {
			t.Errorf("children[%d] (%s:%s) Attrs[value] present = %v, want %v", i, w.kind, w.name, ok, w.hasValue)
			continue
		}
		if w.hasValue && v != w.value {
			t.Errorf("children[%d] (%s:%s) Attrs[value] = %q, want %q", i, w.kind, w.name, v, w.value)
		}
	}
}

const emptyElementXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<imgdir name="empty.img">
  <imgdir name="container">
    <null name="empty"/>
  </imgdir>
</imgdir>
`

// TestLoadImageXML_EmptyElement covers a self-closing element with no
// attributes and no children, distinct from the sample's self-closing
// <vector/>/<canvas/> which both carry attributes.
func TestLoadImageXML_EmptyElement(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.img.xml")
	if err := os.WriteFile(path, []byte(emptyElementXML), 0o644); err != nil {
		t.Fatalf("os.WriteFile: %v", err)
	}

	nodes, err := LoadImageXML(path)
	if err != nil {
		t.Fatalf("LoadImageXML: %v", err)
	}
	if len(nodes) != 1 || nodes[0].Kind != "imgdir" || nodes[0].Name != "container" {
		t.Fatalf("nodes = %+v, want single imgdir:container", nodes)
	}
	if len(nodes[0].Children) != 1 {
		t.Fatalf("len(container.Children) = %d, want 1", len(nodes[0].Children))
	}
	empty := nodes[0].Children[0]
	if empty.Kind != "null" || empty.Name != "empty" {
		t.Fatalf("Children[0] = %+v, want null:empty", empty)
	}
	if len(empty.Attrs) != 0 {
		t.Errorf("empty.Attrs = %+v, want none", empty.Attrs)
	}
	if len(empty.Children) != 0 {
		t.Errorf("empty.Children = %+v, want none", empty.Children)
	}
}
