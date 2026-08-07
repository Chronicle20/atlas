package icons_test

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-wz/icons"
	"github.com/Chronicle20/atlas/libs/atlas-wz/wztest"
)

// TestNpcPrefersInfoDefault locks in the task-196 fix. Npc.wz/1101000.img
// carries its real 129x86 likeness at info/default and a 1x60 placeholder at
// stand/0; the placeholder used to win.
func TestNpcPrefersInfoDefault(t *testing.T) {
	f := openFixture(t, newArchive().
		AddImage(wztest.Img("1101000.img",
			wztest.Sub("info", wztest.Canvas("default", payloadFor(t, markDefault))),
			wztest.Sub("stand", wztest.Canvas("0", payloadFor(t, markStand))),
		)))

	img, err := icons.ExtractNpcIcon(f, 1101000)
	if err != nil {
		t.Fatalf("ExtractNpcIcon: %v", err)
	}
	if got := pixelAt(t, img); got != markDefault {
		t.Errorf("got %+v, want info/default marker %+v", got, markDefault)
	}
}

// TestNpcFallsBackToStand is the regression guard for the ~1211 NPCs that
// have no info/default at all.
func TestNpcFallsBackToStand(t *testing.T) {
	f := openFixture(t, newArchive().
		AddImage(wztest.Img("1002000.img",
			wztest.Sub("info", wztest.Int("hideName", 1)),
			wztest.Sub("stand", wztest.Canvas("0", payloadFor(t, markStand))),
		)))

	img, err := icons.ExtractNpcIcon(f, 1002000)
	if err != nil {
		t.Fatalf("ExtractNpcIcon: %v", err)
	}
	if got := pixelAt(t, img); got != markStand {
		t.Errorf("got %+v, want stand marker %+v", got, markStand)
	}
}
