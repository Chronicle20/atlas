package clientbound

import (
	"encoding/binary"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	testlog "github.com/sirupsen/logrus/hooks/test"
)

func TestUiLock(t *testing.T) {
	input := NewUiLock(true, 5000)
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			pt.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}

// TestUiLockWireShape proves the exact wire layout matches what
// CUserLocal::OnSetDirectionMode (GMS v95 @ 0x9054f0) reads:
//
//	Decode1 (enable flag) + Decode4 (tAfterLeaveDirectionMode) = 5 bytes for GMS v90+.
//
// The second field was introduced for GMS v90+; older/non-GMS versions omit it.
//
// v87 evidence (GMSv87_4GB.exe, md5 2e692f3a…): CUserLocal::SetDirectionMode
// @ 0x9e312a reads ONLY ONE byte (Decode1 enable flag); the int32
// tAfterLeaveDirectionMode is absent — v87 mirrors v83. The >=90 gate therefore
// emits 1 byte for v87 → CONFIRMED CORRECT (task-080 B4.1).
func TestUiLockWireShape(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	const tAfter = int32(5000)
	in := NewUiLock(true, tAfter)

	// Explicit v87 assertion: gate (>=90) is below the v87 boundary → 1-byte
	// (enable-only) narrow form, matching CUserLocal::SetDirectionMode @ 0x9e312a.
	if v87 := in.Encode(l, pt.CreateContext("GMS", 87, 1))(nil); len(v87) != 1 || v87[0] != 0x01 {
		t.Errorf("v87 ui-Lock packet = % x (%d bytes), want [01] (1 byte, enable-only)", v87, len(v87))
	}

	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			b := in.Encode(l, ctx)(nil)

			isGMSv90Plus := v.Region == "GMS" && v.MajorVersion >= 90
			if isGMSv90Plus {
				// 1 (flag) + 4 (tAfterLeaveDirectionMode) = 5 bytes
				if len(b) != 5 {
					t.Errorf("wire size = %d bytes, want 5: % x", len(b), b)
				}
				if b[0] != 0x01 {
					t.Errorf("byte[0] = 0x%02x, want 0x01 (enable)", b[0])
				}
				got := int32(binary.LittleEndian.Uint32(b[1:5]))
				if got != tAfter {
					t.Errorf("tAfterLeaveDirectionMode = %d, want %d", got, tAfter)
				}
			} else {
				// 1 (flag only) = 1 byte for older / non-GMS
				if len(b) != 1 {
					t.Errorf("wire size = %d bytes, want 1: % x", len(b), b)
				}
				if b[0] != 0x01 {
					t.Errorf("byte[0] = 0x%02x, want 0x01 (enable)", b[0])
				}
			}
		})
	}
}
