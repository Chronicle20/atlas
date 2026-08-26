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
