package charparts

import (
	"image"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-wz/wz"
	"github.com/Chronicle20/atlas/libs/atlas-wz/wz/property"
)

// TestEncodeDecodePartNameRoundTrip locks in the dotted form that lets the
// caller re-attach stance/frame to manifest sprites after atlas.Pack reorders
// inputs. Frames > 9 must round-trip exactly so e.g. "stand1.10.body" still
// parses; part names that themselves contain a "." (rare but possible in
// future WZ data) must keep the suffix intact.
func TestEncodeDecodePartNameRoundTrip(t *testing.T) {
	cases := []struct {
		stance string
		frame  int
		part   string
	}{
		{"stand1", 0, "body"},
		{"walk1", 3, "armOverHair"},
		{"default", 0, "head"},
		{"stand2", 10, "arm"},
	}
	for _, c := range cases {
		name := EncodePartName(c.stance, c.frame, c.part)
		gs, gf, gp, ok := DecodePartName(name)
		if !ok {
			t.Fatalf("decode failed for %q", name)
		}
		if gs != c.stance || gf != c.frame || gp != c.part {
			t.Fatalf("round trip mismatch for %q: got (%q, %d, %q)", name, gs, gf, gp)
		}
	}
}

func TestDecodePartNameRejectsMalformed(t *testing.T) {
	cases := []string{"", "body", "stand1.body", "stand1..body", "stand1.x.body"}
	for _, c := range cases {
		if _, _, _, ok := DecodePartName(c); ok {
			t.Errorf("expected decode to fail for %q", c)
		}
	}
}

