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
		// Reset mask: names ONLY the stats being cancelled. Not EncodeSetMask —
		// that one asserts the TwoState/base bits unconditionally, which on a
		// reset would tell the client to tear down RideVehicle and GuidedBullet
		// on every buff expiry (task-190).
		m.cts.EncodeMask(l, t, options)(w)
		// Trailing byte: nSecondaryStatChangedPoint, NOT tSwallowBuffTime. The
		// client reads it only when the mask contains a movement-affecting stat
		// (SecondaryStat::IsMovementAffectingStat — Speed/Jump/Stun/Weakness/
		// Slow/Morph/Ghost/BasicStatUp/Attract), then feeds it to
		// CMovePath::SetStatChangedPoint. Writing it unconditionally is safe:
		// when the client does not read it, it is trailing slack the client
		// ignores; it must never be omitted, or a movement-affecting reset
		// (any mob disease) would read one byte past the end.
		w.WriteByte(0)
		return w.Bytes()
	}
}

func (m *BuffCancel) Decode(_ logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	return func(r *request.Reader, options map[string]interface{}) {
		m.cts = *model.NewCharacterTemporaryStat()
		_ = m.cts.DecodeMask(r, t)
		_ = r.ReadByte() // nSecondaryStatChangedPoint
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
		// Reset mask — same reasoning as BuffCancel.Encode above. The foreign
		// reset feeds CUserRemote::OnResetTemporaryStat, which drives the other
		// players' view of this character's mount; an unconditional RideVehicle
		// bit desyncs their render of it just as it desyncs the owner's.
		m.cts.EncodeMask(l, t, options)(w)
		w.WriteByte(0) // nSecondaryStatChangedPoint — see BuffCancel.Encode
		return w.Bytes()
	}
}

func (m *BuffCancelForeign) Decode(_ logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	return func(r *request.Reader, options map[string]interface{}) {
		m.characterId = r.ReadUint32()
		m.cts = *model.NewCharacterTemporaryStat()
		_ = m.cts.DecodeMask(r, t)
		_ = r.ReadByte() // nSecondaryStatChangedPoint
	}
}
