package snapshot

import (
	"atlas-channel/character"
	"atlas-channel/character/buff"
	"atlas-channel/character/skill"
	"context"
	"math/rand"
	"os"
	"strconv"
	"sync"
	"sync/atomic"

	"github.com/sirupsen/logrus"

	charconst "github.com/Chronicle20/atlas/libs/atlas-constants/character"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// Shadow verification (design §8, resolves PRD Open Question 5): on a
// sampled full-hit read, asynchronously fetch the REST projection and
// compare the attack-relevant fields, logging + counting divergence. This
// answers the owner's accuracy concern with runtime evidence. Default off
// (rate 0); enabled in staging soaks via CHAR_SNAPSHOT_SHADOW_SAMPLE_RATE.

const (
	envShadowSampleRate = "CHAR_SNAPSHOT_SHADOW_SAMPLE_RATE"
	// positionToleranceBand allows for natural drift between this pod's
	// movement fold and the async REST projection of the same packets.
	positionToleranceBand = 100
	shadowMaxInFlight     = 4
)

var (
	shadowRateOnce sync.Once
	shadowRateVal  float64
	shadowSem      = make(chan struct{}, shadowMaxInFlight)
	shadowInFlight atomic.Int32
)

func shadowRate() float64 {
	shadowRateOnce.Do(func() {
		v, ok := os.LookupEnv(envShadowSampleRate)
		if !ok || v == "" {
			return
		}
		f, err := strconv.ParseFloat(v, 64)
		if err != nil || f < 0 || f > 1 {
			logrus.StandardLogger().Warnf("invalid %s=%q; shadow verification disabled", envShadowSampleRate, v)
			return
		}
		shadowRateVal = f
	})
	return shadowRateVal
}

// maybeShadow samples a full-hit read. Bounded: skips the sample when
// shadowMaxInFlight comparisons are already running (never queues, never
// blocks the attack path).
func (p *Processor) maybeShadow(characterId uint32, served character.Model, servedBuffs []buff.Model) {
	rate := shadowRate()
	if rate <= 0 || rand.Float64() >= rate {
		return
	}
	select {
	case shadowSem <- struct{}{}:
	default:
		return
	}
	shadowInFlight.Add(1)
	l, ctx, t := p.l, p.ctx, p.t
	go func() {
		defer func() {
			<-shadowSem
			shadowInFlight.Add(-1)
		}()
		shadowCompare(l, ctx, t, characterId, served, servedBuffs)
	}()
}

func shadowCompare(l logrus.FieldLogger, ctx context.Context, t tenant.Model, characterId uint32, served character.Model, servedBuffs []buff.Model) {
	core, err := coreFetchFn(l, ctx, characterId)
	if err != nil {
		return // shadow is best-effort; fallback health is already metered
	}
	inv, err := inventoryFetchFn(l, ctx, characterId)
	if err != nil {
		return
	}
	skills, err := skillsFetchFn(l, ctx, characterId)
	if err != nil {
		return
	}
	restBuffs, err := buffsFetchFn(l, ctx, characterId)
	if err != nil {
		return
	}
	restModel := core.SetInventory(inv).SetSkills(skills)
	if servedBuffs == nil {
		// Controller ruling R9: the fast-path hook does not yet carry real
		// served buffs, so there is nothing to compare against — record
		// that visibly rather than silently, and never synthesize a
		// divergence out of an absent input.
		l.WithField("component", componentBuffs).Debug("Snapshot shadow comparison skipped buffs component: no served buffs available.")
	}
	for _, component := range compareProjection(served, restModel, servedBuffs, restBuffs) {
		recordDivergence(t, component)
		l.Warnf("Snapshot shadow divergence for character [%d] component [%s].", characterId, component)
	}
}

// compareProjection compares the attack-relevant projection of two
// decorated models (+ buff sets) and returns the diverging components.
// snapBuffs == nil means "not supplied by the caller" and the buffs
// component is skipped rather than compared (controller ruling R9): a nil
// served-buffs input is a missing sample, not evidence of an empty buff
// set, so it must never be reported as a divergence.
func compareProjection(snap, rest character.Model, snapBuffs, restBuffs []buff.Model) []string {
	var out []string

	if snap.Level() != rest.Level() || snap.JobId() != rest.JobId() {
		out = append(out, componentCore)
	}

	dx, dy := int32(snap.X())-int32(rest.X()), int32(snap.Y())-int32(rest.Y())
	if dx < -positionToleranceBand || dx > positionToleranceBand || dy < -positionToleranceBand || dy > positionToleranceBand {
		out = append(out, componentPosition)
	}

	if !sameWeapon(snap, rest) || !sameAssetQuantities(snap, rest) {
		out = append(out, componentInventory)
	}

	if !sameSkillLevels(snap.Skills(), rest.Skills()) {
		out = append(out, componentSkills)
	}

	if snapBuffs != nil {
		if hasGateBuff(snapBuffs, charconst.TemporaryStatTypeSoulArrow) != hasGateBuff(restBuffs, charconst.TemporaryStatTypeSoulArrow) ||
			hasGateBuff(snapBuffs, charconst.TemporaryStatTypeShadowPartner) != hasGateBuff(restBuffs, charconst.TemporaryStatTypeShadowPartner) {
			out = append(out, componentBuffs)
		}
	}
	return out
}

func sameWeapon(a, b character.Model) bool {
	wa, oka := a.Equipment().Get("weapon")
	wb, okb := b.Equipment().Get("weapon")
	if oka != okb {
		return false
	}
	if !oka {
		return true
	}
	ta, tb := uint32(0), uint32(0)
	if wa.Equipable != nil {
		ta = wa.Equipable.TemplateId()
	}
	if wb.Equipable != nil {
		tb = wb.Equipable.TemplateId()
	}
	return ta == tb
}

func sameAssetQuantities(a, b character.Model) bool {
	type key struct {
		slot       int16
		templateId uint32
	}
	build := func(m character.Model) map[key]uint32 {
		out := map[key]uint32{}
		for _, as := range m.Inventory().Consumable().Assets() {
			out[key{as.Slot(), as.TemplateId()}] = as.Quantity()
		}
		for _, as := range m.Inventory().Cash().Assets() {
			out[key{as.Slot(), as.TemplateId()}] = as.Quantity()
		}
		return out
	}
	ma, mb := build(a), build(b)
	if len(ma) != len(mb) {
		return false
	}
	for k, v := range ma {
		if mb[k] != v {
			return false
		}
	}
	return true
}

func sameSkillLevels(a, b []skill.Model) bool {
	build := func(ms []skill.Model) map[uint32]byte {
		out := map[uint32]byte{}
		for _, s := range ms {
			out[uint32(s.Id())] = s.Level()
		}
		return out
	}
	ma, mb := build(a), build(b)
	if len(ma) != len(mb) {
		return false
	}
	for k, v := range ma {
		if mb[k] != v {
			return false
		}
	}
	return true
}

func hasGateBuff(bs []buff.Model, statType charconst.TemporaryStatType) bool {
	for _, b := range bs {
		if b.Expired() {
			continue
		}
		for _, c := range b.Changes() {
			if c.Type() == string(statType) {
				return true
			}
		}
	}
	return false
}