func TestNormalizeIdStripsLeadingZerosAndImgSuffix(t *testing.T) {
	cases := map[string]string{
		"00002000":     "2000",
		"00012000.img": "12000",
		"000":          "0",
		"":             "0",
	}
	for in, want := range cases {
		if got := normalizeId(in); got != want {
			t.Errorf("normalizeId(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestCanonicalizeUOLPathHandlesDotDot(t *testing.T) {
	cases := []struct {
		anchor string
		value  string
		want   string
	}{
		{"stand1/0", "../front/head", "stand1/front/head"},
		{"stand1/0", "head", "stand1/0/head"},
		{"", "front/head", "front/head"},
		{"a/b/c", "../../x", "a/x"},
	}
	for _, c := range cases {
		if got := canonicalizeUOLPath(c.anchor, c.value); got != c.want {
			t.Errorf("canonicalizeUOLPath(%q, %q) = %q, want %q", c.anchor, c.value, got, c.want)
		}
	}
}

func TestAccessoryPartClassFor(t *testing.T) {
	cases := map[uint32]struct {
		want string
		ok   bool
	}{
		1010000: {"FaceAccessory", true},
		1020000: {"EyeAccessory", true},
		1030000: {"Earrings", true},
		1040000: {"", false},
	}
	for id, c := range cases {
		got, ok := accessoryPartClassFor(id)
		if ok != c.ok || got != c.want {
			t.Errorf("accessoryPartClassFor(%d) = (%q, %v), want (%q, %v)", id, got, ok, c.want, c.ok)
		}
	}
}

func TestExtractInfoBlock(t *testing.T) {
	props := []property.Property{
		property.NewSub("info", []property.Property{
			property.NewString("islot", "Hp"),
			property.NewString("vslot", "CpHnHd"),
			property.NewInt("cash", 0),
		}),
	}
	info := extractInfoBlock(props)
	if info.Islot != "Hp" || info.Vslot != "CpHnHd" || info.Cash != 0 {
		t.Fatalf("unexpected info: %+v", info)
	}
}

// TestExtractPartMetadataParsesAnchors ensures the donor's map sub-property is
// projected into the Anchors field used downstream by atlas.Pack.
func TestExtractPartMetadataParsesAnchors(t *testing.T) {
	children := []property.Property{
		property.NewVector("origin", 8, 32),
		property.NewString("z", "weaponOverGlove"),
		property.NewSub("map", []property.Property{
			property.NewVector("neck", 8, 0),
			property.NewVector("navel", 8, 16),
		}),
	}
	origin, anchors, z := extractPartMetadata(children)
	if origin != (image.Point{X: 8, Y: 32}) {
		t.Errorf("origin = %+v", origin)
	}
	if z != "weaponOverGlove" {
		t.Errorf("z = %q, want %q", z, "weaponOverGlove")
	}
	if anchors["neck"] != (image.Point{X: 8, Y: 0}) {
		t.Errorf("neck anchor = %+v", anchors["neck"])
	}
	if anchors["navel"] != (image.Point{X: 8, Y: 16}) {
		t.Errorf("navel anchor = %+v", anchors["navel"])
	}
}

// TestWalkCharacterNilFile guards against caller misuse without forcing them
// to construct a real archive just to check the error path.
func TestWalkCharacterNilFile(t *testing.T) {
	if _, err := WalkCharacter(nil, nil); err == nil {
		t.Fatal("expected error for nil wz.File")
	}
}

// TestWalkCharacterEmptyRoot returns an empty result rather than panicking.
func TestWalkCharacterEmptyRoot(t *testing.T) {
	f := wz.NewFileWithRoot("Character", wz.NewDirectory("Character", nil, nil))
	sets, err := WalkCharacter(f, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(sets) != 0 {
		t.Fatalf("want 0 sets, got %d", len(sets))
	}
}

// TestWalkCharacterRoutesPartClassByDirectory verifies the partClass mapping
// by constructing an in-memory tree with placeholder .img files in each
// recognized subdirectory. The .img files have no canvas data, so the
// extracted PartSets will have empty Sprites; the test asserts only on the
// (PartClass, ID) routing.
func TestWalkCharacterRoutesPartClassByDirectory(t *testing.T) {
	// Subdir + img name -> expected partClass mapping.
	tree := []struct {
		dir, img, wantPC string
	}{
		{"Cap", "00001002.img", "Cap"},
		{"Coat", "01040000.img", "Coat"},
		{"Longcoat", "01050000.img", "Longcoat"},
		{"Pants", "01060000.img", "Pants"},
		{"Shoes", "01070000.img", "Shoes"},
		{"Glove", "01080000.img", "Glove"},
		{"Cape", "01100000.img", "Cape"},
		{"Shield", "01090000.img", "Shield"},
		{"Weapon", "01302000.img", "Weapon"},
		{"Hair", "00030000.img", "Hair"},
		{"Face", "00020000.img", "Face"},
		// Accessory expands per-id range:
		{"Accessory", "01010000.img", "FaceAccessory"},
		{"Accessory", "01020000.img", "EyeAccessory"},
		{"Accessory", "01030000.img", "Earrings"},
	}

	dirs := map[string][]*wz.Image{}
	for _, c := range tree {
		dirs[c.dir] = append(dirs[c.dir], wz.NewParsedImage(c.img, nil))
	}
	subdirs := make([]*wz.Directory, 0, len(dirs))
	for name, imgs := range dirs {
		subdirs = append(subdirs, wz.NewDirectory(name, nil, imgs))
	}

	// Plus a couple of root-level body skin imgs.
	rootImgs := []*wz.Image{
		wz.NewParsedImage("00002000.img", nil), // body skin 2000
		wz.NewParsedImage("00012000.img", nil), // head 12000
	}

	root := wz.NewDirectory("Character", subdirs, rootImgs)
	f := wz.NewFileWithRoot("Character", root)

	sets, err := WalkCharacter(f, nil)
	if err != nil {
		t.Fatal(err)
	}

	type key struct {
		pc string
		id uint32
	}
	got := map[key]bool{}
	for _, s := range sets {
		got[key{s.PartClass, s.ID}] = true
	}

	wantKeys := []key{
		{"Body", 2000}, {"Body", 12000},
		{"Cap", 1002},
		{"Coat", 1040000},
		{"Longcoat", 1050000},
		{"Pants", 1060000},
		{"Shoes", 1070000},
		{"Glove", 1080000},
		{"Cape", 1100000},
		{"Shield", 1090000},
		{"Weapon", 1302000},
		{"Hair", 30000},
		{"Face", 20000},
		{"FaceAccessory", 1010000},
		{"EyeAccessory", 1020000},
		{"Earrings", 1030000},
	}
	for _, k := range wantKeys {
		if !got[k] {
			t.Errorf("missing PartSet %s/%d", k.pc, k.id)
		}
	}
}

// TestWalkCharacterFilterRespectsAllowList: requesting only a subset must
// suppress every other partClass.
func TestWalkCharacterFilterRespectsAllowList(t *testing.T) {
	root := wz.NewDirectory("Character", []*wz.Directory{
		wz.NewDirectory("Hair", nil, []*wz.Image{wz.NewParsedImage("00030000.img", nil)}),
		wz.NewDirectory("Face", nil, []*wz.Image{wz.NewParsedImage("00020000.img", nil)}),
	}, []*wz.Image{wz.NewParsedImage("00002000.img", nil)})

	f := wz.NewFileWithRoot("Character", root)
	sets, err := WalkCharacter(f, map[string]bool{"Hair": true})
	if err != nil {
		t.Fatal(err)
	}
	if len(sets) != 1 {
		t.Fatalf("want 1 set, got %d", len(sets))
	}
	if sets[0].PartClass != "Hair" || sets[0].ID != 30000 {
		t.Errorf("unexpected set: %+v", sets[0])
	}
}

// TestToAtlasInputsSkipsNilImages validates that PartSets with no decoded
// pixel data are dropped (so atlas.Pack never sees a nil Image).
func TestToAtlasInputsSkipsNilImages(t *testing.T) {
	set := PartSet{
		PartClass: "Coat",
		ID:        1040000,
		Sprites: []SpriteInput{
			{Stance: "stand1", Frame: 0, Part: "arm", Img: nil},
			{Stance: "stand1", Frame: 0, Part: "body", Img: image.NewNRGBA(image.Rect(0, 0, 4, 4))},
		},
	}
	in := ToAtlasInputs(set)
	if len(in) != 1 {
		t.Fatalf("want 1 input, got %d", len(in))
	}
	if in[0].Name != "stand1.0.body" {
		t.Errorf("unexpected name: %q", in[0].Name)
	}
}
