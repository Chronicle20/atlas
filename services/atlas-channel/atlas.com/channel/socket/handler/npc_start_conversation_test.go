package handler

import (
	"atlas-channel/session"
	"testing"

	"github.com/sirupsen/logrus"

	objectid "github.com/Chronicle20/atlas/libs/atlas-object-id"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
)

// startConversationPayload encodes StartConversation's GMS v83 wire layout:
// uint32 oid, int16 x, int16 y (startConversationHasXY is true from v72 on).
func startConversationPayload(oid uint32, x int16, y int16) []byte {
	return []byte{
		byte(oid), byte(oid >> 8), byte(oid >> 16), byte(oid >> 24),
		byte(x), byte(x >> 8),
		byte(y), byte(y >> 8),
	}
}

// TestNPCStartConversation_PlayerNpcOidIsNoOp is the regression test for
// task-251 bug report §3: a click on a Player NPC (client-visible oid in
// [objectid.PlayerNpcObjectIdBase, objectid.MinId), never present in
// atlas-data's per-map NPC list) must be a no-op (PRD FR-7.4), not the
// anti-cheat session Destroy the handler used to run when
// GetInMapByObjectId inevitably failed to find it.
func TestNPCStartConversation_PlayerNpcOidIsNoOp(t *testing.T) {
	const charId = uint32(701)
	oid := objectid.PlayerNpcObjectIdBase + 1000 // deployed Player NPC oid

	s, ctx, cleanup := newCashItemUseTestSession(t, charId)
	defer cleanup()

	req := request.Request(startConversationPayload(oid, 100, -50))
	reader := request.NewRequestReader(&req, 0)
	NPCStartConversationHandleFunc(logrus.New(), ctx, nil)(s, &reader, map[string]interface{}{})

	if _, err := session.NewProcessor(logrus.New(), ctx).GetByCharacterId(s.Field().Channel())(charId); err != nil {
		t.Errorf("session for character [%d] was destroyed on a Player NPC click: %v", charId, err)
	}
}
