package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-packet/teleportrock"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

// ItemUseTeleportRock - the teleport-rock branch of
// CWvsContext::SendConsumeCashItemUseRequest, after the common ItemUse prefix:
// shared RunMapTransferItem target payload, then trailing int updateTime on
// all versions (design task-124 §1 Q1).
type ItemUseTeleportRock struct {
	updateTimeFirst bool
	target          teleportrock.Target
	updateTime      uint32
}

func NewItemUseTeleportRock(updateTimeFirst bool) *ItemUseTeleportRock {
	return &ItemUseTeleportRock{updateTimeFirst: updateTimeFirst}
}

func (m ItemUseTeleportRock) Target() teleportrock.Target { return m.target }
func (m ItemUseTeleportRock) UpdateTime() uint32          { return m.updateTime }

func (m ItemUseTeleportRock) String() string {
	return fmt.Sprintf("ItemUseTeleportRock{target=%s updateTime=%d}", m.target.String(), m.updateTime)
}

func (m ItemUseTeleportRock) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		m.target.Encode(w)
		w.WriteInt(m.updateTime)
		return w.Bytes()
	}
}

func (m *ItemUseTeleportRock) Decode(l logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.target.Decode(l)(r)
		if r.Available() >= 4 {
			m.updateTime = r.ReadUint32()
		}
	}
}
