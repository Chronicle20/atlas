package character

import (
	"errors"

	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
)

// Builder constructs a Model. Immutable-model discipline: no test-only
// constructor, this is also what production Extract goes through.
type Builder struct {
	id     uint32
	jobId  job.Id
	x      int16
	y      int16
	stance byte
}

func NewBuilder(id uint32) *Builder { return &Builder{id: id} }

func (b *Builder) SetJobId(id job.Id) *Builder { b.jobId = id; return b }
func (b *Builder) SetX(x int16) *Builder       { b.x = x; return b }
func (b *Builder) SetY(y int16) *Builder       { b.y = y; return b }
func (b *Builder) SetStance(s byte) *Builder   { b.stance = s; return b }

func (b *Builder) Build() (Model, error) {
	if b.id == 0 {
		return Model{}, errors.New("id is required")
	}
	return Model{id: b.id, jobId: b.jobId, x: b.x, y: b.y, stance: b.stance}, nil
}
