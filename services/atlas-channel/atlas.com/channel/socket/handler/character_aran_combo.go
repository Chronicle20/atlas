package handler

import (
	"atlas-channel/character"
	"atlas-channel/character/buff"
	"atlas-channel/character/combo"
	skill2 "atlas-channel/data/skill"
	"atlas-channel/data/skill/effect/statup"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"
	"time"

	"github.com/sirupsen/logrus"

	constants "github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	charcb "github.com/Chronicle20/atlas/libs/atlas-packet/character/clientbound"
	charsb "github.com/Chronicle20/atlas/libs/atlas-packet/character/serverbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// eligibilityTTL bounds how long a cached gate result is trusted. The attack
// pipeline refreshes it on every melee hit, so this only covers the cold case
// (a fresh login whose first combo packet beats the attack hook) and a
// modified client sending the op without attacking. Staleness is benign: an
// unequip stops the real client sending at all (task-217 design.md §3.5).
const eligibilityTTL = 60 * time.Second

// idleWindowFromOptions resolves the tenant's configured idle window. The
// client's own ClearCombo timer is 3000 ms on v83/v84/v87/v92/jms185 and
// 5000 ms on v95; the server decay must AGREE with it rather than drive it,
// so the value is tenant configuration instead of a compiled major-version
// branch (task-217 design.md §2.5, §4).
func idleWindowFromOptions(options map[string]interface{}) time.Duration {
	if options == nil {
		return combo.DefaultIdleWindow
	}
	raw, ok := options["idleResetMs"]
	if !ok {
		return combo.DefaultIdleWindow
	}
	var ms float64
	switch v := raw.(type) {
	case float64:
		ms = v
	case int:
		ms = float64(v)
	case int64:
		ms = float64(v)
	default:
		return combo.DefaultIdleWindow
	}
	if ms <= 0 {
		return combo.DefaultIdleWindow
	}
	return time.Duration(ms) * time.Millisecond
}

// aranComboDeps groups the side-effecting lookups the increment path needs so
// the decision is unit-testable without a live session or broker. Mirrors the
// comboOrbDeps / comboOrbProductionDeps split in character_attack_combo.go.
type aranComboDeps struct {
	eligibility func(characterId uint32) (combo.Eligibility, bool)
	seed        func(el combo.Eligibility, characterId uint32) error
	announce    func(count uint32) error
	now         func() time.Time
}

// aranComboProductionDeps wires aranComboDeps to the real cache-then-fetch
// eligibility lookup, the Combo Ability seed emit, and the SHOW_COMBO write.
func aranComboProductionDeps(l logrus.FieldLogger, ctx context.Context, wp writer.Producer, s session.Model) aranComboDeps {
	t := tenant.MustFromContext(ctx)
	return aranComboDeps{
		eligibility: func(characterId uint32) (combo.Eligibility, bool) {
			if el, ok := combo.GetMirror().Eligibility(t, characterId, time.Now(), eligibilityTTL); ok {
				return el, true
			}
			// Cold start: the attack hook has not run for this character yet
			// (fresh login whose first combo packet beats the attack). One
			// triple-decorator fetch, then cached for eligibilityTTL.
			cp := character.NewProcessor(l, ctx)
			c, err := cp.GetById(cp.InventoryDecorator, cp.SkillModelDecorator)(characterId)
			if err != nil {
				l.WithError(err).Debugf("Aran combo: character [%d] fetch failed; ignoring increment.", characterId)
				return combo.Eligibility{}, false
			}
			el, gate, ok := combo.Evaluate(c, skill2.NewProcessor(l, ctx).GetEffect)
			if !ok {
				l.Debugf("Aran combo: character [%d] rejected at gate [%s].", characterId, gate)
				return combo.Eligibility{}, false
			}
			combo.GetMirror().SetEligibility(t, characterId, s.Field(), el, time.Now())
			return el, true
		},
		seed: func(el combo.Eligibility, characterId uint32) error {
			ups := []statup.Model{statup.NewModel(string(constants.TemporaryStatTypeAranCombo), el.StatAmount())}
			return buff.NewProcessor(l, ctx).ApplyNoExpiry(s.Field(), characterId, int32(el.ComboId()), el.ComboLevel(), ups)(characterId)
		},
		announce: func(count uint32) error {
			return session.Announce(l)(ctx)(wp)(charcb.ShowComboWriter)(charcb.NewShowCombo(count).Encode)(s)
		},
		now: time.Now,
	}
}

// aranComboAdvance advances the counter for one accepted increment request.
//
// The Combo Ability buff is seeded exactly once per combo chain -- it carries
// the icon and the ARAN_COMBO damage-calc stat, never the count -- and a seed
// failure is logged and swallowed so the counter still advances (NFR-4).
//
// At the cap, Increment still refreshes lastHit (a player pinned at the cap
// must not decay while they are still hitting) and returns the unchanged
// count, so the client still gets one response per request.
func aranComboAdvance(l logrus.FieldLogger, deps aranComboDeps, t tenant.Model, characterId uint32, f field.Model, options map[string]interface{}) {
	el, ok := deps.eligibility(characterId)
	if !ok {
		return
	}
	now := deps.now()
	count, seeded := combo.GetMirror().Increment(t, characterId, f, idleWindowFromOptions(options), now)
	if seeded {
		if err := deps.seed(el, characterId); err != nil {
			l.WithError(err).Errorf("Aran combo: Combo Ability seed emit failed for character [%d].", characterId)
		}
	}
	if count == 0 {
		return
	}
	l.Debugf("Aran combo: character [%d] count [%d].", characterId, count)
	if err := deps.announce(uint32(count)); err != nil {
		l.WithError(err).Errorf("Aran combo: SHOW_COMBO write failed for character [%d].", characterId)
	}
}

// AranComboCounterHandleFunc advances the Aran combo counter for one client
// increment request. The packet carries NO body (CUserLocal::RequestIncCombo
// is a guard plus COutPacket plus SendPacket), so every gate the client
// applied is re-derived here from authoritative state.
//
// Steady-state cost is one mutex-guarded map read/write plus one socket
// write: the count lives in the channel-local mirror, not on a buff stat, so
// no Kafka round trip stands between the request and its response
// (task-217 design.md §3.3, §5.1).
func AranComboCounterHandleFunc(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		p := charsb.AranComboCounterRequest{}
		p.Decode(l, ctx)(r, readerOptions)
		aranComboAdvance(l, aranComboProductionDeps(l, ctx, wp, s), tenant.MustFromContext(ctx), s.CharacterId(), s.Field(), readerOptions)
	}
}
