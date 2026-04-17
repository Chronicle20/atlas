package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-packet/model"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const CharacterBuffGiveWriter = "CharacterBuffGive"

type BuffGive struct {
	cts model.CharacterTemporaryStat
}

func NewBuffGive(cts model.CharacterTemporaryStat) BuffGive {
	return BuffGive{cts: cts}
}

func (m BuffGive) Operation() string { return CharacterBuffGiveWriter }
func (m BuffGive) String() string    { return "buff give" }

func (m BuffGive) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByteArray(m.cts.Encode(l, ctx)(options))
		w.WriteShort(0) // tDelay
		w.WriteByte(0)  // MovementAffectingStat
		return w.Bytes()
	}
}

func (m *BuffGive) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.cts = *model.NewCharacterTemporaryStat()
		m.cts.Decode(l, ctx)(r, options)
		_ = r.ReadUint16() // tDelay
		_ = r.ReadByte()   // MovementAffectingStat
	}
}

func (m BuffGive) Cts() model.CharacterTemporaryStat { return m.cts }

const CharacterBuffGiveForeignWriter = "CharacterBuffGiveForeign"

type BuffGiveForeign struct {
	characterId uint32
	cts         model.CharacterTemporaryStat
}

func NewBuffGiveForeign(characterId uint32, cts model.CharacterTemporaryStat) BuffGiveForeign {
	return BuffGiveForeign{characterId: characterId, cts: cts}
}

func (m BuffGiveForeign) CharacterId() uint32 { return m.characterId }
func (m BuffGiveForeign) Operation() string   { return CharacterBuffGiveForeignWriter }
func (m BuffGiveForeign) String() string {
	return fmt.Sprintf("characterId [%d]", m.characterId)
}

func (m BuffGiveForeign) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.characterId)
		w.WriteByteArray(m.cts.EncodeForeign(l, ctx)(options))
		w.WriteShort(0) // tDelay
		w.WriteByte(0)  // MovementAffectingStat
		return w.Bytes()
	}
}

func (m *BuffGiveForeign) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.characterId = r.ReadUint32()
		m.cts = *model.NewCharacterTemporaryStat()
		m.cts.DecodeForeign(l, ctx)(r, options)
		_ = r.ReadUint16() // tDelay
		_ = r.ReadByte()   // MovementAffectingStat
	}
}

func (m BuffGiveForeign) Cts() model.CharacterTemporaryStat { return m.cts }
