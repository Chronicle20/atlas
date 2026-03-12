package character

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-packet/model"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const CharacterBuffCancelWriter = "CharacterBuffCancel"

type BuffCancel struct {
	cts model.CharacterTemporaryStat
}

func NewBuffCancel(cts model.CharacterTemporaryStat) BuffCancel {
	return BuffCancel{cts: cts}
}

func (m BuffCancel) Operation() string { return CharacterBuffCancelWriter }
func (m BuffCancel) String() string    { return "buff cancel" }

func (m BuffCancel) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	t := tenant.MustFromContext(ctx)
	return func(options map[string]interface{}) []byte {
		m.cts.EncodeMask(l, t, options)(w)
		w.WriteByte(0) // tSwallowBuffTime
		return w.Bytes()
	}
}

func (m *BuffCancel) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.cts = *model.NewCharacterTemporaryStat()
		_ = m.cts.DecodeMask(r)
		_ = r.ReadByte() // tSwallowBuffTime
	}
}

const CharacterBuffCancelForeignWriter = "CharacterBuffCancelForeign"

type BuffCancelForeign struct {
	characterId uint32
	cts         model.CharacterTemporaryStat
}

func NewBuffCancelForeign(characterId uint32, cts model.CharacterTemporaryStat) BuffCancelForeign {
	return BuffCancelForeign{characterId: characterId, cts: cts}
}

func (m BuffCancelForeign) CharacterId() uint32 { return m.characterId }
func (m BuffCancelForeign) Operation() string   { return CharacterBuffCancelForeignWriter }
func (m BuffCancelForeign) String() string {
	return fmt.Sprintf("characterId [%d]", m.characterId)
}

func (m BuffCancelForeign) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	t := tenant.MustFromContext(ctx)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.characterId)
		m.cts.EncodeMask(l, t, options)(w)
		w.WriteByte(0) // tSwallowBuffTime
		return w.Bytes()
	}
}

func (m *BuffCancelForeign) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.characterId = r.ReadUint32()
		m.cts = *model.NewCharacterTemporaryStat()
		_ = m.cts.DecodeMask(r)
		_ = r.ReadByte() // tSwallowBuffTime
	}
}
