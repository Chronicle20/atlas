package character

import (
	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
)

// Model is the slice of a character atlas-dragons needs: the job (to decide
// whether a dragon exists at all) and the position (to seed the dragon's spawn
// coordinates). Nothing else is fetched.
type Model struct {
	id     uint32
	jobId  job.Id
	x      int16
	y      int16
	stance byte
}

func (m Model) Id() uint32    { return m.id }
func (m Model) JobId() job.Id { return m.jobId }
func (m Model) X() int16      { return m.x }
func (m Model) Y() int16      { return m.y }
func (m Model) Stance() byte  { return m.stance }

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

func (b *Builder) Build() Model {
	return Model{id: b.id, jobId: b.jobId, x: b.x, y: b.y, stance: b.stance}
}
