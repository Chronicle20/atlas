package icons_test

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-wz/icons"
	"github.com/Chronicle20/atlas/libs/atlas-wz/wztest"
)

// TestMobIgnoresInfoDefault proves the fix is NPC-scoped. Mob.wz contains no
// info/default nodes today, so a mob must keep resolving through stand/0 even
// when one is present.
func TestMobIgnoresInfoDefault(t *testing.T) {
	f := openFixture(t, newArchive().
		AddImage(wztest.Img("100100.img",
			wztest.Sub("info", wztest.Canvas("default", payloadFor(t, markDefault))),
			wztest.Sub("stand", wztest.Canvas("0", payloadFor(t, markStand))),
		)))

	img, err := icons.ExtractMobIcon(f, 100100)
	if err != nil {
		t.Fatalf("ExtractMobIcon: %v", err)
	}
	if got := pixelAt(t, img); got != markStand {
		t.Errorf("got %+v, want stand marker %+v — mobs must not follow info/default", got, markStand)
	}
}

// TestNpcIgnoresTopLevelDefaultDir guards the 1209003.img shape: a top-level
// `default` imgdir holding a 14-frame animation, which is NOT info/default.
// That NPC has a healthy stand/0 and must keep it.
func TestNpcIgnoresTopLevelDefaultDir(t *testing.T) {
	f := openFixture(t, newArchive().
		AddImage(wztest.Img("1209003.img",
			wztest.Sub("info", wztest.Int("hideName", 1)),
			wztest.Sub("stand", wztest.Canvas("0", payloadFor(t, markStand))),
			wztest.Sub("default", wztest.Canvas("0", payloadFor(t, markTopLevel))),
		)))

	img, err := icons.ExtractNpcIcon(f, 1209003)
	if err != nil {
		t.Fatalf("ExtractNpcIcon: %v", err)
	}
	if got := pixelAt(t, img); got != markStand {
		t.Errorf("got %+v, want stand marker %+v — a top-level default dir is not info/default", got, markStand)
	}
}

// TestNpcInfoDefaultBeatsLink covers the 2 NPCs carrying both info/default and
// info/link: the link is a fallback, not an override.
func TestNpcInfoDefaultBeatsLink(t *testing.T) {
	f := openFixture(t, newArchive().
		AddImage(wztest.Img("1101010.img",
			wztest.Sub("info",
				wztest.Str("link", "1101011"),
				wztest.Canvas("default", payloadFor(t, markDefault)),
			),
		)).
		AddImage(wztest.Img("1101011.img",
			wztest.Sub("info", wztest.Canvas("default", payloadFor(t, markLink))),
		)))

	img, err := icons.ExtractNpcIcon(f, 1101010)
	if err != nil {
		t.Fatalf("ExtractNpcIcon: %v", err)
	}
	if got := pixelAt(t, img); got != markDefault {
		t.Errorf("got %+v, want own info/default %+v", got, markDefault)
	}
}

// TestNpcFollowsLinkToInfoDefault is the regression guard for the 33 linked
// NPCs that carry no canvas of their own. Verified working before the fix;
// must stay working after.
func TestNpcFollowsLinkToInfoDefault(t *testing.T) {
	f := openFixture(t, newArchive().
		AddImage(wztest.Img("1101020.img",
			wztest.Sub("info", wztest.Str("link", "1101021")),
		)).
		AddImage(wztest.Img("1101021.img",
			wztest.Sub("info", wztest.Canvas("default", payloadFor(t, markLink))),
		)))

	img, err := icons.ExtractNpcIcon(f, 1101020)
	if err != nil {
		t.Fatalf("ExtractNpcIcon: %v", err)
	}
	if got := pixelAt(t, img); got != markLink {
		t.Errorf("got %+v, want link-target marker %+v", got, markLink)
	}
}
