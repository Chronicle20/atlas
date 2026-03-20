package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const CharacterShowChairWriter = "CharacterShowChair"

type CharacterChairShow struct {
	characterId uint32
	chairId     uint32
}

func NewCharacterChairShow(characterId uint32, chairId uint32) CharacterChairShow {
	return CharacterChairShow{characterId: characterId, chairId: chairId}
}

func (m CharacterChairShow) CharacterId() uint32 { return m.characterId }
func (m CharacterChairShow) ChairId() uint32     { return m.chairId }
func (m CharacterChairShow) Operation() string   { return CharacterShowChairWriter }
func (m CharacterChairShow) String() string {
	return fmt.Sprintf("characterId [%d], chairId [%d]", m.characterId, m.chairId)
}

func (m CharacterChairShow) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.characterId)
		w.WriteInt(m.chairId)
		return w.Bytes()
	}
}

func (m *CharacterChairShow) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.characterId = r.ReadUint32()
		m.chairId = r.ReadUint32()
	}
}
