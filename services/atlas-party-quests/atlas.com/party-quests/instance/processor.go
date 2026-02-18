package instance

import (
	"atlas-party-quests/condition"
	"atlas-party-quests/definition"
	"atlas-party-quests/guild"
	"atlas-party-quests/kafka/message"
	character2 "atlas-party-quests/kafka/message/character"
	pq "atlas-party-quests/kafka/message/party_quest"
	reactorMessage "atlas-party-quests/kafka/message/reactor"
	systemMessage "atlas-party-quests/kafka/message/system_message"
	"atlas-party-quests/kafka/producer"
	"atlas-party-quests/monster"
	"atlas-party-quests/party"
	"atlas-party-quests/reward"
	"atlas-party-quests/stage"
	"context"
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/field"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type Processor interface {
	Register(mb *message.Buffer) func(questId string, partyId uint32, channelId channel.Id, mapId uint32, characters []CharacterEntry) (Model, error)
	RegisterAndEmit(questId string, partyId uint32, channelId channel.Id, mapId uint32, characters []CharacterEntry) (Model, error)

	Start(mb *message.Buffer) func(instanceId uuid.UUID) error
	StartAndEmit(instanceId uuid.UUID) error

	StageClearAttempt(mb *message.Buffer) func(instanceId uuid.UUID) error
	StageClearAttemptAndEmit(instanceId uuid.UUID) error

	StageAdvance(mb *message.Buffer) func(instanceId uuid.UUID) error
	StageAdvanceAndEmit(instanceId uuid.UUID) error

	ForceStageComplete(mb *message.Buffer) func(instanceId uuid.UUID) error
	ForceStageCompleteAndEmit(instanceId uuid.UUID) error

	Forfeit(mb *message.Buffer) func(instanceId uuid.UUID) error
	ForfeitAndEmit(instanceId uuid.UUID) error

	Leave(mb *message.Buffer) func(characterId uint32, reason string) error
	LeaveAndEmit(characterId uint32, reason string) error

	UpdateStageState(instanceId uuid.UUID, itemCounts map[uint32]uint32, monsterKills map[uint32]uint32) error
	UpdateCustomData(instanceId uuid.UUID, updates map[string]string, increments []string) error

	BroadcastMessage(mb *message.Buffer) func(instanceId uuid.UUID, messageType string, msg string) error
	BroadcastMessageAndEmit(instanceId uuid.UUID, messageType string, msg string) error

	HandleFriendlyMonsterDamaged(mb *message.Buffer) func(f field.Model, monsterId uint32) error
	HandleFriendlyMonsterDamagedAndEmit(f field.Model, monsterId uint32) error

	HandleFriendlyMonsterKilled(mb *message.Buffer) func(f field.Model, monsterId uint32) error
	HandleFriendlyMonsterKilledAndEmit(f field.Model, monsterId uint32) error

	HandleFriendlyMonsterDrop(mb *message.Buffer) func(f field.Model, monsterId uint32, itemCount uint32) error
	HandleFriendlyMonsterDropAndEmit(f field.Model, monsterId uint32, itemCount uint32) error

	GetByFieldInstance(fieldInstance uuid.UUID) (Model, error)

	EnterBonus(mb *message.Buffer) func(instanceId uuid.UUID) error
	EnterBonusAndEmit(instanceId uuid.UUID) error

	Destroy(mb *message.Buffer) func(instanceId uuid.UUID, reason string) error
	DestroyAndEmit(instanceId uuid.UUID, reason string) error

	TickGlobalTimer(mb *message.Buffer) error
	TickGlobalTimerAndEmit() error

	TickStageTimer(mb *message.Buffer) error
	TickStageTimerAndEmit() error

	TickBonusTimer(mb *message.Buffer) error
	TickBonusTimerAndEmit() error

	TickCompletionTimer(mb *message.Buffer) error
	TickCompletionTimerAndEmit() error

	TickRegistrationTimer(mb *message.Buffer) error
	TickRegistrationTimerAndEmit() error

	GracefulShutdown(mb *message.Buffer) error
	GracefulShutdownAndEmit() error

	GetById(instanceId uuid.UUID) (Model, error)
	GetByCharacter(characterId uint32) (Model, error)
	GetTimerByCharacter(characterId uint32) (uint64, error)
	GetAll() []Model
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	t   tenant.Model
	p   producer.Provider
	db  *gorm.DB
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) Processor {
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
		t:   tenant.MustFromContext(ctx),
		p:   producer.ProviderImpl(l)(ctx),
		db:  db,
	}
}

func (p *ProcessorImpl) GetById(instanceId uuid.UUID) (Model, error) {
	return GetRegistry().Get(p.t, instanceId)
}

func (p *ProcessorImpl) GetByCharacter(characterId uint32) (Model, error) {
	return GetRegistry().GetByCharacter(p.t, characterId)
}

func (p *ProcessorImpl) GetTimerByCharacter(characterId uint32) (uint64, error) {
	inst, err := GetRegistry().GetByCharacter(p.t, characterId)
	if err != nil {
		return 0, err
	}

	def, err := definition.NewProcessor(p.l, p.ctx, p.db).ByIdProvider(inst.DefinitionId())()
	if err != nil {
		return 0, err
	}

	now := time.Now()

	// Bonus state: return remaining bonus time.
	if inst.State() == StateBonus {
		bonus := def.Bonus()
		if bonus != nil && bonus.Duration() > 0 {
			elapsed := now.Sub(inst.StageStartedAt())
			remaining := int64(bonus.Duration()) - int64(elapsed.Seconds())
			if remaining < 0 {
				remaining = 0
			}
			return uint64(remaining), nil
		}
		return 0, errors.New("no bonus timer configured")
	}

	// Completed state: no player-facing timer. Completion timeout is internal only.
	if inst.State() == StateCompleted {
		return 0, nil
	}

	if inst.State() != StateActive {
		return 0, errors.New("instance not active")
	}

	// Stage timer takes precedence over global timer.
	stageIdx := inst.CurrentStageIndex()
	if int(stageIdx) < len(def.Stages()) {
		stg := def.Stages()[stageIdx]
		if stg.Duration() > 0 {
			elapsed := now.Sub(inst.StageStartedAt())
			remaining := int64(stg.Duration()) - int64(elapsed.Seconds())
			if remaining < 0 {
				remaining = 0
			}
			return uint64(remaining), nil
		}
	}

	// Fall back to global timer.
	if def.Duration() > 0 {
		elapsed := now.Sub(inst.StartedAt())
		remaining := int64(def.Duration()) - int64(elapsed.Seconds())
		if remaining < 0 {
			remaining = 0
		}
		return uint64(remaining), nil
	}

	return 0, errors.New("no timer configured")
}

func (p *ProcessorImpl) GetAll() []Model {
	return GetRegistry().GetAll(p.t)
}

