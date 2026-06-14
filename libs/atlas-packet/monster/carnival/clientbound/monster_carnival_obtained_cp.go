package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const MonsterCarnivalObtainedCPWriter = "MonsterCarnivalObtainedCP"

// MonsterCarnivalObtainedCP is the clientbound MONSTER_CARNIVAL_OBTAINED_CP packet
// (CField_MonsterCarnival::OnPersonalCP): the server updates the local player's
// personal CP counter on the carnival scoreboard.
//
// Byte layout (IDA-verified, identical across all 5 versions — two Decode2):
//   - cp    : uint16 — Decode2; SetPersonalCP arg1 (current personal CP)
//   - total : uint16 — Decode2; SetPersonalCP arg2 (total personal CP earned)
//
// IDA basis: CField_MonsterCarnival::OnPersonalCP — v83 @0x56550e, v84 @0x572215,
// v87 @0x590294, v95 @0x55a2a0, jms @0x5b02c3: `v2=Decode2; v3=Decode2;
// SetPersonalCP(v2, v3)`. All five identical.
type MonsterCarnivalObtainedCP struct {
	cp    uint16
	total uint16
}

func NewMonsterCarnivalObtainedCP(cp uint16, total uint16) MonsterCarnivalObtainedCP {
	return MonsterCarnivalObtainedCP{cp: cp, total: total}
}

func (m MonsterCarnivalObtainedCP) Cp() uint16        { return m.cp }
func (m MonsterCarnivalObtainedCP) Total() uint16     { return m.total }
func (m MonsterCarnivalObtainedCP) Operation() string { return MonsterCarnivalObtainedCPWriter }
func (m MonsterCarnivalObtainedCP) String() string {
	return fmt.Sprintf("cp [%d], total [%d]", m.cp, m.total)
}

func (m MonsterCarnivalObtainedCP) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteShort(m.cp)
		w.WriteShort(m.total)
		return w.Bytes()
	}
}

func (m *MonsterCarnivalObtainedCP) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.cp = r.ReadUint16()
		m.total = r.ReadUint16()
	}
}
