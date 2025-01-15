package invite

import "time"

type Model struct {
	id           uint32
	inviteType   string
	referenceId  uint32
	originatorId uint32
	targetId     uint32
	age          time.Time
}

func (m Model) ReferenceId() uint32 {
	return m.referenceId
}

func (m Model) TargetId() uint32 {
	return m.targetId
}

func (m Model) OriginatorId() uint32 {
	return m.originatorId
}