func (p *ProcessorImpl) RegisterAndEmit(questId string, partyId uint32, channelId channel.Id, mapId uint32, characters []CharacterEntry) (Model, error) {
	var inst Model
	err := message.Emit(p.p)(func(buf *message.Buffer) error {
		var err error
		inst, err = p.Register(buf)(questId, partyId, channelId, mapId, characters)
		return err
	})
	return inst, err
}

func (p *ProcessorImpl) Register(mb *message.Buffer) func(questId string, partyId uint32, channelId channel.Id, mapId uint32, characters []CharacterEntry) (Model, error) {
	return func(questId string, partyId uint32, channelId channel.Id, mapId uint32, characters []CharacterEntry) (Model, error) {
		def, err := definition.NewProcessor(p.l, p.ctx, p.db).ByQuestIdProvider(questId)()
		if err != nil {
			p.l.WithError(err).Errorf("PQ definition [%s] not found.", questId)
			return Model{}, err
		}

		if len(characters) == 0 {
			return Model{}, errors.New("at least one character required")
		}

		reg := def.Registration()

		switch reg.Type() {
		case "party":
			return p.registerParty(mb, def, questId, partyId, channelId, characters)
		case "individual":
			return p.registerIndividual(mb, def, questId, characters[0].WorldId(), channelId, mapId, characters[0])
		default:
			return p.registerParty(mb, def, questId, partyId, channelId, characters)
		}
	}
}

func (p *ProcessorImpl) registerParty(mb *message.Buffer, def definition.Model, questId string, partyId uint32, channelId channel.Id, characters []CharacterEntry) (Model, error) {
	reg := def.Registration()

	// Resolve all party members via cross-service REST call.
	if partyId > 0 {
		members, err := party.NewProcessor(p.l, p.ctx).GetMembers(partyId)
		if err != nil {
			p.l.WithError(err).Errorf("Failed to resolve party [%d] members.", partyId)
			return Model{}, err
		}
		if len(members) == 0 {
			return Model{}, errors.New("party has no members")
		}
		characters = make([]CharacterEntry, 0, len(members))
		for _, m := range members {
			characters = append(characters, NewCharacterEntry(m.Id(), m.WorldId(), m.ChannelId()))
		}
	}

	worldId := characters[0].WorldId()
	if channelId == 0 {
		channelId = characters[0].ChannelId()
	}

	inst, err := NewBuilder().
		SetTenantId(p.t.Id()).
		SetDefinitionId(def.Id()).
		SetQuestId(questId).
		SetWorldId(worldId).
		SetChannelId(channelId).
		SetPartyId(partyId).
		SetAffinityId(partyId).
		SetCharacters(characters).
		Build()
	if err != nil {
		return Model{}, err
	}

	inst = inst.SetState(StateRegistering)
	inst = GetRegistry().Create(p.t, inst)

	p.l.Infof("PQ instance [%s] created for quest [%s], party [%d], characters: %d.",
		inst.Id(), questId, partyId, len(characters))

	err = mb.Put(pq.EnvEventStatusTopic, instanceCreatedEventProvider(worldId, inst.Id(), questId, partyId, channelId))
	if err != nil {
		return Model{}, err
	}

	if reg.Mode() == "instant" {
		return inst, p.Start(mb)(inst.Id())
	}

	if reg.Mode() == "timed" && reg.Duration() > 0 {
		err = mb.Put(pq.EnvEventStatusTopic, registrationOpenedEventProvider(worldId, inst.Id(), questId, reg.Duration()))
		if err != nil {
			return Model{}, err
		}
	}

	return inst, nil
}

func (p *ProcessorImpl) registerIndividual(mb *message.Buffer, def definition.Model, questId string, worldId world.Id, channelId channel.Id, mapId uint32, character CharacterEntry) (Model, error) {
	reg := def.Registration()

	// Validate registration map.
	if reg.MapId() != 0 && mapId != 0 && reg.MapId() != mapId {
		return Model{}, errors.New("character is not on the registration map")
	}

	// Resolve affinity for this character.
	affinityId, err := p.resolveAffinity(reg.Affinity(), character.CharacterId())
	if err != nil {
		p.l.WithError(err).Errorf("Failed to resolve affinity [%s] for character [%d].", reg.Affinity(), character.CharacterId())
		return Model{}, err
	}

	// Check for an existing registering instance to join.
	existing, found := p.findRegistering(questId, worldId, channelId, affinityId)
	if found {
		// Check for duplicate character.
		for _, c := range existing.Characters() {
			if c.CharacterId() == character.CharacterId() {
				p.l.Infof("Character [%d] already registered in PQ instance [%s].", character.CharacterId(), existing.Id())
				return existing, nil
			}
		}

		updated, err := GetRegistry().Update(p.t, existing.Id(), func(m Model) Model {
			return m.AddCharacter(character)
		})
		if err != nil {
			return Model{}, err
		}

		p.l.Infof("Character [%d] joined registering PQ instance [%s] for quest [%s].",
			character.CharacterId(), existing.Id(), questId)

		err = mb.Put(pq.EnvEventStatusTopic, characterRegisteredEventProvider(worldId, existing.Id(), questId, character.CharacterId()))
		if err != nil {
			return Model{}, err
		}

		return updated, nil
	}

	// No existing instance — create a new one.
	inst, err := NewBuilder().
		SetTenantId(p.t.Id()).
		SetDefinitionId(def.Id()).
		SetQuestId(questId).
		SetWorldId(worldId).
		SetChannelId(channelId).
		SetPartyId(0).
		SetAffinityId(affinityId).
		SetCharacters([]CharacterEntry{character}).
		Build()
	if err != nil {
		return Model{}, err
	}

	inst = inst.SetState(StateRegistering)
	inst = GetRegistry().Create(p.t, inst)

	p.l.Infof("PQ instance [%s] created for quest [%s], individual registration, affinity [%d].",
		inst.Id(), questId, affinityId)

	err = mb.Put(pq.EnvEventStatusTopic, instanceCreatedEventProvider(worldId, inst.Id(), questId, 0, channelId))
	if err != nil {
		return Model{}, err
	}

	if reg.Mode() == "instant" {
		return inst, p.Start(mb)(inst.Id())
	}

	if reg.Mode() == "timed" && reg.Duration() > 0 {
		err = mb.Put(pq.EnvEventStatusTopic, registrationOpenedEventProvider(worldId, inst.Id(), questId, reg.Duration()))
		if err != nil {
			return Model{}, err
		}
	}

	return inst, nil
}

func (p *ProcessorImpl) findRegistering(questId string, worldId world.Id, channelId channel.Id, affinityId uint32) (Model, bool) {
	for _, inst := range GetRegistry().GetAll(p.t) {
		if inst.State() != StateRegistering {
			continue
		}
		if inst.QuestId() != questId {
			continue
		}
		if inst.WorldId() != worldId {
			continue
		}
		if channelId != 0 && inst.ChannelId() != channelId {
			continue
		}
		if affinityId != 0 && inst.AffinityId() != affinityId {
			continue
		}
		return inst, true
	}
	return Model{}, false
}

