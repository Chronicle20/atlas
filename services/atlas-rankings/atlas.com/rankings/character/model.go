package character

import (
	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

// Model is the trimmed, immutable read model of an atlas-character
// character — only the attributes ranking computation needs.
type Model struct {
	id         uint32
	worldId    world.Id
	jobId      job.Id
	level      byte
	experience uint32
	gm         int
}

func (m Model) Id() uint32         { return m.id }
func (m Model) WorldId() world.Id  { return m.worldId }
func (m Model) JobId() job.Id      { return m.jobId }
func (m Model) Level() byte        { return m.level }
func (m Model) Experience() uint32 { return m.experience }
func (m Model) Gm() int            { return m.gm }
