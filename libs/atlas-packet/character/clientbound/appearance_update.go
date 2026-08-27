package clientbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-packet/model"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

const CharacterAppearanceUpdateWriter = "CharacterAppearanceUpdate"

type CharacterAppearanceUpdate struct {
	characterId uint32
	avatar      model.Avatar
	rings       model.RingSet
}

func NewCharacterAppearanceUpdate(characterId uint32, avatar model.Avatar, rings model.RingSet) CharacterAppearanceUpdate {
	return CharacterAppearanceUpdate{characterId: characterId, avatar: avatar, rings: rings}
}

func (m CharacterAppearanceUpdate) CharacterId() uint32 { return m.characterId }
func (m CharacterAppearanceUpdate) Operation() string   { return CharacterAppearanceUpdateWriter }
func (m CharacterAppearanceUpdate) String() string {
	return fmt.Sprintf("characterId [%d]", m.characterId)
}

func (m CharacterAppearanceUpdate) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	t := tenant.MustFromContext(ctx)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.characterId)
		// flags byte: bit0=avatarLook, bit1=speed, bit2=carryItem
		// (CUserRemote::OnAvatarModified, gms_v83.json @0x98367e). Atlas always
		// writes bit0 only, so the client's &2/&4 reads never fire.
		w.WriteByte(1)
		w.WriteByteArray(m.avatar.Encode(l, ctx)(options))
		m.rings.EncodeField(w, t)
		return w.Bytes()
	}
}

func (m *CharacterAppearanceUpdate) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	return func(r *request.Reader, options map[string]interface{}) {
		m.characterId = r.ReadUint32()
		_ = r.ReadByte() // flags: bit0=avatarLook, bit1=speed, bit2=carryItem
		m.avatar.Decode(l, ctx)(r, options)
		m.rings.DecodeField(r, t)
	}
}
