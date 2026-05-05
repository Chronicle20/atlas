package monsterbook

import (
	"context"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/sirupsen/logrus"
)

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
