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
		m.cts.EncodeCancelMask(l, t, options)(w)
		// The trailing byte is the movement flag, read by the client only
		// when the cancel mask intersects the version's movement filter
		// (previously mislabelled tSwallowBuffTime and written
		// unconditionally off the give-shape EncodeMask — task-167 F1).
		// design.md §5.5.3.
		if !m.cts.CancelMask(t).And(model.MovementAffectingMask(t)).IsZero() {
			w.WriteByte(0) // movement flag
		}
		return w.Bytes()
	}
}

func (m *BuffCancel) Decode(_ logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	return func(r *request.Reader, options map[string]interface{}) {
		m.cts = *model.NewCharacterTemporaryStat()
		mask := m.cts.DecodeMask(r, t)
		if !mask.And(model.MovementAffectingMask(t)).IsZero() {
			_ = r.ReadByte() // movement flag
		}
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
		m.cts.EncodeCancelMask(l, t, options)(w)
		// The trailing byte is the movement flag — see BuffCancel.Encode.
		if !m.cts.CancelMask(t).And(model.MovementAffectingMask(t)).IsZero() {
			w.WriteByte(0) // movement flag
		}
		return w.Bytes()
	}
}

func (m *BuffCancelForeign) Decode(_ logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	return func(r *request.Reader, options map[string]interface{}) {
		m.characterId = r.ReadUint32()
		m.cts = *model.NewCharacterTemporaryStat()
		mask := m.cts.DecodeMask(r, t)
		if !mask.And(model.MovementAffectingMask(t)).IsZero() {
			_ = r.ReadByte() // movement flag
		}
	}
}
