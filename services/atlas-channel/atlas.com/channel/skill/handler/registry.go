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

// AttackCastHandler is the per-skill cast handler invoked from processAttack
// (character_attack_common.go) for skills the client delivers on an ATTACK
// packet instead of USE_SKILL.
//
// This is a SEPARATE registry from Handler above, and the separation is
// load-bearing rather than stylistic:
//
//   - Some skills are dual-packet. Heal sends a magic-attack packet for the
//     undead damage AND a use-skill packet for the party heal, so invoking the
//     Handler registry from both dispatch sites would run Heal's handler twice
//     per cast.
//   - The Handler registry doubles as the "this handler owns the HP/MP cost"
//     signal that makes processAttack skip its generic cost block. An
//     attack-only skill has no buff-side packet to charge the cost on, so it
//     must NOT appear in that registry or it would cast for free (task-200:
//     Poison Mist 2111003 is a `damage`/`attackCount`/`mobCount` attack skill;
//     the client never sends USE_SKILL for it, so registering it as a Handler
//     both never fired and silently zeroed its 21+ MP cost).
//
// The signature takes the resolved skill id and level directly rather than a
// packetmodel.SkillUsageInfo: processAttack has an AttackInfo, and
// synthesizing a SkillUsageInfo would hand handlers zero-valued
// AffectedMobIds / AffectedPartyMemberBitmap fields that look real.
type AttackCastHandler func(l logrus.FieldLogger) func(ctx context.Context) func(
	wp writer.Producer,
	f field.Model,
	characterId uint32,
	skillId skill2.Id,
	skillLevel byte,
	e effect.Model,
) error

// attackCastRegistry is keyed on skill2.Identity for the same version-blind
// reason as registry above.
var attackCastRegistry = map[skill2.Identity]AttackCastHandler{}

// RegisterAttackCast installs an AttackCastHandler for the given skill
// identity. Intended to be called from package init() in per-skill
// subpackages.
func RegisterAttackCast(id skill2.Identity, h AttackCastHandler) {
	attackCastRegistry[id] = h
}

// LookupAttackCast returns the registered AttackCastHandler for the skill
// identity, if any.
func LookupAttackCast(id skill2.Identity) (AttackCastHandler, bool) {
	h, ok := attackCastRegistry[id]
	return h, ok
}

// UnregisterAttackCast removes an AttackCastHandler from the registry.
// Exposed for tests only; production code should never call this.
func UnregisterAttackCast(id skill2.Identity) {
	delete(attackCastRegistry, id)
}
