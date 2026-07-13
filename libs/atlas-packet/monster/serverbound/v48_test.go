package serverbound

import (
	"bytes"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// TestMonsterMovementBytesV48 pins the exact v48 MOVE_LIFE (op 129) wire. The
// send site is the unnamed CMob move/action sender sub_550383 @0x550383
// (GMS_v48_1_DEVM.exe, port 13337); the COutPacket build block is @0x550868:
//
//	COutPacket(129) @0x550868
//	Encode4 @0x550888 — fused mob id (SecureFuse(this+328, this[84]))  -> uniqueId
//	Encode2 @0x5508ba — move SN counter (this[67]+1)                   -> moveId
//	Encode1 @0x5508cc — flags (v63 | 16*v64)                           -> dwFlag
//	Encode1 @0x5508d7 — (2*action)|dir (a2)                            -> nActionAndDir
//	Encode4 @0x5508e2 — skillData (a4)                                 -> skillData
//	Encode1 @0x5508f2 — moveFlags (sub_6E1C55)                         -> moveFlags
//	sub_5622DA @0x550938 — CMovePath::Flush, opaque movement payload (§5)
//
// v48 (<61) goes moveFlags -> Flush directly: it OMITS the Encode4 hackedCode
// that v61 CMob::GenerateMovePath @0x5cada5 inserts between moveFlags and Flush
// (and therefore also the v79+ flyCtx, v84+ multiTarget/randTime, v87+ CRC/chase).
// Legacy gate applied in movement.go. model.Movement is OPAQUE (§5); fixtured
// empty (StartX/StartY int16 + 0 element-count = 5 deterministic bytes).
//
// packet-audit:verify packet=monster/serverbound/MonsterMovementRequest version=gms_v48 ida=0x550383
func TestMonsterMovementBytesV48(t *testing.T) {
	p := MovementRequest{}
	p.uniqueId = 1001
	p.moveId = 55
	p.dwFlag = 1
	p.nActionAndDir = -3
	p.skillData = 0x0305
	p.moveFlags = 0
	// v61+ fields set but gated off at v48:
	p.hackedCode = 999
	p.flyCtxTargetX = 100
	p.flyCtxTargetY = 200
	p.hackedCodeCRC = 999
	p.tChaseDuration = 500

	ctx := test.CreateContext("GMS", 48, 1)
	want := []byte{
		0xE9, 0x03, 0x00, 0x00, // uniqueId 1001 (Encode4 @0x550888)
		0x37, 0x00, // moveId 55 (Encode2 @0x5508ba)
		0x01, // dwFlag 1 (Encode1 @0x5508cc)
		0xFD, // nActionAndDir -3 (Encode1 @0x5508d7)
		0x05, 0x03, 0x00, 0x00, // skillData 0x0305 (Encode4 @0x5508e2)
		0x00, // moveFlags 0 (Encode1 @0x5508f2)
		// hackedCode OMITTED (v61+); flyCtx OMITTED (v79+)
		// opaque movement (empty): StartX int16, StartY int16, count byte
		0x00, 0x00, 0x00, 0x00, 0x00,
	}
	got := test.Encode(t, ctx, p.Encode, nil)
	if !bytes.Equal(got, want) {
		t.Errorf("v48 movement bytes:\n got % x\nwant % x", got, want)
	}
}
