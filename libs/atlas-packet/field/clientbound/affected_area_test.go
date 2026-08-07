package clientbound

import (
	"context"
	"encoding/binary"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	testlog "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// TestAffectedAreaCreatedWireShape proves the client RECT-buffer layout at the
// byte level (CAffectedAreaPool::OnAffectedAreaCreated read-order):
//
//	Decode4 dwId, Decode4 nType, Decode4 dwOwnerId, Decode4 nSkillID,
//	Decode1 nSLV, Decode2 phase, DecodeBuf(16) rcArea (4×int32 absolute RECT),
//	[Decode4 tStart — v95 GMS only], Decode4 tEnd.
//
//	v79/v83/v87/JMS185: 4+4+4+4+1+2+16+4 = 39 bytes (no nPhase).
//	v95:                +4 for nPhase    = 43 bytes.
//
// gms_v79 read order verified in CAffectedAreaPool::OnAffectedAreaCreated
// @0x42e7fc (GMS_v79_1_DEVM.exe): Decode4 dwId @0x42e82b, Decode4 nType
// @0x42e835, Decode4 dwOwnerId @0x42e83f, Decode4 nSkillID @0x42e848, Decode1
// nSLV @0x42e855, Decode2 skillDelay @0x42e860, DecodeBuffer(16) rcArea
// @0x42e86b, Decode4 nElemAttr @0x42e877 — NO nPhase (matches the v83/v87/JMS
// path, 39 bytes).
func TestAffectedAreaCreatedWireShape(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	id := uuid.New()
	// origin (100,200), offsets lt(-50,-30) rb(50,30) → abs LT(50,170) RB(150,230)
	in := NewAffectedAreaCreated(id /*ownerId*/, 42 /*nType*/, 0 /*skillId*/, 2121006,
		/*skillLevel*/ 20 /*skillDelay*/, 0 /*originX*/, 100 /*originY*/, 200,
		/*ltX*/ -50 /*ltY*/, -30 /*rbX*/, 50 /*rbY*/, 30 /*elemAttr*/, 0 /*phase*/, 10000)

	for _, v := range []struct {
		Name, Region string
		Major, Minor uint16
	}{
		{"GMS v79", "GMS", 79, 1}, {"GMS v83", "GMS", 83, 1}, {"GMS v87", "GMS", 87, 1}, {"JMS v185", "JMS", 185, 1},
	} {
		b := in.Encode(l, pt.CreateContext(v.Region, v.Major, v.Minor))(nil)
		if len(b) != 39 {
			t.Errorf("%s: got %d bytes, want 39: % x", v.Name, len(b), b)
		}
	}
	// v95: +4 for nPhase = 43 bytes.
	b95 := in.Encode(l, pt.CreateContext("GMS", 95, 1))(nil)
	if len(b95) != 43 {
		t.Errorf("v95: got %d bytes, want 43: % x", len(b95), b95)
	}
}

// TestAffectedAreaCreatedByteOutput pins the full wire body of
// CAffectedAreaPool::OnAffectedAreaCreated (SPAWN_MIST) against the client
// read order per version (docs/packets/ida-exports/):
//
//	Decode4 dwId, Decode4 nType, Decode4 dwOwnerId, Decode4 nSkillID,
//	Decode1 nSLV, Decode2 skillDelay, DecodeBuf(16) rcArea (4×int32 absolute
//	LTRB), Decode4 nElemAttr, [Decode4 nPhase — gms_v92 and gms_v95].
//
// Atlas encodes rcArea as origin+offset absolute LTRB (4×WriteInt32) matching
// the client's single DecodeBuf(16). Wire body: 39 bytes (43 on gms_v92/gms_v95,
// 33 on gms_v48, 28 on gms_v12).
// The nPhase gate is IsRegion("GMS") && MajorAtLeast(92), so JMS185 does NOT
// carry it. gms_v92
// CAffectedAreaPool::OnAffectedAreaCreated @0x4392a0 reads two trailing Decode4s
// (@0x43932e, @0x439339), which is why the floor is 92 and not 95.
//
// gms_v48 (task-165 Tier B): CAffectedAreaPool::OnPacket @0x42182f dispatches
// op 0xCA -> OnAffectedAreaCreated @0x421854. That handler narrows nType to
// Decode1 (@0x421881) and nElemAttr (+0x30) to Decode1 (@0x4218c4), so the
// v48 body is 33 bytes, not 39. Read order taken from disassembly (SEH function).
//
// packet-audit:verify packet=field/clientbound/FieldAffectedAreaCreated version=gms_v48 ida=0x421854
// packet-audit:verify packet=field/clientbound/FieldAffectedAreaCreated version=gms_v61 ida=0x423edc
// packet-audit:verify packet=field/clientbound/FieldAffectedAreaCreated version=gms_v72 ida=0x42e36c
// packet-audit:verify packet=field/clientbound/FieldAffectedAreaCreated version=gms_v79 ida=0x42e7fc
// packet-audit:verify packet=field/clientbound/FieldAffectedAreaCreated version=gms_v83 ida=0x431a63
// packet-audit:verify packet=field/clientbound/FieldAffectedAreaCreated version=gms_v84 ida=0x4326ca
// packet-audit:verify packet=field/clientbound/FieldAffectedAreaCreated version=gms_v87 ida=0x432f3f
// packet-audit:verify packet=field/clientbound/FieldAffectedAreaCreated version=gms_v92 ida=0x4392a0
// packet-audit:verify packet=field/clientbound/FieldAffectedAreaCreated version=gms_v95 ida=0x437ec0
// packet-audit:verify packet=field/clientbound/FieldAffectedAreaCreated version=jms_v185 ida=0x436572
func TestAffectedAreaCreatedByteOutput(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	// mistId chosen so uuid.ID() (time_low, first 4 UUID bytes big-endian) is a
	// known value: bytes 00 01 02 03 → 0x00010203 (same trick as the Removed test).
	mistId := uuid.MustParse("00010203-0000-0000-0000-000000000000")
	wantKey := mistKey(mistId) // == 0x00010203

	// Common prefix: dwId nType dwOwnerId nSkillID nSLV skillDelay rcArea (abs LTRB).
	prefix := []byte{
		byte(wantKey), byte(wantKey >> 8), byte(wantKey >> 16), byte(wantKey >> 24), // dwId LE
		0x07, 0x00, 0x00, 0x00, // nType = 7
		0x2A, 0x00, 0x00, 0x00, // dwOwnerId = 42
		0x2E, 0x5D, 0x20, 0x00, // nSkillID = 2121006
		0x14,       // nSLV = 20
		0x03, 0x00, // skillDelay = 3
		0x32, 0x00, 0x00, 0x00, // rcArea.left   = 100-50  = 50
		0xAA, 0x00, 0x00, 0x00, // rcArea.top    = 200-30  = 170
		0x96, 0x00, 0x00, 0x00, // rcArea.right  = 100+50  = 150
		0xE6, 0x00, 0x00, 0x00, // rcArea.bottom = 200+30  = 230
	}
	elemAttr := []byte{0xD2, 0x04, 0x00, 0x00} // nElemAttr = 1234 (all versions >= v48)
	nPhase := []byte{0x10, 0x27, 0x00, 0x00}   // nPhase = 10000 (gms_v92/gms_v95 only)

	// gms_v48 narrows nType to one byte and nElemAttr (+0x30) to one byte
	// (CAffectedAreaPool::OnAffectedAreaCreated @0x421854, disasm @0x421881 /
	// @0x4218c4). 4+1+4+4+1+2+16+1 = 33 bytes.
	prefixV48 := []byte{
		byte(wantKey), byte(wantKey >> 8), byte(wantKey >> 16), byte(wantKey >> 24), // dwId LE
		0x07,                   // nType = 7 (Decode1)
		0x2A, 0x00, 0x00, 0x00, // dwOwnerId = 42
		0x2E, 0x5D, 0x20, 0x00, // nSkillID = 2121006
		0x14,       // nSLV = 20
		0x03, 0x00, // skillDelay = 3
		0x32, 0x00, 0x00, 0x00, // rcArea.left   = 50
		0xAA, 0x00, 0x00, 0x00, // rcArea.top    = 170
		0x96, 0x00, 0x00, 0x00, // rcArea.right  = 150
		0xE6, 0x00, 0x00, 0x00, // rcArea.bottom = 230
		0xD2, // nElemAttr = low byte of 1234 (1234 & 0xFF)
	}
	// gms_v12 drops dwOwnerId entirely and reads no nElemAttr
	// (CAffectedAreaPool::OnAffectedAreaCreated @0x4166f5). 4+1+4+1+2+16 = 28 bytes.
	prefixV12 := []byte{
		byte(wantKey), byte(wantKey >> 8), byte(wantKey >> 16), byte(wantKey >> 24), // dwId LE
		0x07,                   // nType = 7 (Decode1)
		0x2E, 0x5D, 0x20, 0x00, // nSkillID = 2121006
		0x14,       // nSLV = 20
		0x03, 0x00, // skillDelay = 3
		0x32, 0x00, 0x00, 0x00, // rcArea.left   = 50
		0xAA, 0x00, 0x00, 0x00, // rcArea.top    = 170
		0x96, 0x00, 0x00, 0x00, // rcArea.right  = 150
		0xE6, 0x00, 0x00, 0x00, // rcArea.bottom = 230
	}

	for _, v := range []struct {
		Name, Region string
		Major, Minor uint16
		HasPhase     bool
		Want         []byte
	}{
		{"GMS v12", "GMS", 12, 1, false, prefixV12},
		{"GMS v48", "GMS", 48, 1, false, prefixV48},
		{"GMS v61", "GMS", 61, 1, false, nil},
		{"GMS v72", "GMS", 72, 1, false, nil},
		{"GMS v79", "GMS", 79, 1, false, nil},
		{"GMS v83", "GMS", 83, 1, false, nil},
		{"GMS v84", "GMS", 84, 1, false, nil},
		{"GMS v87", "GMS", 87, 1, false, nil},
		{"GMS v92", "GMS", 92, 1, true, nil},
		{"GMS v95", "GMS", 95, 1, true, nil},
		{"JMS v185", "JMS", 185, 1, false, nil},
	} {
		t.Run(v.Name, func(t *testing.T) {
			want := v.Want
			if want == nil {
				want = append([]byte{}, prefix...)
				want = append(want, elemAttr...)
				if v.HasPhase {
					want = append(want, nPhase...)
				}
			}

			in := NewAffectedAreaCreated(mistId /*ownerId*/, 42 /*nType*/, 7,
				/*skillId*/ 2121006 /*skillLevel*/, 20 /*skillDelay*/, 3,
				/*originX*/ 100 /*originY*/, 200,
				/*ltX*/ -50 /*ltY*/, -30 /*rbX*/, 50 /*rbY*/, 30,
				/*elemAttr*/ 1234 /*phase*/, 10000)
			got := in.Encode(l, pt.CreateContext(v.Region, v.Major, v.Minor))(nil)
			require.Equal(t, want, got, "%s wire bytes", v.Name)
		})
	}
}

// TestAffectedAreaCreatedFields verifies the exact field offsets and the absolute
// RECT computation (origin + offset) at the rcArea offset, little-endian.
func TestAffectedAreaCreatedFields(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	id := uuid.MustParse("00000000-0000-0000-0000-00000000000a")
	in := NewAffectedAreaCreated(id /*ownerId*/, 42 /*nType*/, 7 /*skillId*/, 2121006,
		/*skillLevel*/ 20 /*skillDelay*/, 3 /*originX*/, 100 /*originY*/, 200,
		/*ltX*/ -50 /*ltY*/, -30 /*rbX*/, 50 /*rbY*/, 30 /*elemAttr*/, 10000 /*phase*/, 0)

	b := in.Encode(l, pt.CreateContext("GMS", 83, 1))(nil)
	require.Len(t, b, 39)

	// dwId at [0:4]
	require.Equal(t, mistKey(id), binary.LittleEndian.Uint32(b[0:4]), "dwId")
	// nType at [4:8]
	require.Equal(t, int32(7), int32(binary.LittleEndian.Uint32(b[4:8])), "nType")
	// dwOwnerId at [8:12]
	require.Equal(t, uint32(42), binary.LittleEndian.Uint32(b[8:12]), "dwOwnerId")
	// nSkillID at [12:16]
	require.Equal(t, int32(2121006), int32(binary.LittleEndian.Uint32(b[12:16])), "nSkillID")
	// nSLV at [16]
	require.Equal(t, byte(20), b[16], "nSLV")
	// skillDelay at [17:19]
	require.Equal(t, int16(3), int16(binary.LittleEndian.Uint16(b[17:19])), "skillDelay")
	// rcArea — absolute RECT at [19:35]: LT(50,170) RB(150,230)
	require.Equal(t, int32(50), int32(binary.LittleEndian.Uint32(b[19:23])), "rcArea.left")
	require.Equal(t, int32(170), int32(binary.LittleEndian.Uint32(b[23:27])), "rcArea.top")
	require.Equal(t, int32(150), int32(binary.LittleEndian.Uint32(b[27:31])), "rcArea.right")
	require.Equal(t, int32(230), int32(binary.LittleEndian.Uint32(b[31:35])), "rcArea.bottom")
	// nElemAttr at [35:39] (no nPhase in v83)
	require.Equal(t, int32(10000), int32(binary.LittleEndian.Uint32(b[35:39])), "nElemAttr")

	// v95: nElemAttr at [35:39], nPhase at [39:43].
	b95 := in.Encode(l, pt.CreateContext("GMS", 95, 1))(nil)
	require.Len(t, b95, 43)
	require.Equal(t, int32(10000), int32(binary.LittleEndian.Uint32(b95[35:39])), "nElemAttr (v95)")
	require.Equal(t, int32(0), int32(binary.LittleEndian.Uint32(b95[39:43])), "nPhase (v95)")
}

// TestAffectedAreaRemovedByteOutput pins the full wire body of
// CAffectedAreaPool::OnAffectedAreaRemoved (REMOVE_MIST). The client handler
// reads exactly one CInPacket::Decode4 (dwId, the mist object id) and then does
// only local rendering/cleanup — no further packet reads in any version:
//
//	v61  @0x4246b0: CInPacket::Decode4
//	v72  @0x42ec4e: CInPacket::Decode4
//	v79  @0x42f0de: CInPacket::Decode4
//	v83  @0x43234d: v39 = CInPacket::Decode4(a2)        [0x43236f]
//	v87  @0x43388c: v37 = CInPacket::Decode4(a2)        [0x4338ae]
//	v95  @0x4360a0: pos = CInPacket::Decode4(iPacket)   [0x4360e1]
//	jms  @0x436eda: v40 = CInPacket::Decode4(iPacket)   [0x436efc]
//
// Wire layout is identical across all versions: dwId(4) little-endian = 4 bytes.
// Atlas encodes WriteInt(mistKey(mistId)) — a single LE uint32 — matching exactly.
//
//	v12  @0x416cc4: sub_416B99 (Decode4)               [0x416cdf]
//	v48  @0x421e7f: CInPacket::Decode4                 [0x421e9a]
//	v92  @0x4371f0: CInPacket::Decode4                 [0x43722a]
//
// packet-audit:verify packet=field/clientbound/FieldAffectedAreaRemoved version=gms_v48 ida=0x421e7f
// packet-audit:verify packet=field/clientbound/FieldAffectedAreaRemoved version=gms_v61 ida=0x4246b0
// packet-audit:verify packet=field/clientbound/FieldAffectedAreaRemoved version=gms_v72 ida=0x42ec4e
// packet-audit:verify packet=field/clientbound/FieldAffectedAreaRemoved version=gms_v79 ida=0x42f0de
// packet-audit:verify packet=field/clientbound/FieldAffectedAreaRemoved version=gms_v83 ida=0x43234d
// packet-audit:verify packet=field/clientbound/FieldAffectedAreaRemoved version=gms_v84 ida=0x432fb4
// packet-audit:verify packet=field/clientbound/FieldAffectedAreaRemoved version=gms_v87 ida=0x43388c
// packet-audit:verify packet=field/clientbound/FieldAffectedAreaRemoved version=gms_v92 ida=0x4371f0
// packet-audit:verify packet=field/clientbound/FieldAffectedAreaRemoved version=gms_v95 ida=0x4360a0
// packet-audit:verify packet=field/clientbound/FieldAffectedAreaRemoved version=jms_v185 ida=0x436eda
func TestAffectedAreaRemovedByteOutput(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	// mistId chosen so its uuid.ID() (time_low = first 4 bytes of the UUID,
	// big-endian) is a known value: bytes 00 01 02 03 → 0x00010203.
	mistId := uuid.MustParse("00010203-0000-0000-0000-000000000000")
	wantKey := mistKey(mistId) // == 0x00010203
	want := []byte{
		byte(wantKey), byte(wantKey >> 8), byte(wantKey >> 16), byte(wantKey >> 24), // dwId LE uint32
	}

	for _, v := range []struct {
		Name, Region string
		Major, Minor uint16
	}{
		{"GMS v12", "GMS", 12, 1},
		{"GMS v48", "GMS", 48, 1},
		{"GMS v61", "GMS", 61, 1},
		{"GMS v72", "GMS", 72, 1},
		{"GMS v79", "GMS", 79, 1},
		{"GMS v83", "GMS", 83, 1},
		{"GMS v84", "GMS", 84, 1},
		{"GMS v87", "GMS", 87, 1},
		{"GMS v92", "GMS", 92, 1},
		{"GMS v95", "GMS", 95, 1},
		{"JMS v185", "JMS", 185, 1},
	} {
		t.Run(v.Name, func(t *testing.T) {
			in := NewAffectedAreaRemoved(mistId, 0xCAFE)
			got := in.Encode(l, pt.CreateContext(v.Region, v.Major, v.Minor))(nil)
			require.Equal(t, want, got, "%s wire bytes", v.Name)
			require.Len(t, got, 4, "%s body is a single uint32", v.Name)
		})
	}
}

func TestAffectedAreaRemoved_EncodeShape(t *testing.T) {
	mistId := uuid.MustParse("00000000-0000-0000-0000-00000000000b")
	w := NewAffectedAreaRemoved(mistId, 0xCAFE)

	require.Equal(t, AffectedAreaRemovedWriter, w.Operation())

	enc := w.Encode(logrus.New(), context.Background())
	require.NotNil(t, enc)

	out := enc(map[string]interface{}{})
	require.NotEmpty(t, out)
}
