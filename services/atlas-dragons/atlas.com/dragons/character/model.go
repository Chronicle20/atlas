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
