package monsterbook

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

const MonsterBookCoverHandler = "MonsterBookCover"

// Cover - serverbound monster book cover request (recv 0x39). Body is a single
// int cardId the client sends to set its monster-book cover.
//
// Byte layout (IDA-verified, identical across versions — one Encode4): cardId :
// int32. The send site delegates to CUserLocal::SetMonsterBookCover (the named
// cover setter; the send site itself is unnamed/inlined) — v83 @0x95fb3e,
// v87 @0x9e2d06, v95 @0x908dd0, jms @0xa2c930. v84: the setter is unnamed in the
// IDB so the op cannot be evidence-pinned there (documented blocker).
// packet-audit:fname CUserLocal::SetMonsterBookCover
type Cover struct {
	cardId uint32
}

func NewCover(cardId uint32) Cover {
	return Cover{cardId: cardId}
}

// Encode mirrors the client's wire write (one Encode4 cardId). Serverbound
// models implement Encode so the round-trip test and packet-audit analyzer have
// a canonical serializer to diff against the client's read order.
func (c Cover) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(c.cardId)
		return w.Bytes()
	}
}

func (c *Cover) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, _ map[string]interface{}) {
		c.cardId = r.ReadUint32()
	}
}

func (c Cover) CardId() uint32    { return c.cardId }
func (c Cover) Operation() string { return MonsterBookCoverHandler }
func (c Cover) String() string    { return fmt.Sprintf("monster book cover cardId [%d]", c.cardId) }
