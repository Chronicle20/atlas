package wz

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-wz/wz/property"
	"github.com/Chronicle20/atlas/libs/atlas-wz/wztest"
)

// dedupImage builds the fixture image whose property names ("repeat") and
// extended type tags ("Shape2D#Vector2D") repeat within a single image, so
// the second occurrence of each must go through the offset-referenced
// string block path.
func dedupImage() wztest.Image {
	return wztest.Img("dedup",
		wztest.Sub("0",
			wztest.Int("state", 1),
			wztest.Int("repeat", 1),
			wztest.Vector("lt", -48, -48),
		),
		wztest.Sub("1",
			wztest.Int("state", 2),
			wztest.Int("repeat", 1),
			wztest.Vector("lt", -48, -48),
		),
	)
}

// stringBlockTags records, via SetTrace, the actual tag byte the parser
// consumed at the start of every "stringblock" (property name) and
// "extended" (type-tag string) event. This reads the real tag byte the
// builder wrote at the position the parser itself identifies as a
// string-block start, rather than scanning the whole file for a byte
// value — plain WZ ints, checksums and directory-offset fields legitimately
// contain the bytes 0x01/0x1B too, so a blind file-wide scan is unreliable.
func stringBlockTags(t *testing.T, f *File, data []byte) map[byte]bool {
	t.Helper()
	tags := make(map[byte]bool)
	f.SetTrace(func(ev TraceEvent) {
		if ev.Kind != "stringblock" && ev.Kind != "extended" {
			return
		}
		if ev.StartOff < 0 || ev.StartOff >= int64(len(data)) {
			t.Fatalf("trace event %+v: StartOff out of range", ev)
		}
		tags[data[ev.StartOff]] = true
	})
	return tags
}

// buildAndOpenDedup materializes b as a fixture file and opens it, handing
// back the tag bytes the parser saw at every string-block/extended-tag
// start, and the parsed "dedup" image's Sub("0") and Sub("1") child
// property lists.
func buildAndOpenDedup(t *testing.T, b *wztest.Builder) (tags map[byte]bool, zero, one []property.Property) {
	t.Helper()
	data, err := b.Build()
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	path := filepath.Join(t.TempDir(), "Dedup.wz")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	f, err := Open(logrus.StandardLogger(), path)
	if err != nil {
		t.Fatalf("open fixture: %v", err)
	}
	t.Cleanup(f.Close)

	tags = stringBlockTags(t, f, data)

	imgs := f.Root().Images()
	if len(imgs) != 1 || imgs[0].Name() != "dedup" {
		t.Fatalf("images = %+v, want one image dedup", imgs)
	}
	props, err := imgs[0].Properties()
	if err != nil {
		t.Fatalf("properties: %v", err)
	}

	sub0, ok := findDedupProp(t, props, "0").(*property.SubProperty)
	if !ok {
		t.Fatalf("0 is not a SubProperty")
	}
	sub1, ok := findDedupProp(t, props, "1").(*property.SubProperty)
	if !ok {
		t.Fatalf("1 is not a SubProperty")
	}
	return tags, sub0.Children(), sub1.Children()
}

func findDedupProp(t *testing.T, props []property.Property, name string) property.Property {
	t.Helper()
	for _, p := range props {
		if p.Name() == name {
			return p
		}
	}
	t.Fatalf("property %q not found among %d props", name, len(props))
	return nil
}

func assertDedupInt(t *testing.T, props []property.Property, name string, want int32) {
	t.Helper()
	p := findDedupProp(t, props, name)
	ip, ok := p.(*property.IntProperty)
	if !ok {
		t.Fatalf("%q: expected *property.IntProperty, got %T", name, p)
	}
	if ip.Value() != want {
		t.Errorf("%q.Value() = %d, want %d", name, ip.Value(), want)
	}
	if p.Name() != name {
		t.Errorf("%q: Name() = %q, want %q", name, p.Name(), name)
	}
}

func assertDedupVector(t *testing.T, props []property.Property, name string, wantX, wantY int32) {
	t.Helper()
	p := findDedupProp(t, props, name)
	vp, ok := p.(*property.VectorProperty)
	if !ok {
		t.Fatalf("%q: expected *property.VectorProperty, got %T", name, p)
	}
	if vp.X() != wantX || vp.Y() != wantY {
		t.Errorf("%q = (%d,%d), want (%d,%d)", name, vp.X(), vp.Y(), wantX, wantY)
	}
}

func TestBuilderStringDedupRoundTrip(t *testing.T) {
	b := wztest.NewBuilder().SetStringDedup(true).AddImage(dedupImage())
	tags, zero, one := buildAndOpenDedup(t, b)

	if !tags[0x01] {
		t.Errorf("expected at least one 0x01 (name offset-reference) string-block tag, saw tags %v", tags)
	}
	if !tags[0x1B] {
		t.Errorf("expected at least one 0x1B (type-tag offset-reference) string-block tag, saw tags %v", tags)
	}

	assertDedupInt(t, zero, "state", 1)
	assertDedupInt(t, one, "state", 2)
	assertDedupInt(t, zero, "repeat", 1)
	assertDedupInt(t, one, "repeat", 1)
	assertDedupVector(t, zero, "lt", -48, -48)
	assertDedupVector(t, one, "lt", -48, -48)
}

func TestBuilderStringDedupOffByDefault(t *testing.T) {
	b := wztest.NewBuilder().AddImage(dedupImage())
	tags, zero, one := buildAndOpenDedup(t, b)

	if tags[0x01] || tags[0x1B] {
		t.Errorf("dedup off: expected only inline (0x73) string-block tags, saw tags %v", tags)
	}
	if !tags[0x73] {
		t.Errorf("dedup off: expected inline (0x73) string-block tags, saw tags %v", tags)
	}

	assertDedupInt(t, zero, "state", 1)
	assertDedupInt(t, one, "state", 2)
	assertDedupInt(t, zero, "repeat", 1)
	assertDedupInt(t, one, "repeat", 1)
	assertDedupVector(t, zero, "lt", -48, -48)
	assertDedupVector(t, one, "lt", -48, -48)
}