func (p *ProcessorImpl) resolveAffinity(affinityType string, characterId uint32) (uint32, error) {
	switch affinityType {
	case "guild":
		g, err := guild.NewProcessor(p.l, p.ctx).GetByMemberId(characterId)
		if err != nil {
			return 0, err
		}
		return g.Id(), nil
	case "party":
		pa, err := party.NewProcessor(p.l, p.ctx).GetByMemberId(characterId)
		if err != nil {
			return 0, err
		}
		return pa.Id(), nil
	case "none", "":
		return 0, nil
	default:
		return 0, errors.New("unknown affinity type: " + affinityType)
	}
}

func (p *ProcessorImpl) StartAndEmit(instanceId uuid.UUID) error {
	return message.Emit(p.p)(func(buf *message.Buffer) error {
		return p.Start(buf)(instanceId)
	})
}

func (p *ProcessorImpl) Start(mb *message.Buffer) func(instanceId uuid.UUID) error {
	return func(instanceId uuid.UUID) error {
		inst, err := GetRegistry().Get(p.t, instanceId)
		if err != nil {
			return err
		}

		def, err := definition.NewProcessor(p.l, p.ctx, p.db).ByIdProvider(inst.DefinitionId())()
		if err != nil {
			return err
		}

		if len(def.Stages()) == 0 {
			return errors.New("definition has no stages")
		}

		now := time.Now()
		stg := def.Stages()[0]

		// Generate initial stage state
		ss := NewStageState()
		if stg.Type() == stage.TypeCombinationPuzzle {
			ss = ss.WithCombination(generateCombination(stg.Properties()))
		}

		// Update instance state
		_, err = GetRegistry().Update(p.t, instanceId, func(m Model) Model {
			return m.
				SetState(StateActive).
				SetStartedAt(now).
				SetStageStartedAt(now).
				SetCurrentStageIndex(0).
				SetStageState(ss)
		})
		if err != nil {
			return err
		}

		p.l.Infof("PQ instance [%s] started for quest [%s], stage 0.", instanceId, inst.QuestId())

		// Warp characters to stage 0 maps
		if len(stg.MapIds()) > 0 {
			targetMapId := _map.Id(stg.MapIds()[0])
			for _, c := range inst.Characters() {
				err = mb.Put(character2.EnvCommandTopic, warpCharacterProvider(c.WorldId(), c.ChannelId(), c.CharacterId(), targetMapId, inst.Id()))
				if err != nil {
					p.l.WithError(err).Errorf("Failed to warp character [%d] to map [%d].", c.CharacterId(), targetMapId)
				}
			}
		}

		// Spawn friendly monster if configured for this stage
		p.spawnFriendlyMonster(mb, stg, inst)

		// Emit STARTED event
		return mb.Put(pq.EnvEventStatusTopic, startedEventProvider(inst.WorldId(), instanceId, inst.QuestId(), 0, stg.MapIds()))
	}
}

func (p *ProcessorImpl) StageClearAttemptAndEmit(instanceId uuid.UUID) error {
	return message.Emit(p.p)(func(buf *message.Buffer) error {
		return p.StageClearAttempt(buf)(instanceId)
	})
}

func (p *ProcessorImpl) StageClearAttempt(mb *message.Buffer) func(instanceId uuid.UUID) error {
	return func(instanceId uuid.UUID) error {
		inst, err := GetRegistry().Get(p.t, instanceId)
		if err != nil {
			return err
		}

		if inst.State() != StateActive {
			return errors.New("instance not active")
		}

		def, err := definition.NewProcessor(p.l, p.ctx, p.db).ByIdProvider(inst.DefinitionId())()
		if err != nil {
			return err
		}

		stageIdx := inst.CurrentStageIndex()
		if int(stageIdx) >= len(def.Stages()) {
			return errors.New("invalid stage index")
		}

		stg := def.Stages()[stageIdx]

		// Evaluate clear conditions
		if !evaluateClearConditions(stg.ClearConditions(), inst.StageState()) {
			p.l.Debugf("PQ instance [%s] stage [%d] clear conditions not met.", instanceId, stageIdx)
			return nil
		}

		// Stage cleared
		_, err = GetRegistry().Update(p.t, instanceId, func(m Model) Model {
			return m.SetState(StateClearing)
		})
		if err != nil {
			return err
		}

		p.l.Infof("PQ instance [%s] stage [%d] cleared.", instanceId, stageIdx)

		// Execute clear actions
		p.executeClearActions(stg, inst)

		// Distribute stage rewards
		p.emitExperienceRewards(mb, inst, stg.Rewards())

		// Emit STAGE_CLEARED event
		err = mb.Put(pq.EnvEventStatusTopic, stageClearedEventProvider(inst.WorldId(), instanceId, inst.QuestId(), stageIdx, inst.ChannelId(), stg.MapIds(), []uuid.UUID{inst.Id()}))
		if err != nil {
			return err
		}

		// Auto-advance to the next stage
		return p.StageAdvance(mb)(instanceId)
	}
}

func (p *ProcessorImpl) executeClearActions(stg stage.Model, inst Model) {
	for _, action := range stg.ClearActions() {
		switch action {
		case "destroy_monsters":
			p.destroyMonstersInStage(stg, inst)
		default:
			p.l.Warnf("Unknown clear action [%s] for stage [%d].", action, stg.Index())
		}
	}
}

func (p *ProcessorImpl) emitExperienceRewards(mb *message.Buffer, inst Model, rewards []reward.Model) {
	for _, r := range rewards {
		if r.Type() != "experience" || r.Amount() == 0 {
			continue
		}
		distributions := []character2.ExperienceDistributions{
			{
				ExperienceType: character2.ExperienceDistributionTypeChat,
				Amount:         r.Amount(),
			},
		}
		for _, c := range inst.Characters() {
			err := mb.Put(character2.EnvCommandTopic, awardExperienceProvider(c.WorldId(), c.ChannelId(), c.CharacterId(), distributions))
			if err != nil {
				p.l.WithError(err).Errorf("Failed to award experience to character [%d].", c.CharacterId())
			}
		}
	}
}

func (p *ProcessorImpl) emitDestroyReactorsInField(mb *message.Buffer, inst Model, mapIds []uint32) {
	for _, mid := range mapIds {
		err := mb.Put(reactorMessage.EnvCommandTopic, destroyReactorsInFieldProvider(inst.WorldId(), inst.ChannelId(), _map.Id(mid), inst.Id()))
		if err != nil {
			p.l.WithError(err).Warnf("Failed to emit destroy reactors for map [%d] instance [%s].", mid, inst.Id())
		}
	}
}

