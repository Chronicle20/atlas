package party

import (
	"github.com/google/uuid"
	"math/rand"
)

type Model struct {
	tenantId uuid.UUID
	id       uint32
	leaderId uint32
	members  []uint32
}

func (m Model) ElectLeader() Model {
	index := rand.Intn(len(m.members))
	return Model{
		tenantId: m.tenantId,
		id:       m.id,
		leaderId: m.members[index],
		members:  m.members,
	}
}

func (m Model) SetLeader(leaderId uint32) Model {
	return Model{
		tenantId: m.tenantId,
		id:       m.id,
		leaderId: leaderId,
		members:  m.members,
	}
}

func (m Model) AddMember(memberId uint32) Model {
	ms := append(m.members, memberId)
	return Model{
		tenantId: m.tenantId,
		id:       m.id,
		leaderId: m.leaderId,
		members:  ms,
	}
}

func (m Model) RemoveMember(memberId uint32) Model {
	oms := make([]uint32, 0)
	for _, m := range m.members {
		if m != memberId {
			oms = append(oms, m)
		}
	}

	return Model{
		tenantId: m.tenantId,
		id:       m.id,
		leaderId: m.leaderId,
		members:  oms,
	}
}

func (m Model) Id() uint32 {
	return m.id
}

func (m Model) LeaderId() uint32 {
	return m.leaderId
}
