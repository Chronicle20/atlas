package invite

type Model struct {
	id           uint32
	referenceId  uint32
	originatorId uint32
	targetId     uint32
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
