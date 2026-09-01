package character

import (
	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
)

// Model is the deploy-time snapshot input read from atlas-character: the
// appearance and identity fields FR-5.1/FR-5.2 freeze at deploy.
// Equipment is read separately from atlas-inventory (see the inventory/
// package) — atlas-character does not serve it.
type Model struct {
	id        uint32
	name      string
	gender    byte
	skinColor byte
	face      uint32
	hair      uint32
	jobId     job.Id
	level     byte
	gm        bool
}

func (m Model) Id() uint32      { return m.id }
func (m Model) Name() string    { return m.name }
func (m Model) Gender() byte    { return m.gender }
func (m Model) SkinColor() byte { return m.skinColor }
func (m Model) Face() uint32    { return m.face }
func (m Model) Hair() uint32    { return m.hair }
func (m Model) JobId() job.Id   { return m.jobId }
func (m Model) Level() byte     { return m.level }
func (m Model) Gm() bool        { return m.gm }
