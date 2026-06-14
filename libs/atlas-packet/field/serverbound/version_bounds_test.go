package serverbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// TestGeneralChatVersionBoundary pins the corrected >83 -> >=87 boundary for
// the general-chat updateTime (delta §3.1.10): v84..86 must encode
// byte-identically to v83 (no leading updateTime int). v87/v95 stay on the
// later-GMS path.
func TestGeneralChatVersionBoundary(t *testing.T) {
	m := General{updateTime: 0xDEADBEEF, msg: "hello world", bOnlyBalloon: true}
	encode := func(major uint16) []byte {
		ctx := pt.CreateContext("GMS", major, 1)
		return pt.Encode(t, ctx, m.Encode, nil)
	}
	v83 := encode(83)
	for _, major := range []uint16{84, 85, 86} {
		if got := encode(major); !bytes.Equal(got, v83) {
			t.Errorf("General chat v%d encode differs from v83 (len %d vs %d); v84..86 must match v83", major, len(got), len(v83))
		}
	}
	if v87 := encode(87); bytes.Equal(v87, v83) {
		t.Errorf("General chat v87 must stay on the later-GMS path, not equal v83")
	}
	if v95 := encode(95); bytes.Equal(v95, v83) {
		t.Errorf("General chat v95 must stay on the later-GMS path, not equal v83")
	}
}
