package progress

import "github.com/google/uuid"

type modelBuilder struct {
	tenantId   uuid.UUID
	id         uint32
	infoNumber uint32
	progress   string
}

func NewModelBuilder() *modelBuilder {
	return &modelBuilder{}
}

func CloneModel(m Model) *modelBuilder {
	return &modelBuilder{
		tenantId:   m.tenantId,
		id:         m.id,
		infoNumber: m.infoNumber,
		progress:   m.progress,
	}
}

func (b *modelBuilder) SetTenantId(tenantId uuid.UUID) *modelBuilder {
	b.tenantId = tenantId
	return b
}

func (b *modelBuilder) SetId(id uint32) *modelBuilder {
	b.id = id
	return b
}

func (b *modelBuilder) SetInfoNumber(infoNumber uint32) *modelBuilder {
	b.infoNumber = infoNumber
	return b
}

func (b *modelBuilder) SetProgress(progress string) *modelBuilder {
	b.progress = progress
	return b
}

func (b *modelBuilder) Build() Model {
	return Model{
		tenantId:   b.tenantId,
		id:         b.id,
		infoNumber: b.infoNumber,
		progress:   b.progress,
	}
}

// BuildWithValidation returns the built Model with validation.
// Progress model has no strictly required fields for creation (InfoNumber can be 0).
func (b *modelBuilder) BuildWithValidation() (Model, error) {
	return b.Build(), nil
}