func (p *ProcessorImpl) spawnFriendlyMonster(mb *message.Buffer, stg stage.Model, inst Model) {
	cfg, ok := extractFriendlyMonsterConfig(stg.Properties())
	if !ok {
		return
	}

	if len(stg.MapIds()) == 0 {
		return
	}

	targetMapId := _map.Id(stg.MapIds()[0])
	f := field.NewBuilder(inst.WorldId(), inst.ChannelId(), targetMapId).SetInstance(inst.Id()).Build()

	mp := monster.NewProcessor(p.l, p.ctx)
	err := mp.SpawnInField(f, cfg.monsterId, cfg.x, cfg.y, cfg.fh)
	if err != nil {
		p.l.WithError(err).Errorf("Failed to spawn friendly monster [%d] for PQ instance [%s].", cfg.monsterId, inst.Id())
		return
	}

	if cfg.spawnMessage != "" {
		_ = p.BroadcastMessage(mb)(inst.Id(), "BLUE_TEXT", cfg.spawnMessage)
	}

	p.l.Infof("Spawned friendly monster [%d] for PQ instance [%s] in field [%s].", cfg.monsterId, inst.Id(), f.Id())
}

func (p *ProcessorImpl) destroyMonstersInStage(stg stage.Model, inst Model) {
	mp := monster.NewProcessor(p.l, p.ctx)
	for _, mid := range stg.MapIds() {
		err := mp.DestroyInField(inst.WorldId(), inst.ChannelId(), _map.Id(mid), inst.Id())
		if err != nil {
			p.l.WithError(err).Warnf("Failed to destroy monsters in map [%d] instance [%s].", mid, inst.Id())
		}
	}
}

func (p *ProcessorImpl) StageAdvanceAndEmit(instanceId uuid.UUID) error {
	return message.Emit(p.p)(func(buf *message.Buffer) error {
		return p.StageAdvance(buf)(instanceId)
	})
}

func (p *ProcessorImpl) StageAdvance(mb *message.Buffer) func(instanceId uuid.UUID) error {
	return func(instanceId uuid.UUID) error {
		inst, err := GetRegistry().Get(p.t, instanceId)
		if err != nil {
			return err
		}

		def, err := definition.NewProcessor(p.l, p.ctx, p.db).ByIdProvider(inst.DefinitionId())()
		if err != nil {
			return err
		}

		currentStageIdx := inst.CurrentStageIndex()
		nextStageIdx := currentStageIdx + 1

		// Destroy monsters in the current stage maps before advancing
		if int(currentStageIdx) < len(def.Stages()) {
			p.destroyMonstersInStage(def.Stages()[currentStageIdx], inst)
		}

		// Check if PQ is complete (no more stages)
		if int(nextStageIdx) >= len(def.Stages()) {
			return p.complete(mb, inst, def)
		}

		nextStage := def.Stages()[nextStageIdx]
		now := time.Now()

		// Generate new stage state
		ss := NewStageState()
		if nextStage.Type() == stage.TypeCombinationPuzzle {
			ss = ss.WithCombination(generateCombination(nextStage.Properties()))
		}

		// Update instance
		_, err = GetRegistry().Update(p.t, instanceId, func(m Model) Model {
			return m.
				SetState(StateActive).
				SetCurrentStageIndex(nextStageIdx).
				SetStageStartedAt(now).
				SetStageState(ss)
		})
		if err != nil {
			return err
		}

		p.l.Infof("PQ instance [%s] advanced to stage [%d].", instanceId, nextStageIdx)

		// Warp characters to new stage maps (skip if current stage warpType is "none")
		currentStage := def.Stages()[currentStageIdx]
		if currentStage.WarpType() != "none" && len(nextStage.MapIds()) > 0 {
			targetMapId := _map.Id(nextStage.MapIds()[0])
			for _, c := range inst.Characters() {
				err = mb.Put(character2.EnvCommandTopic, warpCharacterProvider(c.WorldId(), c.ChannelId(), c.CharacterId(), targetMapId, inst.Id()))
				if err != nil {
					p.l.WithError(err).Errorf("Failed to warp character [%d] to map [%d].", c.CharacterId(), targetMapId)
				}
			}
		}

		// Spawn friendly monster if configured for the new stage
		p.spawnFriendlyMonster(mb, nextStage, inst)

		// Emit STAGE_ADVANCED event
		return mb.Put(pq.EnvEventStatusTopic, stageAdvancedEventProvider(inst.WorldId(), instanceId, inst.QuestId(), nextStageIdx, nextStage.MapIds()))
	}
}

func (p *ProcessorImpl) ForceStageCompleteAndEmit(instanceId uuid.UUID) error {
	return message.Emit(p.p)(func(buf *message.Buffer) error {
		return p.ForceStageComplete(buf)(instanceId)
	})
}

func (p *ProcessorImpl) ForceStageComplete(mb *message.Buffer) func(instanceId uuid.UUID) error {
	return func(instanceId uuid.UUID) error {
		inst, err := GetRegistry().Get(p.t, instanceId)
		if err != nil {
			return err
		}

		if inst.State() != StateActive {
			return errors.New("instance not active")
		}

		def, err := definition.NewProcessor(p.l, p.ctx, p.db).ByIdProvider(inst.DefinitionId())()
		if err != nil {
			return err
		}

		stageIdx := inst.CurrentStageIndex()
		if int(stageIdx) >= len(def.Stages()) {
			return errors.New("invalid stage index")
		}

		stg := def.Stages()[stageIdx]

		_, err = GetRegistry().Update(p.t, instanceId, func(m Model) Model {
			return m.SetState(StateClearing)
		})
		if err != nil {
			return err
		}

		p.l.Infof("PQ instance [%s] stage [%d] force cleared.", instanceId, stageIdx)

		p.executeClearActions(stg, inst)
		p.emitExperienceRewards(mb, inst, stg.Rewards())

		err = mb.Put(pq.EnvEventStatusTopic, stageClearedEventProvider(inst.WorldId(), instanceId, inst.QuestId(), stageIdx, inst.ChannelId(), stg.MapIds(), []uuid.UUID{inst.Id()}))
		if err != nil {
			return err
		}

		return p.StageAdvance(mb)(instanceId)
	}
}

