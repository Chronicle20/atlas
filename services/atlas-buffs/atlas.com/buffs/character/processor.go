package character

import (
	"atlas-buffs/buff/stat"
	extchar "atlas-buffs/external/character"
	"atlas-buffs/kafka/message"
	character2 "atlas-buffs/kafka/message/character"
	"atlas-buffs/periodic"
	"context"
	"errors"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-rest/degrade"
	routine "github.com/Chronicle20/atlas/libs/atlas-routine"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

type Processor interface {
	GetById(characterId uint32) (Model, error)
	Apply(worldId world.Id, channelId channel.Id, characterId uint32, fromId uint32, sourceId int32, level byte, duration int32, changes []stat.Model, accumulate bool, noExpiry bool) error
	Cancel(worldId world.Id, characterId uint32, sourceId int32) error
	CancelAll(worldId world.Id, characterId uint32) error
	CancelByStatTypes(worldId world.Id, characterId uint32, types []string) error
	UpdateStatValue(worldId world.Id, characterId uint32, sourceId int32, statType string, operation string, amount int32, capValue int32) error
	ExpireBuffs() error
	ExpireForCharacter(worldId world.Id, characterId uint32) error
	ProcessPeriodicTicks() error
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	// now and getCharacterHp are injected so the periodic tick pass is
	// deterministic under test (same shape as berserk.ProcessorImpl).
	now            func() time.Time
	getCharacterHp func(characterId uint32) (uint16, error)
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	p := &ProcessorImpl{
		l:   l,
		ctx: ctx,
		now: time.Now,
	}
	p.getCharacterHp = func(characterId uint32) (uint16, error) {
		rm, err := extchar.RequestById(characterId)(l, ctx)
		if err != nil {
			return 0, err
		}
		return rm.Hp, nil
	}
	return p
}

var _ Processor = (*ProcessorImpl)(nil)

func (p *ProcessorImpl) GetById(characterId uint32) (Model, error) {
	return GetRegistry().Get(p.ctx, characterId)
}

func (p *ProcessorImpl) Apply(worldId world.Id, channelId channel.Id, characterId uint32, fromId uint32, sourceId int32, level byte, duration int32, changes []stat.Model, accumulate bool, noExpiry bool) error {
	if isDiseaseChange(changes) && GetRegistry().HasImmunity(p.ctx, characterId) {
		p.l.Debugf("Character [%d] is immune to disease, skipping apply.", characterId)
		return nil
	}

	err := message.Emit(p.l, p.ctx)(func(buf *message.Buffer) error {
		applied, err := GetRegistry().Apply(p.ctx, worldId, channelId, characterId, sourceId, level, duration, changes, accumulate, noExpiry)
		if err != nil {
			return err
		}
		// One APPLIED per stored buff: default mode returns a single whole-source
		// buff; accumulate mode returns one buff per stat, each carrying its own
		// changes/expiry so the channel sets (and later cancels) each stat icon
		// independently.
		for _, b := range applied {
			if err := buf.Put(character2.EnvEventStatusTopic, appliedStatusEventProvider(worldId, characterId, fromId, b.SourceId(), b.Level(), b.Duration(), b.Changes(), b.CreatedAt(), b.ExpiresAt(), b.NoExpiry())); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	markBerserkDirtyOnMaxHpChange(p.l, p.ctx, characterId, changes)
	return nil
}

func (p *ProcessorImpl) Cancel(worldId world.Id, characterId uint32, sourceId int32) error {
	cancelled, err := GetRegistry().Cancel(p.ctx, characterId, sourceId)
	if errors.Is(err, ErrNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	// One EXPIRED per removed buff: a sourceId can map to several per-stat buffs
	// in accumulate mode (Beholder Hex), and each needs its own cancel so the
	// client clears every icon rather than leaving the others stuck.
	err = message.Emit(p.l, p.ctx)(func(buf *message.Buffer) error {
		for _, b := range cancelled {
			if err := buf.Put(character2.EnvEventStatusTopic, expiredStatusEventProvider(worldId, characterId, b.SourceId(), b.Level(), b.Duration(), b.Changes(), b.CreatedAt(), b.ExpiresAt(), b.NoExpiry())); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	sets := make([][]stat.Model, 0, len(cancelled))
	for _, b := range cancelled {
		sets = append(sets, b.Changes())
	}
	GetRegistry().ClearPeriodicTicksFor(p.ctx, characterId, sets...)
	markBerserkDirtyOnMaxHpChange(p.l, p.ctx, characterId, sets...)
	return nil
}

func (p *ProcessorImpl) CancelAll(worldId world.Id, characterId uint32) error {
	buffs := GetRegistry().CancelAll(p.ctx, characterId)
	if len(buffs) == 0 {
		return nil
	}
	err := message.Emit(p.l, p.ctx)(func(buf *message.Buffer) error {
		for _, b := range buffs {
			if err := buf.Put(character2.EnvEventStatusTopic, expiredStatusEventProvider(worldId, characterId, b.SourceId(), b.Level(), b.Duration(), b.Changes(), b.CreatedAt(), b.ExpiresAt(), b.NoExpiry())); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	sets := make([][]stat.Model, 0, len(buffs))
	for _, b := range buffs {
		sets = append(sets, b.Changes())
	}
	GetRegistry().ClearPeriodicTicksFor(p.ctx, characterId, sets...)
	markBerserkDirtyOnMaxHpChange(p.l, p.ctx, characterId, sets...)
	return nil
}

func (p *ProcessorImpl) CancelByStatTypes(worldId world.Id, characterId uint32, types []string) error {
	if len(types) == 0 {
		return nil
	}
	typeSet := make(map[string]bool, len(types))
	for _, t := range types {
		typeSet[t] = true
	}

	cancelled, err := GetRegistry().CancelByStatTypes(p.ctx, characterId, typeSet)
	if err != nil {
		return err
	}
	if len(cancelled) == 0 {
		return nil
	}

	err = message.Emit(p.l, p.ctx)(func(buf *message.Buffer) error {
		for _, b := range cancelled {
			if err := buf.Put(character2.EnvEventStatusTopic, expiredStatusEventProvider(worldId, characterId, b.SourceId(), b.Level(), b.Duration(), b.Changes(), b.CreatedAt(), b.ExpiresAt(), b.NoExpiry())); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	sets := make([][]stat.Model, 0, len(cancelled))
	for _, b := range cancelled {
		sets = append(sets, b.Changes())
	}
	GetRegistry().ClearPeriodicTicksFor(p.ctx, characterId, sets...)
	markBerserkDirtyOnMaxHpChange(p.l, p.ctx, characterId, sets...)
	return nil
}

// UpdateStatValue applies a stat-value mutation to an existing buff and, when
// the value actually changed, emits a STAT_UPDATED status event carrying the
// buff's original createdAt/expiresAt (so the channel re-broadcasts the
// remaining duration). Missing/expired buff and at-cap increments are Debug
// no-ops — the buff can lapse between the channel's attack and this command.
func (p *ProcessorImpl) UpdateStatValue(worldId world.Id, characterId uint32, sourceId int32, statType string, operation string, amount int32, capValue int32) error {
	if operation != character2.StatOperationIncrement && operation != character2.StatOperationSet {
		p.l.Warnf("Unknown stat value operation [%s] for character [%d] buff [%d]; ignoring.", operation, characterId, sourceId)
		return nil
	}
	return message.Emit(p.l, p.ctx)(func(buf *message.Buffer) error {
		updated, changed, err := GetRegistry().UpdateStatValue(p.ctx, characterId, sourceId, statType, operation, amount, capValue)
		if err != nil {
			return err
		}
		if !changed {
			p.l.Debugf("No stat value change for character [%d] buff [%d] stat [%s].", characterId, sourceId, statType)
			return nil
		}
		return buf.Put(character2.EnvEventStatusTopic, statUpdatedStatusEventProvider(worldId, characterId, updated.SourceId(), updated.Level(), updated.Duration(), updated.Changes(), updated.CreatedAt(), updated.ExpiresAt()))
	})
}

func (p *ProcessorImpl) ExpireBuffs() error {
	return message.Emit(p.l, p.ctx)(func(buf *message.Buffer) error {
		for _, c := range GetRegistry().GetCharacters(p.ctx) {
			if err := p.expireInto(buf, c.WorldId(), c.Id()); err != nil {
				return err
			}
		}
		return nil
	})
}

// ExpireForCharacter sweeps ONE character, so a single client's CANCEL_DEBUFF
// nudge does not force a fleet-wide pass. WorldId comes from the command
// envelope — the channel knows the live session's world, which is authoritative
// for an in-session character. Semantics are identical to the fleet sweep by
// construction: both call expireInto. (task-190 FR-2.6.1)
func (p *ProcessorImpl) ExpireForCharacter(worldId world.Id, characterId uint32) error {
	return message.Emit(p.l, p.ctx)(func(buf *message.Buffer) error {
		return p.expireInto(buf, worldId, characterId)
	})
}

// expireInto prunes one character's lapsed buffs and puts one EXPIRED event per
// lapsed buff on buf. Registry.GetExpired already does prune-and-return, so no
// new expiry semantics are invented here. When nothing has lapsed it puts
// nothing, and message.Emit then emits nothing — FR-2.9 / NFR-2.1 hold
// structurally, not by an explicit guard.
func (p *ProcessorImpl) expireInto(buf *message.Buffer, worldId world.Id, characterId uint32) error {
	ebs := GetRegistry().GetExpired(p.ctx, characterId)
	for _, eb := range ebs {
		p.l.Debugf("Expired buff for character [%d] from [%d].", characterId, eb.SourceId())
		if err := buf.Put(character2.EnvEventStatusTopic, expiredStatusEventProvider(worldId, characterId, eb.SourceId(), eb.Level(), eb.Duration(), eb.Changes(), eb.CreatedAt(), eb.ExpiresAt(), eb.NoExpiry())); err != nil {
			return err
		}
	}
	if len(ebs) > 0 {
		sets := make([][]stat.Model, 0, len(ebs))
		for _, eb := range ebs {
			sets = append(sets, eb.Changes())
		}
		GetRegistry().ClearPeriodicTicksFor(p.ctx, characterId, sets...)
		markBerserkDirtyOnMaxHpChange(p.l, p.ctx, characterId, sets...)
	}
	return nil
}

func ExpireBuffs(l logrus.FieldLogger, ctx context.Context) error {
	ts, err := GetRegistry().GetTenants(ctx)
	if err != nil {
		return err
	}

	for _, t := range ts {
		routine.Go(l, ctx, func(_ context.Context) {
			tctx := tenant.WithContext(ctx, t)
			if err := NewProcessor(l, tctx).ExpireBuffs(); err != nil {
				l.WithError(err).Error("Failed to expire buffs for tenant.")
			}
		})
	}
	return nil
}

// hpLookup memoizes one character's HP read for the duration of a single tick
// pass, including the failure outcome — a character whose HP could not be read
// is not retried within the same pass (FR-3.6).
type hpLookup struct {
	hp uint16
	ok bool
}

// maxTickMagnitude clamps a per-tick magnitude before the int16 conversion the
// CHANGE_HP command body requires. No real WZ value approaches it; the clamp
// exists so a corrupt stored amount degrades to a large tick instead of
// wrapping sign and turning a drain into a heal.
const maxTickMagnitude = int32(32767)

// ProcessPeriodicTicks is one tick pass for one tenant. It scans once
// (GetPeriodicEntries), then for each due (character, statType) emits the
// resource change the periodic-effect table prescribes. A row is due when it
// has never ticked or when now - lastTick >= the row's interval, so one 1s
// driving task honors every row's own cadence (FR-2.3).
//
// The throttle read and the store update straddle buf.Put, exactly as the
// pre-task-214 poison path did: a crash between the two re-ticks on the next
// pass, a crash before Put skips a tick. Both are one-interval errors on a
// non-idempotent HP mutation. Making this exactly-once needs an idempotency key
// on the CHANGE_HP command — an atlas-character contract change, out of scope
// for task-214 (design.md §3.5).
func (p *ProcessorImpl) ProcessPeriodicTicks() error {
	entries, err := GetRegistry().GetPeriodicEntries(p.ctx)
	if err != nil {
		// A scan failure degrades to "no ticks this pass, try again next
		// interval" rather than propagating: the ticker must not crash on a
		// transient Redis blip, and the next 1s pass self-heals. Logged here
		// (not returned) so this failure reads distinctly from a genuine
		// message-emit error below, which the outer tenant loop does treat
		// as pass-level failure.
		p.l.WithError(err).Warn("Periodic tick scan failed; skipping this tick pass.")
		return nil
	}
	now := p.now()
	hpCache := make(map[uint32]hpLookup)

	return message.Emit(p.l, p.ctx)(func(buf *message.Buffer) error {
		for _, entry := range entries {
			eff, ok := periodic.Lookup(entry.StatType)
			if !ok {
				continue
			}

			key := TickKey{CharacterId: entry.CharacterId, StatType: entry.StatType}
			if last, ticked := GetRegistry().GetPeriodicTick(p.ctx, key); ticked && now.Sub(last) < eff.Interval() {
				continue
			}

			// A non-positive stored magnitude is skipped, preserving the
			// pre-task-214 poison guard generically (FR-1.5).
			magnitude := entry.Amount
			if magnitude <= 0 {
				continue
			}
			if magnitude > maxTickMagnitude {
				magnitude = maxTickMagnitude
			}

			// One arm per resource. The default is a guard, not a stub: it is
			// unreachable with today's rows, and its job is to make a future MP
			// row fail loudly at the first tick instead of silently emitting
			// nothing.
			switch eff.Resource() {
			case periodic.ResourceHP:
			default:
				p.l.Errorf("Periodic effect [%s] targets unmapped resource [%s]; no command emitted.", entry.StatType, eff.Resource())
				continue
			}

			amount := int16(eff.Direction()) * int16(magnitude)

			if eff.Floor() && amount < 0 {
				hp, ok := p.hpFor(hpCache, entry.CharacterId)
				if !ok {
					continue
				}
				if hp <= 1 {
					p.l.Debugf("Periodic tick [%s] for character [%d] suppressed: already at [%d] HP.", entry.StatType, entry.CharacterId, hp)
					continue
				}
				if int32(hp)+int32(amount) < 1 {
					amount = -int16(hp - 1)
				}
			}

			p.l.Debugf("Periodic tick [%s] for character [%d], amount [%d].", entry.StatType, entry.CharacterId, amount)

			if err := buf.Put(character2.EnvCommandTopicCharacter, changeHPCommandProvider(entry.WorldId, entry.ChannelId, entry.CharacterId, amount)); err != nil {
				return err
			}

			GetRegistry().UpdatePeriodicTick(p.ctx, key, now)
		}
		return nil
	})
}

// hpFor reads a character's current HP at most once per tick pass. A read
// failure is cached as a miss and logged: the caller skips the tick rather than
// emitting an unclamped drain, because one missed 4s tick is invisible and one
// unintended DIED is not (design D5).
func (p *ProcessorImpl) hpFor(cache map[uint32]hpLookup, characterId uint32) (uint16, bool) {
	if c, seen := cache[characterId]; seen {
		return c.hp, c.ok
	}
	hp, err := p.getCharacterHp(characterId)
	if err != nil {
		degrade.Observe(p.l, "buffs.periodic.character_hp", characterId, err)
		cache[characterId] = hpLookup{}
		return 0, false
	}
	cache[characterId] = hpLookup{hp: hp, ok: true}
	return hp, true
}

// ProcessPeriodicTicks fans one tick pass out per tenant (FR-2.5): tenant work
// runs under tenant.WithContext in a routine.Go goroutine, same shape as the
// expiration and berserk sweeps.
func ProcessPeriodicTicks(l logrus.FieldLogger, ctx context.Context) error {
	ts, err := GetRegistry().GetTenants(ctx)
	if err != nil {
		return err
	}

	for _, t := range ts {
		routine.Go(l, ctx, func(_ context.Context) {
			tctx := tenant.WithContext(ctx, t)
			if err := NewProcessor(l, tctx).ProcessPeriodicTicks(); err != nil {
				l.WithError(err).Error("Failed to process periodic ticks for tenant.")
			}
		})
	}
	return nil
}
