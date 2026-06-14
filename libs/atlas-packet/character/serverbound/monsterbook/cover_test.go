package monsterbook

import (
	"context"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/sirupsen/logrus"
)

// packet-audit:verify packet=character/serverbound/monsterbook/CharacterCover version=gms_v83 ida=0x95fb3e
// packet-audit:verify packet=character/serverbound/monsterbook/CharacterCover version=gms_v84 ida=0x99e8ee
// packet-audit:verify packet=character/serverbound/monsterbook/CharacterCover version=gms_v87 ida=0x9e2d06
// packet-audit:verify packet=character/serverbound/monsterbook/CharacterCover version=gms_v95 ida=0x908dd0
// packet-audit:verify packet=character/serverbound/monsterbook/CharacterCover version=jms_v185 ida=0xa2c930
//
// MONSTER_BOOK_COVER (serverbound): the client sends a single int cover cardId,
// consumed by CUserLocal::SetMonsterBookCover (the named cover setter; the send
// site is unnamed/inlined in every IDB). Layout = one Decode4, byte-identical
// across versions. task-092 Stage 4 located + named CUserLocal::SetMonsterBookCover
// in the v84 IDB (@0x99e8ee, local setter writing CharacterData+1523) so the v84
// cell now pins like the others.
//
// TestCoverDecode verifies the serverbound MonsterBookCover (recv 0x39) decoder
// reads a single little-endian uint32 cardId off the wire.
func TestCoverDecode(t *testing.T) {
	// 4 bytes little-endian for cardId 2380000 (0x002450E0).
	raw := []byte{0xE0, 0x50, 0x24, 0x00}
	req := request.Request(raw)
	reader := request.NewRequestReader(&req, 0)

	c := Cover{}
	c.Decode(logrus.New(), context.Background())(&reader, map[string]interface{}{})

	if c.CardId() != 2380000 {
		t.Fatalf("expected cardId 2380000, got %d", c.CardId())
	}
}

func TestCoverOperation(t *testing.T) {
	c := Cover{}
	if c.Operation() != MonsterBookCoverHandler {
		t.Fatalf("expected Operation %q, got %q", MonsterBookCoverHandler, c.Operation())
	}
}

// TestCoverEncode pins the wire format the client emits: one little-endian
// uint32 cardId (CUserLocal::SetMonsterBookCover delegate; one Encode4).
func TestCoverEncode(t *testing.T) {
	out := NewCover(2380000).Encode(logrus.New(), context.Background())(map[string]interface{}{})
	want := []byte{0xE0, 0x50, 0x24, 0x00} // 2380000 = 0x002450E0 LE
	if len(out) != 4 || out[0] != want[0] || out[1] != want[1] || out[2] != want[2] || out[3] != want[3] {
		t.Fatalf("Cover encode mismatch\n got % x\nwant % x", out, want)
	}
}
