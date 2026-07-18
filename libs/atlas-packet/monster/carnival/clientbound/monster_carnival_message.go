package clientbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

const MonsterCarnivalMessageWriter = "MonsterCarnivalMessage"

// MonsterCarnivalMessage is the clientbound MONSTER_CARNIVAL_MESSAGE packet — the
// MESSAGE mode of CField_MonsterCarnival::OnRequestResult (dispatched when the
// dispatcher's sub-op arg == 0). The server tells the client to display a
// pre-canned carnival status line; the displayed text comes from the client's
// StringPool keyed by the selector, NOT from the packet.
//
// Byte layout (IDA-verified, identical across all 5 versions — a single Decode1):
//   - message : byte — Decode1; the message selector (1..6). The switch maps each
//     value to a StringPool entry (e.g. "not enough CP", "already
//     summoned", "unknown error"); no further bytes are read.
//
// This is a DISTINCT wire shape from the SUMMON mode (2x Decode1 + DecodeStr) of
// the same dispatcher function.
//
// IDA basis: CField_MonsterCarnival::OnRequestResult — v83 @0x56557d, v84 @0x572284,
// v87 @0x590303, v95 @0x55a890, jms @0x5b0332. The `bResult == 0` (MESSAGE) branch:
// `v8 = Decode1(); switch(v8){ ... StringPool::GetString(...) ... }` — exactly one
// wire byte; the strings are local (SP_4082.. / 0x101B..).
type MonsterCarnivalMessage struct {
	message byte
}

func NewMonsterCarnivalMessage(message byte) MonsterCarnivalMessage {
	return MonsterCarnivalMessage{message: message}
}

func (m MonsterCarnivalMessage) Message() byte     { return m.message }
func (m MonsterCarnivalMessage) Operation() string { return MonsterCarnivalMessageWriter }
func (m MonsterCarnivalMessage) String() string {
	return fmt.Sprintf("message [%d]", m.message)
}

func (m MonsterCarnivalMessage) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.message)
		return w.Bytes()
	}
}

func (m *MonsterCarnivalMessage) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.message = r.ReadByte()
	}
}
