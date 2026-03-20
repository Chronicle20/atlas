package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-packet/model"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const CharacterAppearanceUpdateWriter = "CharacterAppearanceUpdate"

type CharacterAppearanceUpdate struct {
	characterId uint32
	avatar      model.Avatar
}

func NewCharacterAppearanceUpdate(characterId uint32, avatar model.Avatar) CharacterAppearanceUpdate {
	return CharacterAppearanceUpdate{characterId: characterId, avatar: avatar}
}

func (m CharacterAppearanceUpdate) CharacterId() uint32 { return m.characterId }
func (m CharacterAppearanceUpdate) Operation() string   { return CharacterAppearanceUpdateWriter }
func (m CharacterAppearanceUpdate) String() string {
	return fmt.Sprintf("characterId [%d]", m.characterId)
}

func (m CharacterAppearanceUpdate) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.characterId)
		w.WriteByte(1) // mode
		w.WriteByteArray(m.avatar.Encode(l, ctx)(options))
		w.WriteByte(0) // crush ring
		w.WriteByte(0) // friendship ring
		w.WriteByte(0) // marriage ring
		w.WriteInt(0)  // completed set item id
		return w.Bytes()
	}
}

func (m *CharacterAppearanceUpdate) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.characterId = r.ReadUint32()
		_ = r.ReadByte() // mode
		m.avatar.Decode(l, ctx)(r, options)
		_ = r.ReadByte()   // crush ring
		_ = r.ReadByte()   // friendship ring
		_ = r.ReadByte()   // marriage ring
		_ = r.ReadUint32() // completed set item id
	}
}