func (p *ProcessorImpl) complete(mb *message.Buffer, inst Model, def definition.Model) error {
	_, err := GetRegistry().Update(p.t, inst.Id(), func(m Model) Model {
		return m.SetState(StateCompleted)
	})
	if err != nil {
		return err
	}

	p.l.Infof("PQ instance [%s] completed.", inst.Id())

	// Distribute completion rewards
	p.emitExperienceRewards(mb, inst, def.Rewards())

	// Emit COMPLETED event
	err = mb.Put(pq.EnvEventStatusTopic, completedEventProvider(inst.WorldId(), inst.Id(), inst.QuestId()))
	if err != nil {
		return err
	}

	// Destroy reactors in current stage maps
	stageIdx := inst.CurrentStageIndex()
	if int(stageIdx) < len(def.Stages()) {
		p.emitDestroyReactorsInField(mb, inst, def.Stages()[stageIdx].MapIds())
	}

	bonus := def.Bonus()
	if bonus == nil {
		// No bonus configured, destroy instance.
		return p.Destroy(mb)(inst.Id(), "completed")
	}

	switch bonus.Entry() {
	case definition.BonusEntryAuto:
		// Auto-enter bonus immediately.
		return p.enterBonusInternal(mb, inst, def)

	case definition.BonusEntryManual:
		// Warp to completion map if configured.
		if bonus.CompletionMapId() != 0 {
			targetMapId := _map.Id(bonus.CompletionMapId())
			for _, c := range inst.Characters() {
				_ = mb.Put(character2.EnvCommandTopic, warpCharacterProvider(c.WorldId(), c.ChannelId(), c.CharacterId(), targetMapId, inst.Id()))
			}
		}
		// Remain in StateCompleted. NPC/portal calls EnterBonus or Leave.
		// Track when completion phase began for timeout.
		_, err = GetRegistry().Update(p.t, inst.Id(), func(m Model) Model {
			return m.SetStageStartedAt(time.Now())
		})
		return err

	default:
		return p.Destroy(mb)(inst.Id(), "completed")
	}
}

func (p *ProcessorImpl) EnterBonusAndEmit(instanceId uuid.UUID) error {
	return message.Emit(p.p)(func(buf *message.Buffer) error {
		return p.EnterBonus(buf)(instanceId)
	})
}

func (p *ProcessorImpl) EnterBonus(mb *message.Buffer) func(instanceId uuid.UUID) error {
	return func(instanceId uuid.UUID) error {
		inst, err := GetRegistry().Get(p.t, instanceId)
		if err != nil {
			return err
		}

		if inst.State() != StateCompleted {
			return errors.New("instance not in completed state, cannot enter bonus")
		}

		def, err := definition.NewProcessor(p.l, p.ctx, p.db).ByIdProvider(inst.DefinitionId())()
		if err != nil {
			return err
		}

		if def.Bonus() == nil {
			return errors.New("definition has no bonus configured")
		}

		return p.enterBonusInternal(mb, inst, def)
	}
}

func (p *ProcessorImpl) enterBonusInternal(mb *message.Buffer, inst Model, def definition.Model) error {
	bonus := def.Bonus()
	now := time.Now()

	_, err := GetRegistry().Update(p.t, inst.Id(), func(m Model) Model {
		return m.
			SetState(StateBonus).
			SetStageStartedAt(now).
			SetStageState(NewStageState())
	})
	if err != nil {
		return err
	}

	p.l.Infof("PQ instance [%s] entered bonus stage.", inst.Id())

	// Warp characters to bonus map.
	if bonus.MapId() != 0 {
		targetMapId := _map.Id(bonus.MapId())
		for _, c := range inst.Characters() {
			_ = mb.Put(character2.EnvCommandTopic, warpCharacterProvider(c.WorldId(), c.ChannelId(), c.CharacterId(), targetMapId, inst.Id()))
		}
	}

	// Emit BONUS_ENTERED event.
	return mb.Put(pq.EnvEventStatusTopic, bonusEnteredEventProvider(inst.WorldId(), inst.Id(), inst.QuestId(), bonus.MapId()))
}

func (p *ProcessorImpl) ForfeitAndEmit(instanceId uuid.UUID) error {
	return message.Emit(p.p)(func(buf *message.Buffer) error {
		return p.Forfeit(buf)(instanceId)
	})
}

func (p *ProcessorImpl) Forfeit(mb *message.Buffer) func(instanceId uuid.UUID) error {
	return func(instanceId uuid.UUID) error {
		inst, err := GetRegistry().Get(p.t, instanceId)
		if err != nil {
			return err
		}

		_, err = GetRegistry().Update(p.t, instanceId, func(m Model) Model {
			return m.SetState(StateFailed)
		})
		if err != nil {
			return err
		}

		p.l.Infof("PQ instance [%s] forfeited.", instanceId)

		// Emit FAILED event
		err = mb.Put(pq.EnvEventStatusTopic, failedEventProvider(inst.WorldId(), instanceId, inst.QuestId(), "forfeit"))
		if err != nil {
			return err
		}

		// Destroy reactors in current stage maps
		def, defErr := definition.NewProcessor(p.l, p.ctx, p.db).ByIdProvider(inst.DefinitionId())()
		if defErr == nil {
			stageIdx := inst.CurrentStageIndex()
			if int(stageIdx) < len(def.Stages()) {
				p.emitDestroyReactorsInField(mb, inst, def.Stages()[stageIdx].MapIds())
			}
		}

		return p.Destroy(mb)(instanceId, "forfeit")
	}
}

func (p *ProcessorImpl) LeaveAndEmit(characterId uint32, reason string) error {
	return message.Emit(p.p)(func(buf *message.Buffer) error {
		return p.Leave(buf)(characterId, reason)
	})
}

func (p *ProcessorImpl) Leave(mb *message.Buffer) func(characterId uint32, reason string) error {
	return func(characterId uint32, reason string) error {
		inst, err := GetRegistry().GetByCharacter(p.t, characterId)
		if err != nil {
			return err
		}

		if inst.State() != StateActive && inst.State() != StateClearing && inst.State() != StateCompleted && inst.State() != StateBonus {
			return errors.New("instance not in a leavable state")
		}

		def, err := definition.NewProcessor(p.l, p.ctx, p.db).ByIdProvider(inst.DefinitionId())()
		if err != nil {
			p.l.WithError(err).Warnf("Failed to load definition for instance [%s], using exit map 0.", inst.Id())
		}

		exitMap := _map.Id(def.Exit())

		// Find the character entry for warp details.
		var ce CharacterEntry
		for _, c := range inst.Characters() {
			if c.CharacterId() == characterId {
				ce = c
				break
			}
		}

		// Remove character from instance.
		_, err = GetRegistry().Update(p.t, inst.Id(), func(m Model) Model {
			return m.RemoveCharacter(characterId)
		})
		if err != nil {
			return err
		}

		// Warp the leaving character to exit map.
		err = mb.Put(character2.EnvCommandTopic, warpCharacterProvider(ce.WorldId(), ce.ChannelId(), characterId, exitMap, uuid.Nil))
		if err != nil {
			p.l.WithError(err).Errorf("Failed to warp character [%d] to exit map.", characterId)
		}

		p.l.Infof("Character [%d] left PQ instance [%s]. Reason: %s.", characterId, inst.Id(), reason)

		// Emit CHARACTER_LEFT event.
		err = mb.Put(pq.EnvEventStatusTopic, characterLeftEventProvider(inst.WorldId(), inst.Id(), inst.QuestId(), characterId, ce.ChannelId(), reason))
		if err != nil {
			return err
		}

		// If no characters remain, destroy the instance.
		updated, err := GetRegistry().Get(p.t, inst.Id())
		if err != nil {
			return err
		}
		if len(updated.Characters()) == 0 {
			p.l.Infof("PQ instance [%s] has no remaining characters, destroying.", inst.Id())
			return p.Destroy(mb)(inst.Id(), "empty")
		}

		return nil
	}
}

