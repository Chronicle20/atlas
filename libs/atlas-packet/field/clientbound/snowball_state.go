package clientbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

const SnowballStateWriter = "SnowballState"

// SnowballState mirrors CField_SnowBall::OnSnowBallState. Read order verified
// against the live IDBs and found identical in every version checked:
//
//	gms_v79 @0x5525bf, gms_v83 @0x5750a3, gms_v87 @0x5a3328,
//	gms_v95 @0x560ab0 (PDB-backed names), jms_v185 @0x5c959d
//	(gms_v84 @0x584a1c byte-identical to v83).
//
// Wire layout:
//
//	Decode1  state           -> m_nState (bFirst = previous m_nState == -1)
//	Decode4  leftSnowmanHp   -> m_aSnowMan[0].m_nHP
//	Decode4  rightSnowmanHp  -> m_aSnowMan[1].m_nHP
//	2x { Decode2 x; Decode1 y } -> CSnowBall::SetPos for m_aSnowBall[0..1]
//	if bFirst: Decode2 damageSnowBall, Decode2 damageSnowMan0, Decode2 damageSnowMan1
//
// `first` is not carried on the wire — the client gates the trailing three
// shorts on its own stored state (previous state == -1). The server sets it when
// transmitting the initial snapshot; Decode recovers it from the trailing bytes
// being present.
//
// packet-audit:fname CField_SnowBall::OnSnowBallState
type SnowballState struct {
	state          byte
	leftSnowmanHp  uint32
	rightSnowmanHp uint32
	snowball0X     uint16
	snowball0Y     byte
	snowball1X     uint16
	snowball1Y     byte
	first          bool
	damageSnowBall uint16
	damageSnowMan0 uint16
	damageSnowMan1 uint16
}

func NewSnowballState(state byte, leftSnowmanHp uint32, rightSnowmanHp uint32, snowball0X uint16, snowball0Y byte, snowball1X uint16, snowball1Y byte, first bool, damageSnowBall uint16, damageSnowMan0 uint16, damageSnowMan1 uint16) SnowballState {
	return SnowballState{
		state:          state,
		leftSnowmanHp:  leftSnowmanHp,
		rightSnowmanHp: rightSnowmanHp,
		snowball0X:     snowball0X,
		snowball0Y:     snowball0Y,
		snowball1X:     snowball1X,
		snowball1Y:     snowball1Y,
		first:          first,
		damageSnowBall: damageSnowBall,
		damageSnowMan0: damageSnowMan0,
		damageSnowMan1: damageSnowMan1,
	}
}

func (m SnowballState) State() byte            { return m.state }
func (m SnowballState) LeftSnowmanHp() uint32  { return m.leftSnowmanHp }
func (m SnowballState) RightSnowmanHp() uint32 { return m.rightSnowmanHp }
func (m SnowballState) Snowball0X() uint16     { return m.snowball0X }
func (m SnowballState) Snowball0Y() byte       { return m.snowball0Y }
func (m SnowballState) Snowball1X() uint16     { return m.snowball1X }
func (m SnowballState) Snowball1Y() byte       { return m.snowball1Y }
func (m SnowballState) First() bool            { return m.first }
func (m SnowballState) DamageSnowBall() uint16 { return m.damageSnowBall }
func (m SnowballState) DamageSnowMan0() uint16 { return m.damageSnowMan0 }
func (m SnowballState) DamageSnowMan1() uint16 { return m.damageSnowMan1 }

func (m SnowballState) Operation() string { return SnowballStateWriter }

func (m SnowballState) String() string {
	return fmt.Sprintf("state [%d] leftSnowmanHp [%d] rightSnowmanHp [%d] snowball0 [%d,%d] snowball1 [%d,%d] first [%t] damage [%d,%d,%d]",
		m.state, m.leftSnowmanHp, m.rightSnowmanHp, m.snowball0X, m.snowball0Y, m.snowball1X, m.snowball1Y, m.first, m.damageSnowBall, m.damageSnowMan0, m.damageSnowMan1)
}

func (m SnowballState) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.state)
		w.WriteInt(m.leftSnowmanHp)
		w.WriteInt(m.rightSnowmanHp)
		w.WriteShort(m.snowball0X)
		w.WriteByte(m.snowball0Y)
		w.WriteShort(m.snowball1X)
		w.WriteByte(m.snowball1Y)
		if m.first {
			w.WriteShort(m.damageSnowBall)
			w.WriteShort(m.damageSnowMan0)
			w.WriteShort(m.damageSnowMan1)
		}
		return w.Bytes()
	}
}

func (m *SnowballState) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.state = r.ReadByte()
		m.leftSnowmanHp = r.ReadUint32()
		m.rightSnowmanHp = r.ReadUint32()
		m.snowball0X = r.ReadUint16()
		m.snowball0Y = r.ReadByte()
		m.snowball1X = r.ReadUint16()
		m.snowball1Y = r.ReadByte()
		// The initial snapshot appends three damage shorts; the client gates
		// these on its own prior state, so recover `first` from their presence.
		if r.Available() >= 6 {
			m.first = true
			m.damageSnowBall = r.ReadUint16()
			m.damageSnowMan0 = r.ReadUint16()
			m.damageSnowMan1 = r.ReadUint16()
		}
	}
}
