package serverbound

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
)

const CancelDebuffHandle = "CancelDebuffHandle"

// CancelDebuff - CWvsContext::CheckTemporaryStatDuration
//
// A bare "re-evaluate my temporary stats" nudge with ZERO payload. The client
// computes its locally-expired stat mask via SecondaryStat::CheckByTime and
// then discards it rather than transmitting it, on every one of the ten client
// versions examined (investigation.md §8.1). Do not grow a stat mask, a skill
// id, or a tick field onto this struct — there is nothing on the wire to read.
//
// Because the body is empty everywhere, there is nothing to version-gate. The
// 8-vs-16-byte mask split between v48 and v61+ belongs to the CLIENTBOUND
// reply and is already handled by legacyGmsMask in
// libs/atlas-packet/model/character_temporary_stat.go. (task-190 FR-2.1/2.2)
type CancelDebuff struct{}

func (m CancelDebuff) Operation() string {
	return CancelDebuffHandle
}

func (m CancelDebuff) String() string {
	return ""
}

func (m CancelDebuff) Encode(_ logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	return func(options map[string]interface{}) []byte {
		return []byte{}
	}
}

func (m *CancelDebuff) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
	}
}
