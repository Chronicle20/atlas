package clientbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

const ContiMoveWriter = "ContiMove"

// ContiMove mirrors CField_ContiMove::OnContiMove. Read order verified
// against the live IDBs and found identical in every version checked:
//
//	gms_v79 @0x5374c1, gms_v83 @0x54dca3, gms_v87 @0x577bbc,
//	gms_v95 @0x54d680 (PDB-backed names), jms_v185 @0x58e21b
//	(gms_v84 byte-identical to v83).
//
// Wire layout: Decode1(state) selects one of six arms via (state-7):
//
//	7  -> nullsub, no further read
//	8  -> OnStartShipMoveField: Decode1(subState); ==2 => CShip::LeaveShipMove
//	9  -> nullsub, no further read
//	10 -> OnMoveField: Decode1(subState); ==4 => CShip::AppearShip, ==5 => CShip::DisappearShip
//	11 -> nullsub, no further read
//	12 -> OnEndShipMoveField: Decode1(subState); ==6 => CShip::EnterShipMove
//
// The prior atlas codec wrote/read a single unconditional state byte, silently
// dropping the subState byte the client always reads for states 8/10/12 — a
// false pass (the verify markers asserted the encoder's own one-byte output,
// never the true wire body). Corrected here across all versions (task-181).
//
// packet-audit:fname CField_ContiMove::OnContiMove
type ContiMove struct {
	state    byte
	subState byte
}

func NewContiMove(state byte, subState byte) ContiMove {
	return ContiMove{state: state, subState: subState}
}

func (m ContiMove) State() byte    { return m.state }
func (m ContiMove) SubState() byte { return m.subState }

func (m ContiMove) Operation() string { return ContiMoveWriter }
func (m ContiMove) String() string {
	return fmt.Sprintf("state [%d] subState [%d]", m.state, m.subState)
}

// contiMoveHasSubState reports whether OnContiMove reads a second byte for
// the given state. Arms 8/10/12 (OnStartShipMoveField/OnMoveField/
// OnEndShipMoveField) always Decode1 a subState byte regardless of its
// value; arms 7/9/11 are nullsubs that read nothing further.
func contiMoveHasSubState(state byte) bool {
	switch state {
	case 8, 10, 12:
		return true
	default:
		return false
	}
}

func (m ContiMove) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.state)
		if contiMoveHasSubState(m.state) {
			w.WriteByte(m.subState)
		}
		return w.Bytes()
	}
}

func (m *ContiMove) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.state = r.ReadByte()
		if contiMoveHasSubState(m.state) {
			m.subState = r.ReadByte()
		}
	}
}
