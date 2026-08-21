package character

import (
	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
)

// EquippedItem is a single occupied slot decoded from the character's
// equipment relationship (see rest.go's SetReferencedStructs). Callers
// that need the FR-5.2 signed slot filtering (1-11, 101-111) apply it
// themselves — this client hands back every slot the upstream returned.
type EquippedItem struct {
	slot       int16
	templateId uint32
}

func (e EquippedItem) Slot() int16        { return e.slot }
func (e EquippedItem) TemplateId() uint32 { return e.templateId }

// Model is the deploy-time snapshot input read from atlas-character: the
// appearance and identity fields FR-5.1/FR-5.2 freeze at deploy, plus the
// equipped items backing the appearance snapshot.
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
	equipment []EquippedItem
}

func (m Model) Id() uint32                { return m.id }
func (m Model) Name() string              { return m.name }
func (m Model) Gender() byte              { return m.gender }
func (m Model) SkinColor() byte           { return m.skinColor }
func (m Model) Face() uint32              { return m.face }
func (m Model) Hair() uint32              { return m.hair }
func (m Model) JobId() job.Id             { return m.jobId }
func (m Model) Level() byte               { return m.level }
func (m Model) Gm() bool                  { return m.gm }
func (m Model) Equipment() []EquippedItem { return m.equipment }
