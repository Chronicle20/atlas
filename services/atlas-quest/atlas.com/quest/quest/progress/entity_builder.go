package progress

import "github.com/google/uuid"

type entityBuilder struct {
	tenantId      uuid.UUID
	id            uint32
	questStatusId uint32
	infoNumber    uint32
	progress      string
}

func NewEntityBuilder() *entityBuilder {
	return &entityBuilder{}
}

func CloneEntity(e Entity) *entityBuilder {
	return &entityBuilder{
		tenantId:      e.TenantId,
		id:            e.ID,
		questStatusId: e.QuestStatusId,
		infoNumber:    e.InfoNumber,
		progress:      e.Progress,
	}
}

func (b *entityBuilder) SetTenantId(tenantId uuid.UUID) *entityBuilder {
	b.tenantId = tenantId
	return b
}

func (b *entityBuilder) SetId(id uint32) *entityBuilder {
	b.id = id
	return b
}

func (b *entityBuilder) SetQuestStatusId(questStatusId uint32) *entityBuilder {
	b.questStatusId = questStatusId
	return b
}

func (b *entityBuilder) SetInfoNumber(infoNumber uint32) *entityBuilder {
	b.infoNumber = infoNumber
	return b
}

func (b *entityBuilder) SetProgress(progress string) *entityBuilder {
	b.progress = progress
	return b
}

func (b *entityBuilder) Build() Entity {
	return Entity{
		TenantId:      b.tenantId,
		ID:            b.id,
		QuestStatusId: b.questStatusId,
		InfoNumber:    b.infoNumber,
		Progress:      b.progress,
	}
}
