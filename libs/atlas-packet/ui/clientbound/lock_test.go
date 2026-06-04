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
func TestUiLockWireShape(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	const tAfter = int32(5000)
	in := NewUiLock(true, tAfter)

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
