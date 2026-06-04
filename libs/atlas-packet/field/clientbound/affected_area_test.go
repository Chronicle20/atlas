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
//	v83/v87/JMS185: 4+4+4+4+1+2+16+4 = 39 bytes (no tStart).
//	v95:            +4 for tStart    = 43 bytes.
func TestAffectedAreaCreatedWireShape(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	id := uuid.New()
	// origin (100,200), offsets lt(-50,-30) rb(50,30) → abs LT(50,170) RB(150,230)
	in := NewAffectedAreaCreated(id, /*ownerId*/ 42, /*nType*/ 0, /*skillId*/ 2121006,
		/*skillLevel*/ 20, /*phase*/ 0, /*originX*/ 100, /*originY*/ 200,
		/*ltX*/ -50, /*ltY*/ -30, /*rbX*/ 50, /*rbY*/ 30, /*tStart*/ 0, /*tEnd*/ 10000)

	for _, v := range []struct {
		Name, Region string
		Major, Minor uint16
	}{
		{"GMS v83", "GMS", 83, 1}, {"GMS v87", "GMS", 87, 1}, {"JMS v185", "JMS", 185, 1},
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
	in := NewAffectedAreaCreated(id, /*ownerId*/ 42, /*nType*/ 7, /*skillId*/ 2121006,
		/*skillLevel*/ 20, /*phase*/ 3, /*originX*/ 100, /*originY*/ 200,
		/*ltX*/ -50, /*ltY*/ -30, /*rbX*/ 50, /*rbY*/ 30, /*tStart*/ 0, /*tEnd*/ 10000)

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

func TestAffectedAreaRemoved_EncodeShape(t *testing.T) {
	mistId := uuid.MustParse("00000000-0000-0000-0000-00000000000b")
	w := NewAffectedAreaRemoved(mistId, 0xCAFE)

	require.Equal(t, AffectedAreaRemovedWriter, w.Operation())

	enc := w.Encode(logrus.New(), context.Background())
	require.NotNil(t, enc)

	out := enc(map[string]interface{}{})
	require.NotEmpty(t, out)
}
