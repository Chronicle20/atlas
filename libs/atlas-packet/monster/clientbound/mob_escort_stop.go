package clientbound

import (
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const MobEscortStopWriter = "MobEscortStop"

// MobEscortStop is the clientbound MOB_ESCORT_STOP packet
// (CMob::OnEscortStopEndPermmision): the server tells the client to clear/end an
// escort stop. The plan calls this MOB_ESCORT_RETURN_STOP; the registry op name is
// MOB_ESCORT_STOP.
//
// Byte layout (IDA-verified): EMPTY payload — the handler signature is
// `QAEXXZ` (takes NO CInPacket) and reads nothing; the wire carries only the
// opcode plus the mob oid consumed by the pool dispatcher.
//
// IDA basis: CMob::OnEscortStopEndPermmision — v95 @0x63b9c0, jms @0x6f003c
// (no CInPacket parameter; clears the mob's escort-stop fields). Dispatched from
// CMobPool::OnMobPacket as a special-cased call before the read-side switch.
// v95-only registry row; jms dispatches case 273 but carries no registry row
// (reported gap). Absent in v83/v84/v87.
type MobEscortStop struct {
}

func (m MobEscortStop) Operation() string { return MobEscortStopWriter }
func (m MobEscortStop) String() string    { return "" }

func (m MobEscortStop) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		// empty payload — handler reads no wire bytes
		return w.Bytes()
	}
}

func (m *MobEscortStop) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		// empty payload — nothing to read
	}
}
