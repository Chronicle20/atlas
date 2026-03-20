package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-packet/model"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const MonsterStatSetWriter = "MonsterStatSet"
const MonsterStatResetWriter = "MonsterStatReset"

type StatSet struct {
	uniqueId uint32
	stat     *model.MonsterTemporaryStat
}

func NewMonsterStatSet(uniqueId uint32, stat *model.MonsterTemporaryStat) StatSet {
	return StatSet{uniqueId: uniqueId, stat: stat}
}

func (m StatSet) UniqueId() uint32                       { return m.uniqueId }
func (m StatSet) Stat() *model.MonsterTemporaryStat      { return m.stat }
func (m StatSet) Operation() string                      { return MonsterStatSetWriter }
func (m StatSet) String() string {
	return fmt.Sprintf("uniqueId [%d]", m.uniqueId)
}

func (m StatSet) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	t := tenant.MustFromContext(ctx)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.uniqueId)
		w.WriteByteArray(m.stat.Encode(l, ctx)(options))
		w.WriteInt16(0) // tDelay
		w.WriteByte(0)  // m_nCalcDamageStatIndex
		if m.stat.IsMovementAffectingStat(t) {
			w.WriteByte(0) // bStat
		}
		return w.Bytes()
	}
}

func (m *StatSet) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	return func(r *request.Reader, options map[string]interface{}) {
		m.uniqueId = r.ReadUint32()
		m.stat = model.NewMonsterTemporaryStat()
		m.stat.Decode(l, ctx)(r, options)
		_ = r.ReadInt16() // tDelay
		_ = r.ReadByte()  // m_nCalcDamageStatIndex
		if m.stat.IsMovementAffectingStat(t) {
			_ = r.ReadByte() // bStat
		}
	}
}

type StatReset struct {
	uniqueId uint32
	stat     *model.MonsterTemporaryStat
}

func NewMonsterStatReset(uniqueId uint32, stat *model.MonsterTemporaryStat) StatReset {
	return StatReset{uniqueId: uniqueId, stat: stat}
}

func (m StatReset) UniqueId() uint32                       { return m.uniqueId }
func (m StatReset) Stat() *model.MonsterTemporaryStat      { return m.stat }
func (m StatReset) Operation() string                      { return MonsterStatResetWriter }
func (m StatReset) String() string {
	return fmt.Sprintf("uniqueId [%d]", m.uniqueId)
}

func (m StatReset) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	t := tenant.MustFromContext(ctx)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.uniqueId)
		w.WriteByteArray(m.stat.Encode(l, ctx)(options))
		w.WriteInt16(0) // tDelay
		w.WriteByte(0)  // m_nCalcDamageStatIndex
		if m.stat.IsMovementAffectingStat(t) {
			w.WriteByte(0) // bStat
		}
		return w.Bytes()
	}
}

func (m *StatReset) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	return func(r *request.Reader, options map[string]interface{}) {
		m.uniqueId = r.ReadUint32()
		m.stat = model.NewMonsterTemporaryStat()
		m.stat.Decode(l, ctx)(r, options)
		_ = r.ReadInt16() // tDelay
		_ = r.ReadByte()  // m_nCalcDamageStatIndex
		if m.stat.IsMovementAffectingStat(t) {
			_ = r.ReadByte() // bStat
		}
	}
}
