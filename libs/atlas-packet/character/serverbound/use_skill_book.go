package serverbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

const CharacterSkillBookUseHandle = "CharacterSkillBookUseHandle"

// UseSkillBook - CWvsContext::SendSkillLearnItemUseRequest
// packet-audit:fname CWvsContext::SendSkillLearnItemUseRequest
//
// Wire layout — IDB-verified byte-identical in all 10 versions
// (v48 0x70E3E7 … v95 0x9d65e0 … jms 0xAEEE61):
//
//	Encode4 updateTime
//	Encode2 slot
//	Encode4 itemId
//
// No version gate — only the per-tenant opcode differs (task-125 design §5.1).
type UseSkillBook struct {
	updateTime uint32
	slot       int16
	itemId     uint32
}

func (m UseSkillBook) UpdateTime() uint32 { return m.updateTime }
func (m UseSkillBook) Slot() int16        { return m.slot }
func (m UseSkillBook) ItemId() uint32     { return m.itemId }

func (m UseSkillBook) Operation() string {
	return CharacterSkillBookUseHandle
}

func (m UseSkillBook) String() string {
	return fmt.Sprintf("updateTime [%d], slot [%d], itemId [%d]", m.updateTime, m.slot, m.itemId)
}

func (m UseSkillBook) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.updateTime)
		w.WriteInt16(m.slot)
		w.WriteInt(m.itemId)
		return w.Bytes()
	}
}

func (m *UseSkillBook) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.updateTime = r.ReadUint32()
		m.slot = r.ReadInt16()
		m.itemId = r.ReadUint32()
	}
}
