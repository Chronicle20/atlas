// services/atlas-channel/atlas.com/channel/skill/handler/registry.go
package handler

import (
	"context"

	"atlas-channel/data/skill/effect"
	"atlas-channel/socket/writer"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	skill2 "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
	"github.com/sirupsen/logrus"
)

// Handler is the per-skill cast handler invoked from UseSkill after the
// generic cost / cooldown / buff steps. Heal, Dispel, Cure, MPEater,
// Drain, etc. each register an implementation.
type Handler func(l logrus.FieldLogger) func(ctx context.Context) func(
	wp writer.Producer,
	f field.Model,
	characterId uint32,
	info packetmodel.SkillUsageInfo,
	e effect.Model,
) error

var registry = map[skill2.Id]Handler{}

// Register installs a Handler for the given skill. Intended to be
// called from package init() in per-skill subpackages.
func Register(id skill2.Id, h Handler) {
	registry[id] = h
}

// Lookup returns the registered Handler for the skill, if any.
func Lookup(id skill2.Id) (Handler, bool) {
	h, ok := registry[id]
	return h, ok
}