func (p *ProcessorImpl) UpdateStageState(instanceId uuid.UUID, itemCounts map[uint32]uint32, monsterKills map[uint32]uint32) error {
	_, err := GetRegistry().Update(p.t, instanceId, func(m Model) Model {
		ss := m.StageState()
		for k, v := range itemCounts {
			ss = ss.WithItemCount(k, v)
		}
		for k, v := range monsterKills {
			ss = ss.WithMonsterKill(k, v)
		}
		return m.SetStageState(ss)
	})
	return err
}

func (p *ProcessorImpl) UpdateCustomData(instanceId uuid.UUID, updates map[string]string, increments []string) error {
	_, err := GetRegistry().Update(p.t, instanceId, func(m Model) Model {
		ss := m.StageState()
		for k, v := range updates {
			ss = ss.WithCustomData(k, v)
		}
		for _, k := range increments {
			ss = ss.IncrementCustomData(k)
		}
		return m.SetStageState(ss)
	})
	return err
}

func (p *ProcessorImpl) BroadcastMessageAndEmit(instanceId uuid.UUID, messageType string, msg string) error {
	return message.Emit(p.p)(func(buf *message.Buffer) error {
		return p.BroadcastMessage(buf)(instanceId, messageType, msg)
	})
}

func (p *ProcessorImpl) BroadcastMessage(mb *message.Buffer) func(instanceId uuid.UUID, messageType string, msg string) error {
	return func(instanceId uuid.UUID, messageType string, msg string) error {
		inst, err := GetRegistry().Get(p.t, instanceId)
		if err != nil {
			return err
		}

		for _, c := range inst.Characters() {
			err = mb.Put(systemMessage.EnvCommandTopic, sendMessageProvider(c.WorldId(), c.ChannelId(), c.CharacterId(), messageType, msg))
			if err != nil {
				p.l.WithError(err).Errorf("Failed to send message to character [%d].", c.CharacterId())
			}
		}

		p.l.Infof("Broadcast message to [%d] characters in PQ instance [%s]: %s", len(inst.Characters()), instanceId, msg)
		return nil
	}
}

func (p *ProcessorImpl) GetByFieldInstance(fieldInstance uuid.UUID) (Model, error) {
	return GetRegistry().GetByFieldInstance(p.t, fieldInstance)
}

func (p *ProcessorImpl) DestroyAndEmit(instanceId uuid.UUID, reason string) error {
	return message.Emit(p.p)(func(buf *message.Buffer) error {
		return p.Destroy(buf)(instanceId, reason)
	})
}

func (p *ProcessorImpl) Destroy(mb *message.Buffer) func(instanceId uuid.UUID, reason string) error {
	return func(instanceId uuid.UUID, reason string) error {
		inst, err := GetRegistry().Get(p.t, instanceId)
		if err != nil {
			return err
		}

		def, err := definition.NewProcessor(p.l, p.ctx, p.db).ByIdProvider(inst.DefinitionId())()
		if err != nil {
			p.l.WithError(err).Warnf("Failed to load definition for instance [%s], using exit map 0.", instanceId)
		}

		// Destroy monsters and reactors in the current stage maps
		stageIdx := inst.CurrentStageIndex()
		if int(stageIdx) < len(def.Stages()) {
			p.destroyMonstersInStage(def.Stages()[stageIdx], inst)
			p.emitDestroyReactorsInField(mb, inst, def.Stages()[stageIdx].MapIds())
		}

		// Clean up bonus map if in bonus state.
		if inst.State() == StateBonus && def.Bonus() != nil && def.Bonus().MapId() != 0 {
			p.emitDestroyReactorsInField(mb, inst, []uint32{def.Bonus().MapId()})
		}

		exitMap := _map.Id(def.Exit())

		// Warp all characters to exit map
		for _, c := range inst.Characters() {
			err = mb.Put(character2.EnvCommandTopic, warpCharacterProvider(c.WorldId(), c.ChannelId(), c.CharacterId(), exitMap, uuid.Nil))
			if err != nil {
				p.l.WithError(err).Errorf("Failed to warp character [%d] to exit map.", c.CharacterId())
			}
		}

		// Emit INSTANCE_DESTROYED event
		err = mb.Put(pq.EnvEventStatusTopic, instanceDestroyedEventProvider(inst.WorldId(), instanceId, inst.QuestId()))
		if err != nil {
			p.l.WithError(err).Errorf("Failed to emit instance destroyed event.")
		}

		// Remove from registry
		GetRegistry().Remove(p.t, instanceId)

		p.l.Infof("PQ instance [%s] destroyed. Reason: %s.", instanceId, reason)
		return nil
	}
}

func (p *ProcessorImpl) TickGlobalTimerAndEmit() error {
	return message.Emit(p.p)(func(buf *message.Buffer) error {
		return p.TickGlobalTimer(buf)
	})
}

func (p *ProcessorImpl) TickGlobalTimer(mb *message.Buffer) error {
	now := time.Now()
	for _, inst := range GetRegistry().GetAll(p.t) {
		if inst.State() != StateActive {
			continue
		}

		def, err := definition.NewProcessor(p.l, p.ctx, p.db).ByIdProvider(inst.DefinitionId())()
		if err != nil {
			continue
		}

		if def.Duration() == 0 {
			continue
		}

		elapsed := now.Sub(inst.StartedAt())
		if int64(elapsed.Seconds()) >= int64(def.Duration()) {
			p.l.Infof("PQ instance [%s] global timer expired.", inst.Id())
			_, _ = GetRegistry().Update(p.t, inst.Id(), func(m Model) Model {
				return m.SetState(StateFailed)
			})
			_ = mb.Put(pq.EnvEventStatusTopic, failedEventProvider(inst.WorldId(), inst.Id(), inst.QuestId(), "time_expired"))
			stageIdx := inst.CurrentStageIndex()
			if int(stageIdx) < len(def.Stages()) {
				p.emitDestroyReactorsInField(mb, inst, def.Stages()[stageIdx].MapIds())
			}
			_ = p.Destroy(mb)(inst.Id(), "time_expired")
		}
	}
	return nil
}

func (p *ProcessorImpl) TickStageTimerAndEmit() error {
	return message.Emit(p.p)(func(buf *message.Buffer) error {
		return p.TickStageTimer(buf)
	})
}

