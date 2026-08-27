package progress

import "github.com/google/uuid"

type builder struct {
	tenantId   uuid.UUID
	id         uint32
	infoNumber uint32
	progress   string
}

func NewBuilder() *builder {
	return &builder{}
}

func CloneModel(m Model) *builder {
	return &builder{
		tenantId:   m.tenantId,
		id:         m.id,
		infoNumber: m.infoNumber,
		progress:   m.progress,
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

func (b *builder) SetInfoNumber(infoNumber uint32) *builder {
	b.infoNumber = infoNumber
	return b
}

func (b *builder) SetProgress(progress string) *builder {
	b.progress = progress
	return b
}

func (b *builder) Build() Model {
	return Model{
		tenantId:   b.tenantId,
		id:         b.id,
		infoNumber: b.infoNumber,
		progress:   b.progress,
	}
}

// BuildWithValidation returns the built Model with validation.
// Progress model has no strictly required fields for creation (InfoNumber can be 0).
func (b *builder) BuildWithValidation() (Model, error) {
	return b.Build(), nil
}
