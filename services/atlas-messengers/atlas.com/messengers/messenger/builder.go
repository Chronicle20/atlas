package messenger

import (
	"github.com/google/uuid"
)

const MaxMembers = 3

type builder struct {
	tenantId uuid.UUID
	id       uint32
	members  []MemberModel
}

func NewBuilder() *builder {
	return &builder{
		members: make([]MemberModel, 0),
	}
}

func (b *builder) SetTenantId(tenantId uuid.UUID) *builder {
	b.tenantId = tenantId
	return b
}

func (b *builder) SetId(id uint32) *builder {
	b.id = id
	return b
}

func (b *builder) AddMember(memberId uint32, slot byte) *builder {
	b.members = append(b.members, MemberModel{id: memberId, slot: slot})
	return b
}

func (b *builder) Build() (Model, error) {
	if len(b.members) > MaxMembers {
		return Model{}, ErrAtCapacity
	}
	return Model{
		tenantId: b.tenantId,
		id:       b.id,
		members:  b.members,
	}, nil
}
