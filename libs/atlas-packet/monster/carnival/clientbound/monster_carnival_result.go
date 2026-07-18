package clientbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

const MonsterCarnivalResultWriter = "MonsterCarnivalResult"

// MonsterCarnivalResult is the clientbound MONSTER_CARNIVAL_RESULT packet
// (CField_MonsterCarnival::OnShowGameResult): when a carnival match ends the
// server tells the client the outcome; the client shows the matching end-of-game
// message and waits to be warped out.
//
// Byte layout (IDA-verified, identical across all 5 versions — a single Decode1):
//   - result : byte — Decode1; outcome selector (8=win, 9=lose, 10=draw,
//     11=opponent left). Maps to a StringPool message; no further bytes.
//
// IDA basis: CField_MonsterCarnival::OnShowGameResult — v83 @0x565add, v84 @0x5727e4,
// v87 @0x59085e, v95 @0x55af80, jms @0x5b088a: `v2 = Decode1(); switch(v2-8){ ...
// StringPool win/lose/draw/abrupt-end ... }` — exactly one wire byte.
type MonsterCarnivalResult struct {
	result byte
}

func NewMonsterCarnivalResult(result byte) MonsterCarnivalResult {
	return MonsterCarnivalResult{result: result}
}

func (m MonsterCarnivalResult) Result() byte      { return m.result }
func (m MonsterCarnivalResult) Operation() string { return MonsterCarnivalResultWriter }
func (m MonsterCarnivalResult) String() string {
	return fmt.Sprintf("result [%d]", m.result)
}

func (m MonsterCarnivalResult) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.result)
		return w.Bytes()
	}
}

func (m *MonsterCarnivalResult) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.result = r.ReadByte()
	}
}
