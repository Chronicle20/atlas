package handler

import (
	"context"
	"testing"

	mbsb "github.com/Chronicle20/atlas/libs/atlas-packet/character/serverbound/monsterbook"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/sirupsen/logrus"
)

// TestMonsterBookCoverDecode pins the wire format that
// MonsterBookCoverHandleFunc consumes: a single little-endian uint32 cardId.
//
// A full handler-emission test would require a real session, tenant context,
// and a live Kafka producer; that is out of scope for a unit test. Instead,
// we exercise the same Decode path the handler invokes so that any breaking
// change to the serverbound packet shape surfaces here.
func TestMonsterBookCoverDecode(t *testing.T) {
	// little-endian uint32 for cardId 2380000 (0x002450E0).
	raw := []byte{0xE0, 0x50, 0x24, 0x00}
	req := request.Request(raw)
	reader := request.NewRequestReader(&req, 0)

	p := mbsb.Cover{}
	p.Decode(logrus.New(), context.Background())(&reader, map[string]interface{}{})

	if p.CardId() != 2380000 {
		t.Fatalf("expected cardId 2380000, got %d", p.CardId())
	}
	if p.Operation() != mbsb.MonsterBookCoverHandler {
		t.Fatalf("expected Operation %q, got %q", mbsb.MonsterBookCoverHandler, p.Operation())
	}
}

// TestMonsterBookCoverHandleFuncSymbol verifies the handler can be obtained
// (it must accept a writer.Producer and return a session-aware handler).
func TestMonsterBookCoverHandleFuncSymbol(t *testing.T) {
	// Just calling the constructor with nil writer.Producer must produce a
	// non-nil handler closure; we don't invoke it (would require session/tenant).
	got := MonsterBookCoverHandleFunc(logrus.New(), context.Background(), nil)
	if got == nil {
		t.Fatal("MonsterBookCoverHandleFunc returned nil closure")
	}
}
