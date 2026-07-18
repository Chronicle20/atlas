package berserk

import (
	"atlas-buffs/kafka/message"
	"context"
	"errors"
	"time"

	extchar "atlas-buffs/external/character"
	exteffstats "atlas-buffs/external/effectivestats"
	extskills "atlas-buffs/external/skills"

	character2 "atlas-buffs/kafka/message/character"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	"github.com/Chronicle20/atlas/libs/atlas-constants/stat"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
	routine "github.com/Chronicle20/atlas/libs/atlas-routine"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

type Processor interface {
	TrackOnLogin(worldId world.Id, channelId channel.Id, characterId uint32) error
	Untrack(characterId uint32) error
	HandleStatChanged(worldId world.Id, channelId channel.Id, characterId uint32, updates []stat.Type) error
	HandleTransfer(worldId world.Id, channelId channel.Id, characterId uint32) error
	HandleSkillUpdated(worldId world.Id, characterId uint32, level byte) error
	MarkMaxHpDirty(characterId uint32) error
	ProcessTicks() error
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	now func() time.Time

	getCharacter  func(characterId uint32) (extchar.RestModel, error)
	getSkillLevel func(characterId uint32) (byte, error)
	getMaxHp      func(worldId world.Id, channelId channel.Id, characterId uint32) (uint32, error)
	getEffectX    func(skillLevel byte) (int16, error)
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	p := &ProcessorImpl{
		l:   l,
		ctx: ctx,
		now: time.Now,
	}
	p.getCharacter = func(characterId uint32) (extchar.RestModel, error) {
		return extchar.RequestById(characterId)(l, ctx)
	}
	p.getSkillLevel = func(characterId uint32) (byte, error) {
		rm, err := extskills.RequestByCharacterAndSkill(characterId, uint32(skill.DarkKnightBerserkId))(l, ctx)
		if errors.Is(err, requests.ErrNotFound) {
			// The character never learned the skill: level 0, not an error.
			return 0, nil
		}
		if err != nil {
			return 0, err
		}
		return rm.Level, nil
	}
	p.getMaxHp = func(worldId world.Id, channelId channel.Id, characterId uint32) (uint32, error) {
		rm, err := exteffstats.RequestByCharacter(worldId, channelId, characterId)(l, ctx)
		if err != nil {
			return 0, err
		}
		return rm.MaxHp, nil
	}
	p.getEffectX = func(skillLevel byte) (int16, error) {
		return GetEffectXCache().X(l, ctx, skillLevel)
	}
	return p
}

// TrackOnLogin is the only consumer-driven REST call (design D3): no event
// carries the Berserk level at login. Level 0 (all non-Dark-Knights) is
// filtered here — no registry entry, no ticker work, no events.
func (p *ProcessorImpl) TrackOnLogin(worldId world.Id, channelId channel.Id, characterId uint32) error {
	level, err := p.getSkillLevel(characterId)
	if err != nil {
		return err
	}
	if level == 0 {
		return nil
	}
	m := NewBuilder(worldId, characterId, level).
		SetChannel(channelId).
		SetDirtyAt(p.now()).
		Build()
	p.l.Infof("Tracking berserk for character [%d] at skill level [%d].", characterId, level)
	return GetRegistry().Track(p.ctx, m)
}

func (p *ProcessorImpl) Untrack(characterId uint32) error {
	p.l.Infof("Untracking berserk for character [%d].", characterId)
	return GetRegistry().Untrack(p.ctx, characterId)
}

// HandleStatChanged refreshes the routing channel (design D8: every
// channel-bearing character event refreshes it) and marks dirty when the
// update touches HP. MAX_HP updates get the grace deferral even when HP moved
// too: the effective-stats MAX_HP recompute is exactly what the grace waits
// out (design D5).
func (p *ProcessorImpl) HandleStatChanged(worldId world.Id, channelId channel.Id, characterId uint32, updates []stat.Type) error {
	if err := GetRegistry().UpdateChannel(p.ctx, characterId, worldId, channelId); err != nil {
		return err
	}
	var dirtyAt time.Time
	for _, u := range updates {
		if u == stat.TypeMaxHp {
			dirtyAt = p.now().Add(ReevalGrace)
			break
		}
		if u == stat.TypeHp {
			dirtyAt = p.now()
		}
	}
	if dirtyAt.IsZero() {
		return nil
	}
	return GetRegistry().MarkDirty(p.ctx, characterId, dirtyAt)
}

// HandleTransfer covers MAP_CHANGED and CHANNEL_CHANGED (PRD FR-2: channel/map
// transfer re-checks berserk state).
func (p *ProcessorImpl) HandleTransfer(worldId world.Id, channelId channel.Id, characterId uint32) error {
	if err := GetRegistry().UpdateChannel(p.ctx, characterId, worldId, channelId); err != nil {
		return err
	}
	return GetRegistry().MarkDirty(p.ctx, characterId, p.now())
}

// HandleSkillUpdated tracks SP allocation into Berserk without a relog. New
// entries have no channel (the skill event carries none); the next
// channel-bearing character event fills it in (design D8).
func (p *ProcessorImpl) HandleSkillUpdated(worldId world.Id, characterId uint32, level byte) error {
	if level == 0 {
		return p.Untrack(characterId)
	}
	err := GetRegistry().UpdateSkillLevel(p.ctx, characterId, level)
	if errors.Is(err, ErrNotFound) {
		p.l.Infof("Tracking berserk for character [%d] at skill level [%d] (skill update).", characterId, level)
		return GetRegistry().Track(p.ctx,
			NewBuilder(worldId, characterId, level).SetDirtyAt(p.now()).Build())
	}
	if err != nil {
		return err
	}
	return GetRegistry().MarkDirty(p.ctx, characterId, p.now())
}

// MarkMaxHpDirty is the in-process hook for buff apply/expire/cancel whose
// stat-ups affect max HP (Hyper Body). Grace-deferred: atlas-buffs is the
// producer of the very event effective-stats consumes to recompute max HP
// (design D5).
func (p *ProcessorImpl) MarkMaxHpDirty(characterId uint32) error {
	return GetRegistry().MarkDirty(p.ctx, characterId, p.now().Add(ReevalGrace))
}

// ProcessTicks is one scan pass for one tenant: claim due re-evaluations
// (2 REST reads each), else claim due broadcasts and emit. Claims are atomic
// across replicas — at most one emitter per deadline (design D2).
func (p *ProcessorImpl) ProcessTicks() error {
	now := p.now()
	entries := GetRegistry().GetAll(p.ctx)

	return message.Emit(p.l, p.ctx)(func(buf *message.Buffer) error {
		for _, e := range entries {
			if e.DirtyDue(now) {
				if m, ok := GetRegistry().ClaimReeval(p.ctx, e.CharacterId(), now); ok {
					p.reevaluate(m, now)
				}
			}
			if e.BroadcastDue(now) {
				if m, ok := GetRegistry().ClaimBroadcast(p.ctx, e.CharacterId(), now); ok {
					if err := buf.Put(character2.EnvEventStatusTopic, berserkStatusEventProvider(uuid.New(), m)); err != nil {
						return err
					}
				}
			}
		}
		return nil
	})
}

// reevaluate runs the FR-1 computation for a claimed entry. Any lookup
// failure warns and re-arms dirtyAt so a later pass retries; the existing
// schedule keeps broadcasting the last-known state meanwhile (FR-5).
func (p *ProcessorImpl) reevaluate(m Model, now time.Time) {
	rearm := func(reason string, err error) {
		p.l.WithError(err).Warnf("Berserk re-evaluation for character [%d] failed (%s); retrying.", m.CharacterId(), reason)
		if err := GetRegistry().MarkDirty(p.ctx, m.CharacterId(), now.Add(ReevalRetryDelay)); err != nil {
			p.l.WithError(err).Warnf("Unable to re-arm berserk re-evaluation for character [%d].", m.CharacterId())
		}
	}

	x, err := p.getEffectX(m.SkillLevel())
	if err != nil {
		rearm("effect data", err)
		return
	}
	c, err := p.getCharacter(m.CharacterId())
	if err != nil {
		rearm("character", err)
		return
	}
	maxHp, err := p.getMaxHp(m.WorldId(), m.ChannelId(), m.CharacterId())
	if err != nil {
		rearm("effective stats", err)
		return
	}
	if maxHp == 0 {
		rearm("effective stats", errors.New("effective max HP is zero"))
		return
	}

	active := Evaluate(m.SkillLevel(), c.Hp, maxHp, x)
	if active != m.Active() {
		p.l.Debugf("Berserk state for character [%d] now [%v] (hp [%d] maxHp [%d] x [%d]).", m.CharacterId(), active, c.Hp, maxHp, x)
	}
	if err := GetRegistry().StoreEvaluation(p.ctx, m.CharacterId(), active, c.Level, now); err != nil {
		p.l.WithError(err).Warnf("Unable to store berserk evaluation for character [%d].", m.CharacterId())
	}
}

// ProcessBerserkTicks fans out one ProcessTicks per tenant (ticker entry
// point; same shape as character.ProcessPoisonTicks, character/processor.go).
func ProcessBerserkTicks(l logrus.FieldLogger, ctx context.Context) error {
	ts, err := GetRegistry().GetTenants(ctx)
	if err != nil {
		return err
	}

	for _, t := range ts {
		routine.Go(l, ctx, func(_ context.Context) {
			tctx := tenant.WithContext(ctx, t)
			if err := NewProcessor(l, tctx).ProcessTicks(); err != nil {
				l.WithError(err).Error("Failed to process berserk ticks for tenant.")
			}
		})
	}
	return nil
}
