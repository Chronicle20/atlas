package handler

import (
	"atlas-channel/data/skill/effect"
	"atlas-channel/socket/writer"
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	skill2 "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
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

// registry is keyed on skill2.Identity (task-187): a version-blind skill
// concept, not a raw per-version wire id. A single Register(skill2.SuperGmHide,
// Apply) call covers every provisioned version regardless of which wire id
// that version binds SuperGmHide to (5101004 at v48, 9101004 at v83+) --
// dispatch resolves the incoming wire id to its Identity BEFORE calling
// Lookup (see common.go UseSkill / character_attack_common.go processAttack).
var registry = map[skill2.Identity]Handler{}

// Register installs a Handler for the given skill identity. Intended to be
// called from package init() in per-skill subpackages. Registration is
// version-independent -- the handler is keyed on the stable identity, not a
// wire id.
func Register(id skill2.Identity, h Handler) {
	registry[id] = h
}

// Lookup returns the registered Handler for the skill identity, if any.
func Lookup(id skill2.Identity) (Handler, bool) {
	h, ok := registry[id]
	return h, ok
}

// Unregister removes a Handler from the registry. Exposed for tests
// that register a stub handler under a synthetic identity; production code
// should never call this.
func Unregister(id skill2.Identity) {
	delete(registry, id)
}
