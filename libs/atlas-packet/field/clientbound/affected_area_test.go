package clientbound

import (
	"context"
	"encoding/binary"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	testlog "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"
)

// TestAffectedAreaCreatedWireShape proves the client RECT-buffer layout at the
// byte level (CAffectedAreaPool::OnAffectedAreaCreated read-order):
//
//	Decode4 dwId, Decode4 nType, Decode4 dwOwnerId, Decode4 nSkillID,
//	Decode1 nSLV, Decode2 phase, DecodeBuf(16) rcArea (4×int32 absolute RECT),
//	[Decode4 tStart — v95 GMS only], Decode4 tEnd.
//
//	v79/v83/v87/JMS185: 4+4+4+4+1+2+16+4 = 39 bytes (no tStart).
//	v95:                +4 for tStart    = 43 bytes.
//
// gms_v79 read order verified in CAffectedAreaPool::OnAffectedAreaCreated
// @0x42e7fc (GMS_v79_1_DEVM.exe): Decode4 dwId @0x42e82b, Decode4 nType
// @0x42e835, Decode4 dwOwnerId @0x42e83f, Decode4 nSkillID @0x42e848, Decode1
// nSLV @0x42e855, Decode2 phase @0x42e860, DecodeBuffer(16) rcArea @0x42e86b,
// Decode4 tEnd @0x42e877 — NO tStart (matches the v83/v87/JMS path, 39 bytes).
// packet-audit:verify packet=field/clientbound/FieldAffectedAreaCreated version=gms_v79 ida=0x42e7fc
func TestAffectedAreaCreatedWireShape(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	id := uuid.New()
	// origin (100,200), offsets lt(-50,-30) rb(50,30) → abs LT(50,170) RB(150,230)
	in := NewAffectedAreaCreated(id /*ownerId*/, 42 /*nType*/, 0 /*skillId*/, 2121006,
		/*skillLevel*/ 20 /*phase*/, 0 /*originX*/, 100 /*originY*/, 200,
		/*ltX*/ -50 /*ltY*/, -30 /*rbX*/, 50 /*rbY*/, 30 /*tStart*/, 0 /*tEnd*/, 10000)

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
	// v95: +4 for tStart = 43 bytes.
	b95 := in.Encode(l, pt.CreateContext("GMS", 95, 1))(nil)
	if len(b95) != 43 {
		t.Errorf("v95: got %d bytes, want 43: % x", len(b95), b95)
	}
}

// TestAffectedAreaCreatedFields verifies the exact field offsets and the absolute
// RECT computation (origin + offset) at the rcArea offset, little-endian.
func TestAffectedAreaCreatedFields(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	id := uuid.MustParse("00000000-0000-0000-0000-00000000000a")
	in := NewAffectedAreaCreated(id /*ownerId*/, 42 /*nType*/, 7 /*skillId*/, 2121006,
		/*skillLevel*/ 20 /*phase*/, 3 /*originX*/, 100 /*originY*/, 200,
		/*ltX*/ -50 /*ltY*/, -30 /*rbX*/, 50 /*rbY*/, 30 /*tStart*/, 0 /*tEnd*/, 10000)

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
	// phase at [17:19]
	require.Equal(t, int16(3), int16(binary.LittleEndian.Uint16(b[17:19])), "phase")
	// rcArea — absolute RECT at [19:35]: LT(50,170) RB(150,230)
	require.Equal(t, int32(50), int32(binary.LittleEndian.Uint32(b[19:23])), "rcArea.left")
	require.Equal(t, int32(170), int32(binary.LittleEndian.Uint32(b[23:27])), "rcArea.top")
	require.Equal(t, int32(150), int32(binary.LittleEndian.Uint32(b[27:31])), "rcArea.right")
	require.Equal(t, int32(230), int32(binary.LittleEndian.Uint32(b[31:35])), "rcArea.bottom")
	// tEnd at [35:39] (no tStart in v83)
	require.Equal(t, int32(10000), int32(binary.LittleEndian.Uint32(b[35:39])), "tEnd")

	// v95: tStart present at [35:39], tEnd at [39:43].
	b95 := in.Encode(l, pt.CreateContext("GMS", 95, 1))(nil)
	require.Len(t, b95, 43)
	require.Equal(t, int32(0), int32(binary.LittleEndian.Uint32(b95[35:39])), "tStart (v95)")
	require.Equal(t, int32(10000), int32(binary.LittleEndian.Uint32(b95[39:43])), "tEnd (v95)")
}

// TestAffectedAreaRemovedByteOutput pins the full wire body of
// CAffectedAreaPool::OnAffectedAreaRemoved (REMOVE_MIST). The client handler
// reads exactly one CInPacket::Decode4 (dwId, the mist object id) and then does
// only local rendering/cleanup — no further packet reads in any version:
//
//	v83  @0x43234d: v39 = CInPacket::Decode4(a2)        [0x43236f]
//	v87  @0x43388c: v37 = CInPacket::Decode4(a2)        [0x4338ae]
//	v95  @0x4360a0: pos = CInPacket::Decode4(iPacket)   [0x4360e1]
//	jms  @0x436eda: v40 = CInPacket::Decode4(iPacket)   [0x436efc]
//
// Wire layout is identical across all versions: dwId(4) little-endian = 4 bytes.
// Atlas encodes WriteInt(mistKey(mistId)) — a single LE uint32 — matching exactly.
//
// packet-audit:verify packet=field/clientbound/FieldAffectedAreaRemoved version=gms_v83 ida=0x43234d
// packet-audit:verify packet=field/clientbound/FieldAffectedAreaRemoved version=gms_v84 ida=0x432fb4
// packet-audit:verify packet=field/clientbound/FieldAffectedAreaRemoved version=gms_v87 ida=0x43388c
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
		{"GMS v83", "GMS", 83, 1},
		{"GMS v84", "GMS", 84, 1},
		{"GMS v87", "GMS", 87, 1},
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