func (p *ProcessorImpl) TickStageTimer(mb *message.Buffer) error {
	now := time.Now()
	for _, inst := range GetRegistry().GetAll(p.t) {
		if inst.State() != StateActive {
			continue
		}

		def, err := definition.NewProcessor(p.l, p.ctx, p.db).ByIdProvider(inst.DefinitionId())()
		if err != nil {
			continue
		}

		stageIdx := inst.CurrentStageIndex()
		if int(stageIdx) >= len(def.Stages()) {
			continue
		}

		stg := def.Stages()[stageIdx]
		if stg.Duration() == 0 {
			continue
		}

		elapsed := now.Sub(inst.StageStartedAt())
		if int64(elapsed.Seconds()) >= int64(stg.Duration()) {
			p.l.Infof("PQ instance [%s] stage [%d] timer expired.", inst.Id(), stageIdx)
			// Auto-advance or fail depending on stage type
			_ = p.StageAdvance(mb)(inst.Id())
		}
	}
	return nil
}

func (p *ProcessorImpl) TickBonusTimerAndEmit() error {
	return message.Emit(p.p)(func(buf *message.Buffer) error {
		return p.TickBonusTimer(buf)
	})
}

func (p *ProcessorImpl) TickBonusTimer(mb *message.Buffer) error {
	now := time.Now()
	for _, inst := range GetRegistry().GetAll(p.t) {
		if inst.State() != StateBonus {
			continue
		}

		def, err := definition.NewProcessor(p.l, p.ctx, p.db).ByIdProvider(inst.DefinitionId())()
		if err != nil {
			continue
		}

		bonus := def.Bonus()
		if bonus == nil || bonus.Duration() == 0 {
			continue
		}

		elapsed := now.Sub(inst.StageStartedAt())
		if int64(elapsed.Seconds()) >= int64(bonus.Duration()) {
			p.l.Infof("PQ instance [%s] bonus timer expired.", inst.Id())
			_ = p.Destroy(mb)(inst.Id(), "bonus_expired")
		}
	}
	return nil
}

const completionTimeout = 120 // seconds

func (p *ProcessorImpl) TickCompletionTimerAndEmit() error {
	return message.Emit(p.p)(func(buf *message.Buffer) error {
		return p.TickCompletionTimer(buf)
	})
}

func (p *ProcessorImpl) TickCompletionTimer(mb *message.Buffer) error {
	now := time.Now()
	for _, inst := range GetRegistry().GetAll(p.t) {
		if inst.State() != StateCompleted {
			continue
		}

		elapsed := now.Sub(inst.StageStartedAt())
		if int64(elapsed.Seconds()) >= completionTimeout {
			p.l.Infof("PQ instance [%s] completion timer expired, destroying.", inst.Id())
			_ = p.Destroy(mb)(inst.Id(), "completion_expired")
		}
	}
	return nil
}

func (p *ProcessorImpl) TickRegistrationTimerAndEmit() error {
	return message.Emit(p.p)(func(buf *message.Buffer) error {
		return p.TickRegistrationTimer(buf)
	})
}

func (p *ProcessorImpl) TickRegistrationTimer(mb *message.Buffer) error {
	now := time.Now()
	for _, inst := range GetRegistry().GetAll(p.t) {
		if inst.State() != StateRegistering {
			continue
		}

		def, err := definition.NewProcessor(p.l, p.ctx, p.db).ByIdProvider(inst.DefinitionId())()
		if err != nil {
			continue
		}

		reg := def.Registration()
		if reg.Mode() != "timed" || reg.Duration() <= 0 {
			continue
		}

		elapsed := now.Sub(inst.RegisteredAt())
		if int64(elapsed.Seconds()) >= reg.Duration() {
			p.l.Infof("PQ instance [%s] registration window expired, starting.", inst.Id())
			_ = p.Start(mb)(inst.Id())
		}
	}
	return nil
}

func (p *ProcessorImpl) GracefulShutdownAndEmit() error {
	return message.Emit(p.p)(func(buf *message.Buffer) error {
		return p.GracefulShutdown(buf)
	})
}

func (p *ProcessorImpl) GracefulShutdown(mb *message.Buffer) error {
	instances := GetRegistry().GetAll(p.t)
	for _, inst := range instances {
		p.l.Infof("Graceful shutdown: destroying PQ instance [%s].", inst.Id())
		_ = p.Destroy(mb)(inst.Id(), "shutdown")
	}
	GetRegistry().Clear(p.t)
	return nil
}

type friendlyMonsterConfig struct {
	monsterId       uint32
	x, y            int16
	fh              int16
	spawnMessage    string
	damagedInterval uint32
	damagedMessage  string
	killedMessage   string
	killedAction    string
	dropTemplate    string
}

func extractFriendlyMonsterConfig(properties map[string]any) (friendlyMonsterConfig, bool) {
	fm, ok := properties["friendlyMonster"]
	if !ok {
		return friendlyMonsterConfig{}, false
	}

	fmMap, ok := fm.(map[string]any)
	if !ok {
		return friendlyMonsterConfig{}, false
	}

	cfg := friendlyMonsterConfig{}

	if v, ok := fmMap["monsterId"].(float64); ok {
		cfg.monsterId = uint32(v)
	} else {
		return friendlyMonsterConfig{}, false
	}

	if v, ok := fmMap["x"].(float64); ok {
		cfg.x = int16(v)
	}
	if v, ok := fmMap["y"].(float64); ok {
		cfg.y = int16(v)
	}
	if v, ok := fmMap["fh"].(float64); ok {
		cfg.fh = int16(v)
	}
	if v, ok := fmMap["spawnMessage"].(string); ok {
		cfg.spawnMessage = v
	}

	if onDamaged, ok := fmMap["onDamaged"].(map[string]any); ok {
		if v, ok := onDamaged["hitInterval"].(float64); ok {
			cfg.damagedInterval = uint32(v)
		}
		if v, ok := onDamaged["message"].(string); ok {
			cfg.damagedMessage = v
		}
	}

	if onKilled, ok := fmMap["onKilled"].(map[string]any); ok {
		if v, ok := onKilled["message"].(string); ok {
			cfg.killedMessage = v
		}
		if v, ok := onKilled["action"].(string); ok {
			cfg.killedAction = v
		}
	}

	if onDrop, ok := fmMap["onDrop"].(map[string]any); ok {
		if v, ok := onDrop["messageTemplate"].(string); ok {
			cfg.dropTemplate = v
		}
	}

	return cfg, true
}

