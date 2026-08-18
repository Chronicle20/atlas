package dragon

import (
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
)

type Builder struct {
	ownerCharacterId uint32
	fld              field.Model
	x                int32
	y                int32
	stance           byte
	jobId            job.Id
}

// NewBuilder starts a dragon for ownerCharacterId. The owner is required at
// construction because it is the model's identity, not an optional attribute.
func NewBuilder(ownerCharacterId uint32) *Builder {
	return &Builder{ownerCharacterId: ownerCharacterId}
}

func Clone(m Model) *Builder {
	return &Builder{
		ownerCharacterId: m.ownerCharacterId,
		fld:              m.fld,
		x:                m.x,
		y:                m.y,
		stance:           m.stance,
		jobId:            m.jobId,
	}
}

func (b *Builder) SetField(f field.Model) *Builder { b.fld = f; return b }
func (b *Builder) SetX(x int32) *Builder           { b.x = x; return b }
func (b *Builder) SetY(y int32) *Builder           { b.y = y; return b }
func (b *Builder) SetStance(s byte) *Builder       { b.stance = s; return b }
func (b *Builder) SetJobId(id job.Id) *Builder     { b.jobId = id; return b }

func (b *Builder) Build() Model {
	return Model{
		ownerCharacterId: b.ownerCharacterId,
		fld:              b.fld,
		x:                b.x,
		y:                b.y,
		stance:           b.stance,
		jobId:            b.jobId,
	}
}
