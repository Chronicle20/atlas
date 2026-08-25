package writer

import (
	"bytes"
	"testing"

	"github.com/sirupsen/logrus/hooks/test"
)

// destroyMonsterTestOptions is the six-entry operations table task-253's
// fix-dom25 brief adds to the DestroyMonster writer in every seed template --
// the codes are version-invariant (design §2.2).
var destroyMonsterTestOptions = map[string]interface{}{
	"operations": map[string]interface{}{
		"DISAPPEAR":        float64(0),
		"FADE_OUT":         float64(1),
		"BOMB":             float64(2),
		"DESTRUCT_BY_MISS": float64(3),
		"SWALLOW":          float64(4),
		"SELF_DESTRUCT":    float64(5),
	},
}

// TestDestroyMonsterBodyResolvesCode is the load-bearing wire-identity check
// (DOM-25 fix brief): the resolved byte for an ordinary death (FADE_OUT) and
// for a self-destruct (SELF_DESTRUCT) must match the pre-task-253 hardcoded
// dead-type verbatim, so the wire stays byte-identical.
func TestDestroyMonsterBodyResolvesCode(t *testing.T) {
	tests := []struct {
		name string
		code DestroyMonsterCode
		want []byte
	}{
		{"ordinary death fades out", DestroyMonsterFadeOut, []byte{0xD2, 0x04, 0x00, 0x00, 0x01}},
		{"self-destruct", DestroyMonsterSelfDestruct, []byte{0xD2, 0x04, 0x00, 0x00, 0x05}},
		{"bomb", DestroyMonsterBomb, []byte{0xD2, 0x04, 0x00, 0x00, 0x02}},
		{"destruct by miss", DestroyMonsterDestructByMiss, []byte{0xD2, 0x04, 0x00, 0x00, 0x03}},
		{"swallow", DestroyMonsterSwallow, []byte{0xD2, 0x04, 0x00, 0x00, 0x04}},
		{"disappear", DestroyMonsterDisappear, []byte{0xD2, 0x04, 0x00, 0x00, 0x00}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l, _ := test.NewNullLogger()
			actual := DestroyMonsterBody(1234, tt.code)(l, reportTestContext(t))(destroyMonsterTestOptions)
			if !bytes.Equal(actual, tt.want) {
				t.Errorf("got %v want %v", actual, tt.want)
			}
		})
	}
}
