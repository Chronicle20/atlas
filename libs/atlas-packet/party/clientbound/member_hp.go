package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const PartyMemberHPWriter = "PartyMemberHP"

type MemberHP struct {
	characterId uint32
	hp          uint16
	maxHp       uint16
}

func NewPartyMemberHP(characterId uint32, hp uint16, maxHp uint16) MemberHP {
	return MemberHP{characterId: characterId, hp: hp, maxHp: maxHp}
}

func (m MemberHP) Operation() string { return PartyMemberHPWriter }
func (m MemberHP) String() string {
	return fmt.Sprintf("characterId [%d], hp [%d], maxHp [%d]", m.characterId, m.hp, m.maxHp)
}

func (m MemberHP) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.characterId)
		w.WriteInt(uint32(m.hp))
		w.WriteInt(uint32(m.maxHp))
		return w.Bytes()
	}
}

func (m *MemberHP) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.characterId = r.ReadUint32()
		m.hp = uint16(r.ReadUint32())
		m.maxHp = uint16(r.ReadUint32())
	}
}
