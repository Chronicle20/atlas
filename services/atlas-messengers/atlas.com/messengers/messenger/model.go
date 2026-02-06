package messenger

import (
	"errors"

	"github.com/google/uuid"
)

type Model struct {
	tenantId uuid.UUID
	id       uint32
	members  []MemberModel
}

type MemberModel struct {
	id   uint32
	slot byte
}

func (m MemberModel) Id() uint32 {
	return m.id
}

func (m MemberModel) Slot() byte {
	return m.slot
}

func (m Model) FirstOpenSlot() byte {
	usedSlots := make(map[byte]bool)

	for _, item := range m.Members() {
		usedSlots[item.Slot()] = true
	}

	for i := byte(0); ; i++ {
		if !usedSlots[i] {
			return i
		}
	}
}

func (m Model) AddMember(memberId uint32) Model {
	ms := append(m.members, MemberModel{id: memberId, slot: m.FirstOpenSlot()})
	return Model{
		tenantId: m.tenantId,
		id:       m.id,
		members:  ms,
	}
}

func (m Model) RemoveMember(memberId uint32) Model {
	oms := make([]MemberModel, 0)
	for _, m := range m.members {
		if m.Id() != memberId {
			oms = append(oms, m)
		}
	}

	return Model{
		tenantId: m.tenantId,
		id:       m.id,
		members:  oms,
	}
}

func (m Model) Id() uint32 {
	return m.id
}

func (m Model) Members() []MemberModel {
	return m.members
}

func (m Model) FindMember(id uint32) (MemberModel, error) {
	for _, mm := range m.Members() {
		if mm.Id() == id {
			return mm, nil
		}
	}
	return MemberModel{}, errors.New("not found")
}