func (p *ProcessorImpl) getFriendlyMonsterConfig(f field.Model, monsterId uint32) (Model, friendlyMonsterConfig, error) {
	for _, inst := range GetRegistry().GetAll(p.t) {
		if inst.State() != StateActive {
			continue
		}
		if inst.WorldId() != f.WorldId() || inst.ChannelId() != f.ChannelId() {
			continue
		}
		if f.Instance() != inst.Id() {
			continue
		}

		def, err := definition.NewProcessor(p.l, p.ctx, p.db).ByIdProvider(inst.DefinitionId())()
		if err != nil {
			continue
		}

		stageIdx := inst.CurrentStageIndex()
		if int(stageIdx) >= len(def.Stages()) {
			continue
		}

		stg := def.Stages()[stageIdx]
		for _, mid := range stg.MapIds() {
			if _map.Id(mid) != f.MapId() {
				continue
			}

			cfg, ok := extractFriendlyMonsterConfig(stg.Properties())
			if !ok {
				continue
			}

			if cfg.monsterId != monsterId {
				continue
			}

			return inst, cfg, nil
		}
	}
	return Model{}, friendlyMonsterConfig{}, errors.New("no active PQ instance with friendly monster config for this field")
}

func (p *ProcessorImpl) HandleFriendlyMonsterDamagedAndEmit(f field.Model, monsterId uint32) error {
	return message.Emit(p.p)(func(buf *message.Buffer) error {
		return p.HandleFriendlyMonsterDamaged(buf)(f, monsterId)
	})
}

func (p *ProcessorImpl) HandleFriendlyMonsterDamaged(mb *message.Buffer) func(f field.Model, monsterId uint32) error {
	return func(f field.Model, monsterId uint32) error {
		inst, cfg, err := p.getFriendlyMonsterConfig(f, monsterId)
		if err != nil {
			return err
		}

		// Increment hit counter.
		updated, err := GetRegistry().Update(p.t, inst.Id(), func(m Model) Model {
			return m.SetStageState(m.StageState().IncrementCustomData("friendlyHitCount"))
		})
		if err != nil {
			return err
		}

		if cfg.damagedInterval == 0 || cfg.damagedMessage == "" {
			return nil
		}

		hitCount := uint32(0)
		if v, ok := updated.StageState().CustomData()["friendlyHitCount"]; ok {
			switch n := v.(type) {
			case float64:
				hitCount = uint32(n)
			case int:
				hitCount = uint32(n)
			}
		}

		if hitCount%cfg.damagedInterval == 0 {
			return p.BroadcastMessage(mb)(inst.Id(), "BLUE_TEXT", cfg.damagedMessage)
		}

		return nil
	}
}

func (p *ProcessorImpl) HandleFriendlyMonsterKilledAndEmit(f field.Model, monsterId uint32) error {
	return message.Emit(p.p)(func(buf *message.Buffer) error {
		return p.HandleFriendlyMonsterKilled(buf)(f, monsterId)
	})
}

func (p *ProcessorImpl) HandleFriendlyMonsterKilled(mb *message.Buffer) func(f field.Model, monsterId uint32) error {
	return func(f field.Model, monsterId uint32) error {
		inst, cfg, err := p.getFriendlyMonsterConfig(f, monsterId)
		if err != nil {
			return err
		}

		if cfg.killedMessage != "" {
			_ = p.BroadcastMessage(mb)(inst.Id(), "BLUE_TEXT", cfg.killedMessage)
		}

		if cfg.killedAction == "fail" {
			_, err = GetRegistry().Update(p.t, inst.Id(), func(m Model) Model {
				return m.SetState(StateFailed)
			})
			if err != nil {
				return err
			}

			p.l.Infof("PQ instance [%s] failed due to friendly monster [%d] killed.", inst.Id(), monsterId)

			err = mb.Put(pq.EnvEventStatusTopic, failedEventProvider(inst.WorldId(), inst.Id(), inst.QuestId(), "friendly_monster_killed"))
			if err != nil {
				return err
			}

			return p.Destroy(mb)(inst.Id(), "friendly_monster_killed")
		}

		return nil
	}
}

func (p *ProcessorImpl) HandleFriendlyMonsterDropAndEmit(f field.Model, monsterId uint32, itemCount uint32) error {
	return message.Emit(p.p)(func(buf *message.Buffer) error {
		return p.HandleFriendlyMonsterDrop(buf)(f, monsterId, itemCount)
	})
}

func (p *ProcessorImpl) HandleFriendlyMonsterDrop(mb *message.Buffer) func(f field.Model, monsterId uint32, itemCount uint32) error {
	return func(f field.Model, monsterId uint32, itemCount uint32) error {
		inst, cfg, err := p.getFriendlyMonsterConfig(f, monsterId)
		if err != nil {
			return err
		}

		if cfg.dropTemplate == "" {
			return nil
		}

		// Increment drop counter by itemCount.
		var updated Model
		updated, err = GetRegistry().Update(p.t, inst.Id(), func(m Model) Model {
			ss := m.StageState()
			for i := uint32(0); i < itemCount; i++ {
				ss = ss.IncrementCustomData("friendlyDropCount")
			}
			return m.SetStageState(ss)
		})
		if err != nil {
			return err
		}

		dropCount := uint32(0)
		if v, ok := updated.StageState().CustomData()["friendlyDropCount"]; ok {
			switch n := v.(type) {
			case float64:
				dropCount = uint32(n)
			case int:
				dropCount = uint32(n)
			}
		}

		msg := strings.ReplaceAll(cfg.dropTemplate, "{count}", fmt.Sprintf("%d", dropCount))
		return p.BroadcastMessage(mb)(inst.Id(), "BLUE_TEXT", msg)
	}
}

func evaluateClearConditions(conditions []condition.Model, ss StageState) bool {
	// Empty conditions = always pass (external signal)
	if len(conditions) == 0 {
		return true
	}

	for _, c := range conditions {
		if !evaluateCondition(c, ss) {
			return false
		}
	}
	return true
}

func evaluateCondition(c condition.Model, ss StageState) bool {
	var actual uint32

	switch c.Type() {
	case "item":
		actual = ss.ItemCounts()[c.ReferenceId()]
	case "monster_kill":
		actual = ss.MonsterKills()[c.ReferenceId()]
	case "custom_data":
		if v, ok := ss.CustomData()[c.ReferenceKey()]; ok {
			switch n := v.(type) {
			case float64:
				actual = uint32(n)
			case int:
				actual = uint32(n)
			}
		}
	default:
		return true
	}

	return compareValues(actual, c.Operator(), c.Value())
}

func compareValues(actual uint32, operator string, expected uint32) bool {
	switch operator {
	case ">=":
		return actual >= expected
	case "<=":
		return actual <= expected
	case "=":
		return actual == expected
	case ">":
		return actual > expected
	case "<":
		return actual < expected
	default:
		return false
	}
}

func generateCombination(properties map[string]any) []uint32 {
	digits := uint32(3)
	positions := uint32(3)

	if d, ok := properties["digits"]; ok {
		if df, ok := d.(float64); ok {
			digits = uint32(df)
		}
	}
	if p, ok := properties["positions"]; ok {
		if pf, ok := p.(float64); ok {
			positions = uint32(pf)
		}
	}

	combo := make([]uint32, positions)
	for i := range combo {
		combo[i] = uint32(rand.Intn(int(digits)))
	}
	return combo
}
