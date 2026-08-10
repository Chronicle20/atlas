package clientbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

const CharacterShowUpgradeTombEffectWriter = "CharacterShowUpgradeTombEffect"

// ShowUpgradeTombEffect - CUserRemote::OnShowUpgradeTombEffect
// packet-audit:fname CUserRemote::OnShowUpgradeTombEffect
//
// Broadcast to the OTHER sessions in the map when a dead player's client asks
// for the Wheel of Destiny tomb effect (USE_DEATHITEM). The requesting client
// already plays its own copy locally, so the owner is excluded.
//
// Wire layout — identical on every version that carries the op:
//
//	Encode4  characterId  — consumed by CUserPool::OnUserRemotePacket before the switch
//	Encode4  itemId
//	Encode4  x
//	Encode4  y
//
// IDA gms_v95 @0x954090 (dispatcher @0x94b390 case 221 = 0x0DD), gms_v92
// @0x9307e0, gms_v87 @0xa098f2, gms_v84 @0x9c4206, gms_v83 @0x983e40, gms_v79
// @0x8d9fe6 (dispatcher @0x8c8d4a case 181), gms_v72 @0x88d0e4 (dispatcher
// @0x87c046 case 177), jms_v185 @0xa57a4e. No version divergence.
type ShowUpgradeTombEffect struct {
	characterId uint32
	itemId      uint32
	x           int32
	y           int32
}

func NewShowUpgradeTombEffect(characterId uint32, itemId uint32, x int32, y int32) ShowUpgradeTombEffect {
	return ShowUpgradeTombEffect{characterId: characterId, itemId: itemId, x: x, y: y}
}

func (m ShowUpgradeTombEffect) CharacterId() uint32 { return m.characterId }
func (m ShowUpgradeTombEffect) ItemId() uint32      { return m.itemId }
func (m ShowUpgradeTombEffect) X() int32            { return m.x }
func (m ShowUpgradeTombEffect) Y() int32            { return m.y }
func (m ShowUpgradeTombEffect) Operation() string   { return CharacterShowUpgradeTombEffectWriter }
func (m ShowUpgradeTombEffect) String() string {
	return fmt.Sprintf("characterId [%d], itemId [%d], x [%d], y [%d]", m.characterId, m.itemId, m.x, m.y)
}

func (m ShowUpgradeTombEffect) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.characterId)
		w.WriteInt(m.itemId)
		w.WriteInt32(m.x)
		w.WriteInt32(m.y)
		return w.Bytes()
	}
}

func (m *ShowUpgradeTombEffect) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.characterId = r.ReadUint32()
		m.itemId = r.ReadUint32()
		m.x = r.ReadInt32()
		m.y = r.ReadInt32()
	}
}
