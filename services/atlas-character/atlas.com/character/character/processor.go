package character

import (
	"atlas-character/data/portal"
	skill3 "atlas-character/data/skill"
	"atlas-character/drop"
	"atlas-character/external/effective_stats"
	"atlas-character/kafka/message"
	character2 "atlas-character/kafka/message/character"
	"atlas-character/location"
	skill2 "atlas-character/skill"
	"atlas-character/teleport_rock"
	"context"
	"errors"
	"math"
	"math/rand"
	"regexp"
	"time"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/constants"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	"github.com/Chronicle20/atlas/libs/atlas-constants/stat"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	outbox "github.com/Chronicle20/atlas/libs/atlas-outbox"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

var (
	errBlockedName  = errors.New("blocked name")
	errInvalidLevel = errors.New("invalid level")
)

// ErrNotEnoughMeso signals a rejected meso change: no state was written and
// the rejection status event is emitted outside the transaction.
var ErrNotEnoughMeso = errors.New("not enough meso")

// ErrMesoOverflow rejects a change that would overflow the uint32 meso field.
var ErrMesoOverflow = errors.New("meso overflow")

type NameValidityResult struct {
	Valid  bool
	Reason string
	Detail string
}

const (
	CommandDistributeApAbilityStrength     = "STRENGTH"
	CommandDistributeApAbilityDexterity    = "DEXTERITY"
	CommandDistributeApAbilityIntelligence = "INTELLIGENCE"
	CommandDistributeApAbilityLuck         = "LUCK"
	CommandDistributeApAbilityHp           = "HP"
	CommandDistributeApAbilityMp           = "MP"
)

// appliesAutoAP reports whether Beginner/Noblesse/Legend auto-AP assignment applies.
// Pre-Big-Bang GMS behavior (..94 era); v84 included. Evidence: v84-packet-delta.md §3-5.
func appliesAutoAP(t tenant.Model) bool {
	return t.IsRegion("GMS") && t.MajorAtMost(94)
}

type Processor interface {
	WithTransaction(tx *gorm.DB) Processor
	ByIdProvider(decorators ...model.Decorator[Model]) func(id uint32) model.Provider[Model]
	GetById(decorators ...model.Decorator[Model]) func(id uint32) (Model, error)
	GetForAccountInWorld(decorators ...model.Decorator[Model]) func(accountId uint32, worldId world.Id) ([]Model, error)
	GetForAccountInWorldProvider(page model.Page, decorators ...model.Decorator[Model]) func(accountId uint32, worldId world.Id) model.Provider[model.Paged[Model]]
	GetForName(decorators ...model.Decorator[Model]) func(name string) ([]Model, error)
	GetForNameProvider(page model.Page, decorators ...model.Decorator[Model]) func(name string) model.Provider[model.Paged[Model]]
	AllProvider(page model.Page, decorators ...model.Decorator[Model]) model.Provider[model.Paged[Model]]
	SkillModelDecorator(m Model) Model
	IsValidName(name string) (bool, error)
	CheckNameValidity(name string, worldId world.Id) (NameValidityResult, error)
	CreateAndEmit(transactionId uuid.UUID, input Model, mapId _map.Id) (Model, error)
	Create(mb *message.Buffer) func(transactionId uuid.UUID, input Model, mapId _map.Id) (Model, error)
	DeleteAndEmit(transactionId uuid.UUID, characterId uint32) error
	Delete(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32) error
	DeleteByAccountIdAndEmit(accountId uint32) error
	DeleteByAccountId(mb *message.Buffer) func(accountId uint32) error
	DeleteForSagaCompensationAndEmit(transactionId uuid.UUID, characterId uint32) error
	DeleteForSagaCompensation(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32) error
	LoginAndEmit(transactionId uuid.UUID, characterId uint32, channel channel.Model) error
	Login(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, channel channel.Model) error
	LogoutAndEmit(transactionId uuid.UUID, characterId uint32, channel channel.Model) error
	Logout(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, channel channel.Model) error
	ChangeJobAndEmit(transactionId uuid.UUID, characterId uint32, channel channel.Model, jobId job.Id) error
	ChangeJob(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, channel channel.Model, jobId job.Id) error
	ChangeHairAndEmit(transactionId uuid.UUID, characterId uint32, channel channel.Model, styleId uint32) error
	ChangeHair(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, channel channel.Model, styleId uint32) error
	ChangeFaceAndEmit(transactionId uuid.UUID, characterId uint32, channel channel.Model, styleId uint32) error
	ChangeFace(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, channel channel.Model, styleId uint32) error
	ChangeSkinAndEmit(transactionId uuid.UUID, characterId uint32, channel channel.Model, styleId byte) error
	ChangeSkin(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, channel channel.Model, styleId byte) error
	AwardExperienceAndEmit(transactionId uuid.UUID, characterId uint32, channel channel.Model, experience []ExperienceModel, showEffect bool) error
	AwardExperience(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, channel channel.Model, experience []ExperienceModel, showEffect bool) error
	DeductExperienceAndEmit(transactionId uuid.UUID, characterId uint32, channel channel.Model, amount uint32) error
	DeductExperience(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, channel channel.Model, amount uint32) error
	AwardLevelAndEmit(transactionId uuid.UUID, characterId uint32, channel channel.Model, level byte) error
	AwardLevel(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, channel channel.Model, level byte) error
	Move(characterId uint32, x int16, y int16, fh int16, stance byte) error
	RequestChangeMeso(transactionId uuid.UUID, characterId uint32, amount int32, actorId uint32, actorType string, showEffect bool) error
	AttemptMesoPickUp(transactionId uuid.UUID, field field.Model, characterId uint32, dropId uint32, meso uint32) error
	RequestDropMeso(transactionId uuid.UUID, field field.Model, characterId uint32, amount uint32) error
	RequestChangeFame(transactionId uuid.UUID, characterId uint32, amount int8, actorId uint32, actorType string) error
	RequestDistributeAp(transactionId uuid.UUID, characterId uint32, distributions []Distribution) error
	RequestDistributeSp(transactionId uuid.UUID, characterId uint32, skillId uint32, amount int8) error
	ChangeHPAndEmit(transactionId uuid.UUID, channel channel.Model, characterId uint32, amount int16) error
	ChangeHP(mb *message.Buffer) func(transactionId uuid.UUID, channel channel.Model, characterId uint32, amount int16) error
	SetHPAndEmit(transactionId uuid.UUID, channel channel.Model, characterId uint32, amount uint16) error
	SetHP(mb *message.Buffer) func(transactionId uuid.UUID, channel channel.Model, characterId uint32, amount uint16) error
	ChangeMPAndEmit(transactionId uuid.UUID, channel channel.Model, characterId uint32, amount int16) error
	ChangeMP(mb *message.Buffer) func(transactionId uuid.UUID, channel channel.Model, characterId uint32, amount int16) error
	ClampHPAndEmit(transactionId uuid.UUID, channel channel.Model, characterId uint32, maxValue uint16) error
	ClampHP(mb *message.Buffer) func(transactionId uuid.UUID, channel channel.Model, characterId uint32, maxValue uint16) error
	ClampMPAndEmit(transactionId uuid.UUID, channel channel.Model, characterId uint32, maxValue uint16) error
	ClampMP(mb *message.Buffer) func(transactionId uuid.UUID, channel channel.Model, characterId uint32, maxValue uint16) error
	ProcessLevelChangeAndEmit(transactionId uuid.UUID, channel channel.Model, characterId uint32, amount byte) error
	ProcessLevelChange(mb *message.Buffer) func(transactionId uuid.UUID, channel channel.Model, characterId uint32, amount byte) error
	ProcessJobChangeAndEmit(transactionId uuid.UUID, channel channel.Model, characterId uint32, jobId job.Id) error
	ProcessJobChange(mb *message.Buffer) func(transactionId uuid.UUID, channel channel.Model, characterId uint32, jobId job.Id) error
	UpdateAndEmit(transactionId uuid.UUID, characterId uint32, input RestModel) error
	Update(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, input RestModel) error
	ResetStatsAndEmit(transactionId uuid.UUID, characterId uint32, channel channel.Model) error
	ResetStats(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, channel channel.Model) error
	RebalanceAPAndEmit(transactionId uuid.UUID, characterId uint32, channel channel.Model, targets []RebalanceTarget) error
	RebalanceAP(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, channel channel.Model, targets []RebalanceTarget) error
	TransferAPAndEmit(transactionId uuid.UUID, characterId uint32, channel channel.Model, from string, to string) error
	TransferAP(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, channel channel.Model, from string, to string) error
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	db  *gorm.DB
	t   tenant.Model
	pp  portal.Processor
	sp  skill2.Processor
	sdp skill3.Processor
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) Processor {
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
		db:  db,
		t:   tenant.MustFromContext(ctx),
		pp:  portal.NewProcessor(l, ctx),
		sp:  skill2.NewProcessor(l, ctx),
		sdp: skill3.NewProcessor(l, ctx),
	}
}

var _ Processor = (*ProcessorImpl)(nil)

// set returns this processor's tenant's version-aware skill/job identity
// binding table (task-187). Job/skill wire ids are version-specific: the
// same wire id can mean a different job/skill at different client versions
// (e.g. wire job 500 is GM at v0.48 but Pirate at v0.61+), so any branch
// keyed off c.JobId()/a skill wire id must resolve through this Set rather
// than compare raw job.Id/skill.Id constants directly.
func (p *ProcessorImpl) set() constants.SkillJobSet {
	return constants.For(p.t.Region(), p.t.MajorVersion(), p.t.MinorVersion())
}

func (p *ProcessorImpl) WithTransaction(tx *gorm.DB) Processor {
	return &ProcessorImpl{
		l:   p.l,
		ctx: p.ctx,
		db:  tx,
		t:   p.t,
		pp:  p.pp,
		sp:  p.sp,
		sdp: p.sdp,
	}
}

func (p *ProcessorImpl) ByIdProvider(decorators ...model.Decorator[Model]) func(id uint32) model.Provider[Model] {
	return func(id uint32) model.Provider[Model] {
		mp := model.Map(modelFromEntity)(getById(id)(p.db.WithContext(p.ctx)))
		return model.Map(model.Decorate[Model](decorators))(mp)
	}
}

// GetById Retrieves a singular character by id.
func (p *ProcessorImpl) GetById(decorators ...model.Decorator[Model]) func(id uint32) (Model, error) {
	return func(id uint32) (Model, error) {
		return p.ByIdProvider(decorators...)(id)()
	}
}

func (p *ProcessorImpl) GetForAccountInWorld(decorators ...model.Decorator[Model]) func(accountId uint32, worldId world.Id) ([]Model, error) {
	return func(accountId uint32, worldId world.Id) ([]Model, error) {
		mp := model.SliceMap(modelFromEntity)(getForAccountInWorld(accountId, worldId)(p.db.WithContext(p.ctx)))(model.ParallelMap())
		return model.SliceMap(model.Decorate(decorators))(mp)(model.ParallelMap())()
	}
}

func (p *ProcessorImpl) GetForAccountInWorldProvider(page model.Page, decorators ...model.Decorator[Model]) func(accountId uint32, worldId world.Id) model.Provider[model.Paged[Model]] {
	return func(accountId uint32, worldId world.Id) model.Provider[model.Paged[Model]] {
		ep := getForAccountInWorldPaged(accountId, worldId, page)(p.db.WithContext(p.ctx))
		mp := model.MapPaged(modelFromEntity)(ep)(model.ParallelMap())
		return model.MapPaged(model.Decorate[Model](decorators))(mp)(model.ParallelMap())
	}
}

func (p *ProcessorImpl) GetForName(decorators ...model.Decorator[Model]) func(name string) ([]Model, error) {
	return func(name string) ([]Model, error) {
		mp := model.SliceMap[entity, Model](modelFromEntity)(getForName(name)(p.db.WithContext(p.ctx)))(model.ParallelMap())
		return model.SliceMap(model.Decorate[Model](decorators))(mp)(model.ParallelMap())()
	}
}

func (p *ProcessorImpl) GetForNameProvider(page model.Page, decorators ...model.Decorator[Model]) func(name string) model.Provider[model.Paged[Model]] {
	return func(name string) model.Provider[model.Paged[Model]] {
		ep := getForNamePaged(name, page)(p.db.WithContext(p.ctx))
		mp := model.MapPaged(modelFromEntity)(ep)(model.ParallelMap())
		return model.MapPaged(model.Decorate[Model](decorators))(mp)(model.ParallelMap())
	}
}

func (p *ProcessorImpl) AllProvider(page model.Page, decorators ...model.Decorator[Model]) model.Provider[model.Paged[Model]] {
	ep := getAll(page)(p.db.WithContext(p.ctx))
	mp := model.MapPaged(modelFromEntity)(ep)(model.ParallelMap())
	return model.MapPaged(model.Decorate[Model](decorators))(mp)(model.ParallelMap())
}

func (p *ProcessorImpl) SkillModelDecorator(m Model) Model {
	ms, err := p.sp.GetByCharacterId(m.Id())
	if err != nil {
		return m
	}
	return CloneModel(m).SetSkills(ms).Build()
}

func (p *ProcessorImpl) IsValidName(name string) (bool, error) {
	m, err := regexp.MatchString("^[A-Za-z0-9\u3040-\u309F\u30A0-\u30FF\u4E00-\u9FAF]{3,12}$", name)
	if err != nil {
		return false, err
	}
	if !m {
		return false, nil
	}

	cs, err := p.GetForName()(name)
	if len(cs) != 0 || err != nil {
		return false, nil
	}

	//TODO
	//bn, err := blocked_name.IsBlockedName(l, ctx)(name)
	//if bn {
	//	return false, err
	//}

	return true, nil
}

func (p *ProcessorImpl) CheckNameValidity(name string, worldId world.Id) (NameValidityResult, error) {
	if len(name) < 3 || len(name) > 12 {
		return NameValidityResult{Valid: false, Reason: "length", Detail: "Name must be 3-12 characters."}, nil
	}
	m, err := regexp.MatchString("^[A-Za-z0-9\u3040-\u309F\u30A0-\u30FF\u4E00-\u9FAF]{3,12}$", name)
	if err != nil {
		return NameValidityResult{}, err
	}
	if !m {
		return NameValidityResult{Valid: false, Reason: "regex", Detail: "Name contains invalid characters."}, nil
	}
	cs, err := p.GetForName()(name)
	if err != nil {
		return NameValidityResult{}, err
	}
	for _, c := range cs {
		if c.WorldId() == worldId {
			return NameValidityResult{Valid: false, Reason: "duplicate", Detail: "Name already taken."}, nil
		}
	}
	return NameValidityResult{Valid: true}, nil
}

func (p *ProcessorImpl) CreateAndEmit(transactionId uuid.UUID, input Model, mapId _map.Id) (Model, error) {
	var output Model
	txErr := database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
		return message.Emit(outbox.EmitProvider(p.l, p.ctx, tx))(func(buf *message.Buffer) error {
			var err error
			output, err = p.WithTransaction(tx).Create(buf)(transactionId, input, mapId)
			if err != nil {
				// Emit creation failed event on error
				_ = buf.Put(character2.EnvEventTopicCharacterStatus, creationFailedEventProvider(transactionId, input.WorldId(), input.Name(), err.Error()))
			}
			return err
		})
	})
	return output, txErr
}

func (p *ProcessorImpl) Create(mb *message.Buffer) func(transactionId uuid.UUID, input Model, mapId _map.Id) (Model, error) {
	return func(transactionId uuid.UUID, input Model, mapId _map.Id) (Model, error) {
		ok, err := p.IsValidName(input.Name())
		if err != nil {
			p.l.WithError(err).Errorf("Error validating name [%s] during character creation.", input.Name())
			return Model{}, err
		}
		if !ok {
			p.l.Infof("Attempting to create a character with an invalid name [%s].", input.Name())
			return Model{}, errBlockedName
		}
		if input.Level() < 1 || input.Level() > 200 {
			p.l.Infof("Attempting to create character with an invalid level [%d].", input.Level())
			return Model{}, errInvalidLevel
		}

		var res Model
		txErr := database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
			res, err = create(tx, p.t.Id(), input.accountId, input.worldId, input.name, input.level, input.strength, input.dexterity, input.intelligence, input.luck, input.maxHp, input.maxMp, input.jobId, input.gender, input.hair, input.face, input.skinColor, input.gm, input.meso)
			if err != nil {
				p.l.WithError(err).Errorf("Error persisting character in database.")
				tx.Rollback()
				return err
			}
			// task-055 Blocker 2 follow-up: include the spawn mapId on CREATED so
			// atlas-maps can seed character_locations before the first LOGIN.
			// Without this, the first LOGIN's location.GetField returns 404 and
			// atlas-character falls back to mapId=0, anchoring new characters
			// on map 0. Instance is always uuid.Nil at creation time.
			return mb.Put(character2.EnvEventTopicCharacterStatus, createdEventProvider(transactionId, res.Id(), res.WorldId(), res.Name(), mapId))
		})
		if txErr != nil {
			p.l.WithError(txErr).Errorf("Error persisting character in database.")
			return Model{}, txErr
		}
		return res, nil
	}
}

func (p *ProcessorImpl) DeleteAndEmit(transactionId uuid.UUID, characterId uint32) error {
	return database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
		return message.Emit(outbox.EmitProvider(p.l, p.ctx, tx))(func(buf *message.Buffer) error {
			return p.WithTransaction(tx).Delete(buf)(transactionId, characterId)
		})
	})
}

func (p *ProcessorImpl) Delete(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32) error {
	return func(transactionId uuid.UUID, characterId uint32) error {
		txErr := database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
			c, err := p.GetById()(characterId)
			if err != nil {
				return err
			}

			err = delete(tx, characterId)
			if err != nil {
				return err
			}

			if err = teleport_rock.DeleteForCharacter(tx, p.t.Id(), characterId); err != nil {
				return err
			}

			return mb.Put(character2.EnvEventTopicCharacterStatus, deletedEventProvider(transactionId, characterId, c.WorldId()))
		})
		if txErr != nil {
			p.l.WithError(txErr).Errorf("Error deleting character [%d] from database.", characterId)
			return txErr
		}
		return nil
	}
}

// DeleteForSagaCompensation is the buffer-based inner form of the saga-correlated
// delete. Idempotent on missing rows: if the character is absent, a synthetic
// DELETED event is still buffered so the orchestrator's correlator records the
// compensation step as completed. See PRD §4.3.1 / §4.8 and plan Phase 5.
func (p *ProcessorImpl) DeleteForSagaCompensation(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32) error {
	return func(transactionId uuid.UUID, characterId uint32) error {
		_, err := p.GetById()(characterId)
		if err == nil {
			return p.Delete(mb)(transactionId, characterId)
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		p.l.WithFields(logrus.Fields{
			"transaction_id": transactionId.String(),
			"character_id":   characterId,
		}).Info("Character already absent; buffering synthetic DELETED event for saga compensation.")
		return mb.Put(character2.EnvEventTopicCharacterStatus, deletedEventProvider(transactionId, characterId, 0))
	}
}

// DeleteForSagaCompensationAndEmit wraps DeleteForSagaCompensation with the
// producer emit flow. See DeleteForSagaCompensation for the idempotency
// contract.
func (p *ProcessorImpl) DeleteForSagaCompensationAndEmit(transactionId uuid.UUID, characterId uint32) error {
	return database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
		return message.Emit(outbox.EmitProvider(p.l, p.ctx, tx))(func(buf *message.Buffer) error {
			return p.WithTransaction(tx).DeleteForSagaCompensation(buf)(transactionId, characterId)
		})
	})
}

// DeleteByAccountIdAndEmit deletes every character for an account. Each
// character's delete+emit is its own atomic unit via DeleteAndEmit (already
// migrated to the outbox). This deliberately does NOT wrap the whole batch in
// a single outer transaction: DeleteByAccountId's contract is best-effort
// (log-and-continue on a single character's failure), and sharing one
// Postgres transaction across the batch would let one character's failure
// abort the transaction and cascade-fail every subsequent character in the
// loop (an aborted tx rejects all further statements until rollback).
// Per-character transactions preserve the existing independent-failure
// semantics while still gaining mutation+enqueue atomicity per character.
func (p *ProcessorImpl) DeleteByAccountIdAndEmit(accountId uint32) error {
	cs, err := model.SliceMap(modelFromEntity)(getForAccount(accountId)(p.db.WithContext(p.ctx)))(model.ParallelMap())()
	if err != nil {
		p.l.WithError(err).Errorf("Unable to retrieve characters for account [%d].", accountId)
		return err
	}

	p.l.Infof("Deleting [%d] characters for account [%d].", len(cs), accountId)
	for _, c := range cs {
		if err := p.DeleteAndEmit(uuid.Nil, c.Id()); err != nil {
			p.l.WithError(err).Errorf("Unable to delete character [%d] for account [%d].", c.Id(), accountId)
		}
	}
	return nil
}

func (p *ProcessorImpl) DeleteByAccountId(mb *message.Buffer) func(accountId uint32) error {
	return func(accountId uint32) error {
		cs, err := model.SliceMap(modelFromEntity)(getForAccount(accountId)(p.db.WithContext(p.ctx)))(model.ParallelMap())()
		if err != nil {
			p.l.WithError(err).Errorf("Unable to retrieve characters for account [%d].", accountId)
			return err
		}

		p.l.Infof("Deleting [%d] characters for account [%d].", len(cs), accountId)
		for _, c := range cs {
			err = p.Delete(mb)(uuid.Nil, c.Id())
			if err != nil {
				p.l.WithError(err).Errorf("Unable to delete character [%d] for account [%d].", c.Id(), accountId)
			}
		}
		return nil
	}
}

func (p *ProcessorImpl) LoginAndEmit(transactionId uuid.UUID, characterId uint32, channel channel.Model) error {
	return message.Emit(producer.ProviderImpl(p.l)(p.ctx))(func(buf *message.Buffer) error {
		return p.Login(buf)(transactionId, characterId, channel)
	})
}

func (p *ProcessorImpl) Login(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, channel channel.Model) error {
	return func(transactionId uuid.UUID, characterId uint32, channel channel.Model) error {
		return model.For(p.ByIdProvider()(characterId), func(c Model) error {
			f, err := location.GetField(p.l, p.ctx, c.Id())
			if err != nil {
				if errors.Is(err, location.ErrNotFound) {
					p.l.Warnf("Login: no atlas-maps location for [%d] (likely first login of new character); emitting with zero map.", c.Id())
				} else {
					p.l.WithError(err).Errorf("Login: atlas-maps lookup failed for [%d] (infrastructure error); emitting with zero map.", c.Id())
				}
				f = field.NewBuilder(channel.WorldId(), channel.Id(), 0).SetInstance(uuid.Nil).Build()
			} else {
				// The stored channelId in character_locations reflects the
				// channel the character was last on (or 0 from the
				// CREATED-time seed). The character is now logging in to a
				// possibly different channel — override the stored
				// world/channel with the current login so downstream
				// consumers (atlas-maps -> atlas-monsters -> atlas-channel)
				// route MAP_STATUS / MONSTER_STATUS events to the channel
				// the user is actually connected to. mapId/instance from
				// storage remain authoritative.
				f = field.NewBuilder(channel.WorldId(), channel.Id(), f.MapId()).SetInstance(f.Instance()).Build()
			}
			return mb.Put(character2.EnvEventTopicCharacterStatus, loginEventProvider(transactionId, c.Id(), f))
		})
	}
}

func (p *ProcessorImpl) LogoutAndEmit(transactionId uuid.UUID, characterId uint32, channel channel.Model) error {
	return message.Emit(producer.ProviderImpl(p.l)(p.ctx))(func(buf *message.Buffer) error {
		return p.Logout(buf)(transactionId, characterId, channel)
	})
}

func (p *ProcessorImpl) Logout(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, channel channel.Model) error {
	return func(transactionId uuid.UUID, characterId uint32, channel channel.Model) error {
		return model.For(p.ByIdProvider()(characterId), func(c Model) error {
			f, err := location.GetField(p.l, p.ctx, c.Id())
			if err != nil {
				if errors.Is(err, location.ErrNotFound) {
					p.l.Warnf("Logout: no atlas-maps location for [%d] (likely first login of new character); emitting with zero map.", c.Id())
				} else {
					p.l.WithError(err).Errorf("Logout: atlas-maps lookup failed for [%d] (infrastructure error); emitting with zero map.", c.Id())
				}
				f = field.NewBuilder(channel.WorldId(), channel.Id(), 0).SetInstance(uuid.Nil).Build()
			} else {
				// Override stored world/channel with the channel the
				// character is actually disconnecting from so downstream
				// services release per-channel state (e.g. atlas-monsters
				// control assignment) on the right channel. The `channel`
				// argument here originates from the session registry's
				// transition-state values (see session/task.go), so it is
				// the channel the destroyed session was last on — which is
				// where downstream per-channel state lives. See Login for
				// the symmetric rationale.
				f = field.NewBuilder(channel.WorldId(), channel.Id(), f.MapId()).SetInstance(f.Instance()).Build()
			}
			return mb.Put(character2.EnvEventTopicCharacterStatus, logoutEventProvider(transactionId, c.Id(), f))
		})
	}
}

func (p *ProcessorImpl) ChangeJobAndEmit(transactionId uuid.UUID, characterId uint32, channel channel.Model, jobId job.Id) error {
	return database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
		return message.Emit(outbox.EmitProvider(p.l, p.ctx, tx))(func(buf *message.Buffer) error {
			return p.WithTransaction(tx).ChangeJob(buf)(transactionId, characterId, channel, jobId)
		})
	})
}

func (p *ProcessorImpl) ChangeJob(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, channel channel.Model, jobId job.Id) error {
	return func(transactionId uuid.UUID, characterId uint32, channel channel.Model, jobId job.Id) error {
		p.l.Debugf("Attempting to set character [%d] job to [%d].", characterId, jobId)
		txErr := database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
			c, err := p.WithTransaction(tx).GetById()(characterId)
			if err != nil {
				return err
			}
			err = dynamicUpdate(tx)(SetJob(jobId))(c)
			if err != nil {
				return err
			}
			return nil
		})
		if txErr != nil {
			p.l.WithError(txErr).Errorf("Could not set character [%d] job to [%d].", characterId, jobId)
			return txErr
		}
		_ = mb.Put(character2.EnvEventTopicCharacterStatus, jobChangedEventProvider(transactionId, characterId, channel, jobId))
		_ = mb.Put(character2.EnvEventTopicCharacterStatus, statChangedProvider(transactionId, channel, characterId, []stat.Type{stat.TypeJob}, nil))
		return nil
	}
}

func (p *ProcessorImpl) ChangeHairAndEmit(transactionId uuid.UUID, characterId uint32, channel channel.Model, styleId uint32) error {
	return database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
		return message.Emit(outbox.EmitProvider(p.l, p.ctx, tx))(func(buf *message.Buffer) error {
			return p.WithTransaction(tx).ChangeHair(buf)(transactionId, characterId, channel, styleId)
		})
	})
}

func (p *ProcessorImpl) ChangeHair(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, channel channel.Model, styleId uint32) error {
	return func(transactionId uuid.UUID, characterId uint32, channel channel.Model, styleId uint32) error {
		p.l.Debugf("Attempting to set character [%d] hair to [%d].", characterId, styleId)
		var oldHair uint32
		txErr := database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
			c, err := p.WithTransaction(tx).GetById()(characterId)
			if err != nil {
				return err
			}
			oldHair = c.Hair()
			err = dynamicUpdate(tx)(SetHair(styleId))(c)
			if err != nil {
				return err
			}
			return nil
		})
		if txErr != nil {
			p.l.WithError(txErr).Errorf("Could not set character [%d] hair to [%d].", characterId, styleId)
			return txErr
		}
		_ = mb.Put(character2.EnvEventTopicCharacterStatus, hairChangedEventProvider(transactionId, characterId, channel.WorldId(), oldHair, styleId))
		_ = mb.Put(character2.EnvEventTopicCharacterStatus, statChangedProvider(transactionId, channel, characterId, []stat.Type{stat.TypeHair}, nil))
		return nil
	}
}

func (p *ProcessorImpl) ChangeFaceAndEmit(transactionId uuid.UUID, characterId uint32, channel channel.Model, styleId uint32) error {
	return database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
		return message.Emit(outbox.EmitProvider(p.l, p.ctx, tx))(func(buf *message.Buffer) error {
			return p.WithTransaction(tx).ChangeFace(buf)(transactionId, characterId, channel, styleId)
		})
	})
}

func (p *ProcessorImpl) ChangeFace(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, channel channel.Model, styleId uint32) error {
	return func(transactionId uuid.UUID, characterId uint32, channel channel.Model, styleId uint32) error {
		p.l.Debugf("Attempting to set character [%d] face to [%d].", characterId, styleId)
		var oldFace uint32
		txErr := database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
			c, err := p.WithTransaction(tx).GetById()(characterId)
			if err != nil {
				return err
			}
			oldFace = c.Face()
			err = dynamicUpdate(tx)(SetFace(styleId))(c)
			if err != nil {
				return err
			}
			return nil
		})
		if txErr != nil {
			p.l.WithError(txErr).Errorf("Could not set character [%d] face to [%d].", characterId, styleId)
			return txErr
		}
		_ = mb.Put(character2.EnvEventTopicCharacterStatus, faceChangedEventProvider(transactionId, characterId, channel.WorldId(), oldFace, styleId))
		_ = mb.Put(character2.EnvEventTopicCharacterStatus, statChangedProvider(transactionId, channel, characterId, []stat.Type{stat.TypeFace}, nil))
		return nil
	}
}

func (p *ProcessorImpl) ChangeSkinAndEmit(transactionId uuid.UUID, characterId uint32, channel channel.Model, styleId byte) error {
	return database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
		return message.Emit(outbox.EmitProvider(p.l, p.ctx, tx))(func(buf *message.Buffer) error {
			return p.WithTransaction(tx).ChangeSkin(buf)(transactionId, characterId, channel, styleId)
		})
	})
}

func (p *ProcessorImpl) ChangeSkin(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, channel channel.Model, styleId byte) error {
	return func(transactionId uuid.UUID, characterId uint32, channel channel.Model, styleId byte) error {
		p.l.Debugf("Attempting to set character [%d] skin to [%d].", characterId, styleId)
		var oldSkin byte
		txErr := database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
			c, err := p.WithTransaction(tx).GetById()(characterId)
			if err != nil {
				return err
			}
			oldSkin = c.SkinColor()
			err = dynamicUpdate(tx)(SetSkinColor(styleId))(c)
			if err != nil {
				return err
			}
			return nil
		})
		if txErr != nil {
			p.l.WithError(txErr).Errorf("Could not set character [%d] skin to [%d].", characterId, styleId)
			return txErr
		}
		_ = mb.Put(character2.EnvEventTopicCharacterStatus, skinColorChangedEventProvider(transactionId, characterId, channel.WorldId(), oldSkin, styleId))
		_ = mb.Put(character2.EnvEventTopicCharacterStatus, statChangedProvider(transactionId, channel, characterId, []stat.Type{stat.TypeSkin}, nil))
		return nil
	}
}

type ExperienceModel struct {
	experienceType string
	amount         uint32
	attr1          uint32
}

func NewExperienceModel(experienceType string, amount uint32, attr1 uint32) ExperienceModel {
	return ExperienceModel{experienceType, amount, attr1}
}

func (p *ProcessorImpl) AwardExperienceAndEmit(transactionId uuid.UUID, characterId uint32, channel channel.Model, experience []ExperienceModel, showEffect bool) error {
	return database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
		return message.Emit(outbox.EmitProvider(p.l, p.ctx, tx))(func(buf *message.Buffer) error {
			return p.WithTransaction(tx).AwardExperience(buf)(transactionId, characterId, channel, experience, showEffect)
		})
	})
}

func (p *ProcessorImpl) AwardExperience(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, channel channel.Model, experience []ExperienceModel, showEffect bool) error {
	return func(transactionId uuid.UUID, characterId uint32, channel channel.Model, experience []ExperienceModel, showEffect bool) error {
		amount := uint32(0)
		for _, e := range experience {
			amount += e.amount
		}

		p.l.Debugf("Attempting to award character [%d] [%d] experience.", characterId, amount)
		awardedLevels := byte(0)
		current := uint32(0)
		txErr := database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
			c, err := p.WithTransaction(tx).GetById()(characterId)
			if err != nil {
				return err
			}

			curLevel := c.Level()
			current = c.Experience() + amount
			for current > GetExperienceNeededForLevel(curLevel) {
				current -= GetExperienceNeededForLevel(curLevel)
				curLevel += 1
				awardedLevels += 1
			}

			err = dynamicUpdate(tx)(SetExperience(current))(c)
			if err != nil {
				return err
			}
			return nil
		})
		if txErr != nil {
			p.l.WithError(txErr).Errorf("Could not award character [%d] [%d] experience.", characterId, amount)
			return txErr
		}

		// When the saga step requested a visible effect, append White + Chat
		// distributions so atlas-channel renders the chat line ("You have gained
		// N exp. (+N)"). Non-conversation paths (showEffect=false) keep the
		// existing distribution shape.
		emittedExperience := experience
		if showEffect && amount > 0 {
			emittedExperience = append([]ExperienceModel(nil), experience...)
			emittedExperience = append(emittedExperience,
				NewExperienceModel(character2.ExperienceDistributionTypeWhite, amount, 0),
				NewExperienceModel(character2.ExperienceDistributionTypeChat, amount, 0),
			)
		}
		_ = mb.Put(character2.EnvEventTopicCharacterStatus, experienceChangedEventProvider(transactionId, characterId, channel, emittedExperience, current))
		_ = mb.Put(character2.EnvEventTopicCharacterStatus, statChangedProvider(transactionId, channel, characterId, []stat.Type{stat.TypeExperience}, nil))
		if awardedLevels > 0 {
			_ = mb.Put(character2.EnvCommandTopic, awardLevelCommandProvider(transactionId, characterId, channel, awardedLevels))
		}
		return nil
	}
}

func (p *ProcessorImpl) DeductExperienceAndEmit(transactionId uuid.UUID, characterId uint32, channel channel.Model, amount uint32) error {
	return database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
		return message.Emit(outbox.EmitProvider(p.l, p.ctx, tx))(func(buf *message.Buffer) error {
			return p.WithTransaction(tx).DeductExperience(buf)(transactionId, characterId, channel, amount)
		})
	})
}

func (p *ProcessorImpl) DeductExperience(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, channel channel.Model, amount uint32) error {
	return func(transactionId uuid.UUID, characterId uint32, channel channel.Model, amount uint32) error {
		p.l.Debugf("Attempting to deduct [%d] experience from character [%d].", amount, characterId)
		current := uint32(0)
		txErr := database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
			c, err := p.WithTransaction(tx).GetById()(characterId)
			if err != nil {
				return err
			}

			if c.Experience() >= amount {
				current = c.Experience() - amount
			} else {
				current = 0
			}

			err = dynamicUpdate(tx)(SetExperience(current))(c)
			if err != nil {
				return err
			}
			return nil
		})
		if txErr != nil {
			p.l.WithError(txErr).Errorf("Could not deduct [%d] experience from character [%d].", amount, characterId)
			return txErr
		}

		// Create an experience distribution representing the deduction (negative display)
		deduction := []ExperienceModel{{experienceType: character2.ExperienceDistributionTypeDeath, amount: amount, attr1: 0}}
		_ = mb.Put(character2.EnvEventTopicCharacterStatus, experienceChangedEventProvider(transactionId, characterId, channel, deduction, current))
		_ = mb.Put(character2.EnvEventTopicCharacterStatus, statChangedProvider(transactionId, channel, characterId, []stat.Type{stat.TypeExperience}, nil))
		return nil
	}
}

func (p *ProcessorImpl) AwardLevelAndEmit(transactionId uuid.UUID, characterId uint32, channel channel.Model, amount byte) error {
	return database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
		return message.Emit(outbox.EmitProvider(p.l, p.ctx, tx))(func(buf *message.Buffer) error {
			return p.WithTransaction(tx).AwardLevel(buf)(transactionId, characterId, channel, amount)
		})
	})
}

func (p *ProcessorImpl) AwardLevel(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, channel channel.Model, amount byte) error {
	return func(transactionId uuid.UUID, characterId uint32, channel channel.Model, amount byte) error {
		p.l.Debugf("Attempting to award character [%d] [%d] level(s).", characterId, amount)
		actual := amount
		current := byte(0)
		txErr := database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
			c, err := p.WithTransaction(tx).GetById()(characterId)
			if err != nil {
				return err
			}

			if c.Level()+amount > MaxLevel {
				p.l.Debugf("Awarding [%d] level(s) would cause character [%d] to go over cap [%d]. Setting change to [%d].", amount, characterId, MaxLevel, actual)
				actual = MaxLevel - c.Level()
			}
			current = c.Level() + actual

			err = dynamicUpdate(tx)(SetLevel(current))(c)
			if err != nil {
				return err
			}
			return nil
		})
		if txErr != nil {
			p.l.WithError(txErr).Errorf("Could not award character [%d] [%d] level(s).", characterId, actual)
			return txErr
		}
		_ = mb.Put(character2.EnvEventTopicCharacterStatus, levelChangedEventProvider(transactionId, characterId, channel, actual, current))
		_ = mb.Put(character2.EnvEventTopicCharacterStatus, statChangedProvider(transactionId, channel, characterId, []stat.Type{stat.TypeLevel}, nil))
		return nil
	}
}

func (p *ProcessorImpl) Move(characterId uint32, x int16, y int16, fh int16, stance byte) error {
	GetTemporalRegistry().Update(p.ctx, tenant.MustFromContext(p.ctx), characterId, x, y, fh, stance)
	return nil
}

func (p *ProcessorImpl) RequestChangeMeso(transactionId uuid.UUID, characterId uint32, amount int32, actorId uint32, actorType string, showEffect bool) error {
	var rejectEmit func() error
	txErr := database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
		c, err := p.WithTransaction(tx).GetById()(characterId)
		if err != nil {
			p.l.WithError(err).Errorf("Unable to retrieve character [%d] who is having their meso adjusted.", characterId)
			return err
		}
		if int64(c.Meso())+int64(amount) < 0 {
			p.l.Debugf("Request for character [%d] would leave their meso negative. Amount [%d]. Existing [%d].", characterId, amount, c.Meso())
			rejectEmit = func() error {
				return producer.ProviderImpl(p.l)(p.ctx)(character2.EnvEventTopicCharacterStatus)(notEnoughMesoErrorStatusEventProvider(transactionId, characterId, c.WorldId(), amount))
			}
			return ErrNotEnoughMeso
		}
		if amount > 0 && uint32(amount) > (math.MaxUint32-c.Meso()) {
			p.l.Errorf("Transaction for character [%d] would result in a uint32 overflow. Rejecting transaction.", characterId)
			return ErrMesoOverflow
		}

		if err = dynamicUpdate(tx)(SetMeso(uint32(int64(c.Meso()) + int64(amount))))(c); err != nil {
			return err
		}
		return message.Emit(outbox.EmitProvider(p.l, p.ctx, tx))(func(buf *message.Buffer) error {
			if err := buf.Put(character2.EnvEventTopicCharacterStatus, mesoChangedStatusEventProvider(transactionId, characterId, c.WorldId(), amount, actorId, actorType, showEffect)); err != nil {
				return err
			}
			return buf.Put(character2.EnvEventTopicCharacterStatus, statChangedProvider(transactionId, channel.NewModel(c.WorldId(), 0), characterId, []stat.Type{stat.TypeMeso}, nil))
		})
	})
	if errors.Is(txErr, ErrNotEnoughMeso) && rejectEmit != nil {
		_ = rejectEmit()
		return nil
	}
	return txErr
}

func (p *ProcessorImpl) AttemptMesoPickUp(transactionId uuid.UUID, field field.Model, characterId uint32, dropId uint32, meso uint32) error {
	txErr := database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
		c, err := p.WithTransaction(tx).GetById()(characterId)
		if err != nil {
			p.l.WithError(err).Errorf("Unable to retrieve character [%d] who is having their meso adjusted.", characterId)
			return err
		}
		if meso > (math.MaxUint32 - c.Meso()) {
			p.l.Errorf("Transaction for character [%d] would result in a uint32 overflow. Rejecting transaction.", characterId)
			return ErrMesoOverflow
		}

		if err = dynamicUpdate(tx)(SetMeso(uint32(int64(c.Meso()) + int64(meso))))(c); err != nil {
			return err
		}
		return message.Emit(outbox.EmitProvider(p.l, p.ctx, tx))(func(buf *message.Buffer) error {
			return buf.Put(character2.EnvEventTopicCharacterStatus, statChangedProvider(transactionId, channel.NewModel(field.WorldId(), field.ChannelId()), characterId, []stat.Type{stat.TypeMeso}, nil))
		})
	})
	if txErr != nil {
		return txErr
	}
	return drop.NewProcessor(p.l, p.ctx).RequestPickUp(field, dropId, characterId)
}

func (p *ProcessorImpl) RequestDropMeso(transactionId uuid.UUID, field field.Model, characterId uint32, amount uint32) error {
	var rejectEmit func() error
	txErr := database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
		c, err := p.WithTransaction(tx).GetById()(characterId)
		if err != nil {
			p.l.WithError(err).Errorf("Unable to retrieve character [%d] who is having their meso adjusted.", characterId)
			return err
		}
		if int64(c.Meso())-int64(amount) < 0 {
			p.l.Debugf("Request for character [%d] would leave their meso negative. Amount [%d]. Existing [%d].", characterId, amount, c.Meso())
			rejectEmit = func() error {
				return producer.ProviderImpl(p.l)(p.ctx)(character2.EnvEventTopicCharacterStatus)(notEnoughMesoErrorStatusEventProvider(transactionId, characterId, c.WorldId(), int32(amount)))
			}
			return ErrNotEnoughMeso
		}

		if err = dynamicUpdate(tx)(SetMeso(c.Meso() - amount))(c); err != nil {
			return err
		}
		return message.Emit(outbox.EmitProvider(p.l, p.ctx, tx))(func(buf *message.Buffer) error {
			return buf.Put(character2.EnvEventTopicCharacterStatus, statChangedProvider(transactionId, channel.NewModel(field.WorldId(), field.ChannelId()), characterId, []stat.Type{stat.TypeMeso}, nil))
		})
	})
	if errors.Is(txErr, ErrNotEnoughMeso) && rejectEmit != nil {
		_ = rejectEmit()
		return nil
	}
	if txErr != nil {
		return txErr
	}

	tc := GetTemporalRegistry().GetById(p.ctx, tenant.MustFromContext(p.ctx), characterId)
	// TODO determine appropriate drop type and mod
	_ = drop.NewProcessor(p.l, p.ctx).CreateForMesos(field, amount, 2, tc.X(), tc.Y(), characterId)
	return nil
}

func (p *ProcessorImpl) RequestChangeFame(transactionId uuid.UUID, characterId uint32, amount int8, actorId uint32, actorType string) error {
	return database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
		c, err := p.WithTransaction(tx).GetById()(characterId)
		if err != nil {
			p.l.WithError(err).Errorf("Unable to retrieve character [%d] who is having their fame adjusted.", characterId)
			return err
		}

		total := c.Fame() + int16(amount)
		if err = dynamicUpdate(tx)(SetFame(total))(c); err != nil {
			return err
		}
		return message.Emit(outbox.EmitProvider(p.l, p.ctx, tx))(func(buf *message.Buffer) error {
			if err := buf.Put(character2.EnvEventTopicCharacterStatus, fameChangedStatusEventProvider(transactionId, characterId, c.WorldId(), amount, actorId, actorType)); err != nil {
				return err
			}
			return buf.Put(character2.EnvEventTopicCharacterStatus, statChangedProvider(transactionId, channel.NewModel(c.WorldId(), 0), characterId, []stat.Type{stat.TypeFame}, nil))
		})
	})
}

type Distribution struct {
	Ability string
	Amount  int8
}

func (p *ProcessorImpl) RequestDistributeAp(transactionId uuid.UUID, characterId uint32, distributions []Distribution) error {
	var rejectEmit func() error
	txErr := database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
		c, err := p.WithTransaction(tx).GetById()(characterId)
		if err != nil {
			return err
		}
		if c.AP() < uint16(len(distributions)) {
			rejectEmit = func() error {
				return producer.ProviderImpl(p.l)(p.ctx)(character2.EnvEventTopicCharacterStatus)(statChangedProvider(transactionId, channel.NewModel(c.WorldId(), 0), characterId, []stat.Type{}, nil))
			}
			return errors.New("not enough ap")
		}

		eufs := make([]EntityUpdateFunction, 0)
		stats := make([]stat.Type, 0)
		values := make(map[string]interface{})

		spent := uint16(0)
		for _, d := range distributions {
			switch d.Ability {
			case CommandDistributeApAbilityStrength:
				newVal := uint16(int16(c.Strength()) + int16(d.Amount))
				eufs = append(eufs, SetStrength(newVal))
				stats = append(stats, stat.TypeStrength)
				values["strength"] = newVal
				break
			case CommandDistributeApAbilityDexterity:
				newVal := uint16(int16(c.Dexterity()) + int16(d.Amount))
				eufs = append(eufs, SetDexterity(newVal))
				stats = append(stats, stat.TypeDexterity)
				values["dexterity"] = newVal
				break
			case CommandDistributeApAbilityIntelligence:
				newVal := uint16(int16(c.Intelligence()) + int16(d.Amount))
				eufs = append(eufs, SetIntelligence(newVal))
				stats = append(stats, stat.TypeIntelligence)
				values["intelligence"] = newVal
				break
			case CommandDistributeApAbilityLuck:
				newVal := uint16(int16(c.Luck()) + int16(d.Amount))
				eufs = append(eufs, SetLuck(newVal))
				stats = append(stats, stat.TypeLuck)
				values["luck"] = newVal
				break
			case CommandDistributeApAbilityHp:
				hpGrowth, err := p.getMaxHpGrowth(c)
				if err != nil {
					return err
				}
				newVal := uint16(int16(hpGrowth) * int16(d.Amount))
				eufs = append(eufs, SetMaxHp(newVal))
				eufs = append(eufs, SetHpMpUsed(c.HpMpUsed()+int(d.Amount)))
				stats = append(stats, stat.TypeMaxHp)
				values["max_hp"] = newVal
				break
			case CommandDistributeApAbilityMp:
				mpGrowth, err := p.getMaxMpGrowth(c)
				if err != nil {
					return err
				}
				newVal := uint16(int16(mpGrowth) * int16(d.Amount))
				eufs = append(eufs, SetMaxMp(newVal))
				eufs = append(eufs, SetHpMpUsed(c.HpMpUsed()+int(d.Amount)))
				stats = append(stats, stat.TypeMaxMp)
				values["max_mp"] = newVal
				break
			}
			spent = uint16(int16(spent) + int16(d.Amount))
		}

		if len(eufs) == 0 {
			rejectEmit = func() error {
				return producer.ProviderImpl(p.l)(p.ctx)(character2.EnvEventTopicCharacterStatus)(statChangedProvider(transactionId, channel.NewModel(c.WorldId(), 0), characterId, []stat.Type{}, nil))
			}
			return errors.New("invalid ability")
		}

		eufs = append(eufs, SetAP(c.AP()-spent))
		stats = append(stats, stat.TypeAvailableAP)

		err = dynamicUpdate(tx)(eufs...)(c)
		if err != nil {
			rejectEmit = func() error {
				return producer.ProviderImpl(p.l)(p.ctx)(character2.EnvEventTopicCharacterStatus)(statChangedProvider(transactionId, channel.NewModel(c.WorldId(), 0), characterId, []stat.Type{stat.TypeAvailableAP}, nil))
			}
			return err
		}

		return message.Emit(outbox.EmitProvider(p.l, p.ctx, tx))(func(buf *message.Buffer) error {
			return buf.Put(character2.EnvEventTopicCharacterStatus, statChangedProvider(transactionId, channel.NewModel(c.WorldId(), 0), characterId, stats, values))
		})
	})
	if txErr != nil && rejectEmit != nil {
		_ = rejectEmit()
	}
	return txErr
}

func (p *ProcessorImpl) RequestDistributeSp(transactionId uuid.UUID, characterId uint32, skillId uint32, amount int8) error {
	var c Model
	txErr := database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
		var err error
		c, err = p.WithTransaction(tx).GetById(p.SkillModelDecorator)(characterId)
		if err != nil {
			return err
		}
		sjid, ok := job.FromSkillId(skill.Id(skillId))
		if !ok {
			return errors.New("unable to locate job from skill")
		}
		// NOT routed through p.set().Skill.Resolve(): the generated
		// per-version skill Identity Sets (task-187) do not contain any of
		// the Evan multi-book skill wire ids (e.g. EvanStage2FireCircleId
		// 22101000) at ANY provisioned version -- forcing a Resolve() here
		// would make Evan SP distribution fail outright ("unable to locate
		// job from skill" for every Evan stage skill), a real regression.
		// job.FromSkillId's floor(skillId/10000) only needs to land in the
		// Evan job range (2210-2218) for getSkillBook to answer correctly;
		// that range is disjoint from the audit's one divergent job set
		// (wire 500/510/900/910 GM<->Pirate), so a direct numeric
		// Id->Identity cast (the two types share the same canonical
		// v83-era numbering, see job/identity.go) is exact for Evan and a
		// behavior-preserving no-op for the disjoint GM/Pirate case
		// (getSkillBook returns 0 either way).
		sb := getSkillBook(job.Identity(sjid.Id()))
		if c.SP(sb) < uint32(amount) {
			return errors.New("not enough sp")
		}
		if err = dynamicUpdate(tx)(SetSP(c.SP(sb)-uint32(amount), uint32(sb)))(c); err != nil {
			return err
		}
		return message.Emit(outbox.EmitProvider(p.l, p.ctx, tx))(func(buf *message.Buffer) error {
			return buf.Put(character2.EnvEventTopicCharacterStatus, statChangedProvider(transactionId, channel.NewModel(c.WorldId(), 0), characterId, []stat.Type{stat.TypeAvailableSP}, nil))
		})
	})
	if txErr != nil {
		// Logged, not just returned: every caller of this method discards the
		// error (the Kafka consumer and the channel handler both `_ =` it) and
		// nothing is written back to the client, so a rejected distribution --
		// "not enough sp" being the common one -- is otherwise indistinguishable
		// from a dropped packet. Without this line the only way to tell why a
		// skill did not level is to read the character row by hand.
		p.l.WithError(txErr).Errorf("Unable to distribute [%d] sp to skill [%d] for character [%d].", amount, skillId, characterId)
		return txErr
	}

	// The SP has already been debited by the transaction above; if the skill
	// create/update fails the character is left short the SP with nothing to
	// show for it, which is exactly the state worth a loud log.
	if val := c.GetSkill(skillId); val.Id() != skillId {
		if err := skill2.NewProcessor(p.l, p.ctx).RequestCreate(characterId, skillId, byte(amount), 0, time.Time{}); err != nil {
			p.l.WithError(err).Errorf("Unable to create skill [%d] for character [%d] after debiting [%d] sp.", skillId, characterId, amount)
			return err
		}
	} else {
		if err := skill2.NewProcessor(p.l, p.ctx).RequestUpdate(characterId, skillId, val.Level()+byte(amount), val.MasterLevel(), val.Expiration()); err != nil {
			p.l.WithError(err).Errorf("Unable to update skill [%d] for character [%d] after debiting [%d] sp.", skillId, characterId, amount)
			return err
		}
	}
	return nil
}

func (p *ProcessorImpl) getMaxHpGrowth(c Model) (uint16, error) {
	if c.MaxHp() >= 30000 || c.HpMpUsed() > 9999 {
		return c.MaxHp(), errors.New("max ap to hp")
	}
	var improvingHPSkillId skill.Id
	resMax := c.MaxHp()
	// DIVERGENT (task-187 audit): c.JobId() is a version-specific wire id --
	// wire 500/510 is Pirate/Brawler at v0.61+ but GM/SuperGM at v0.48 (job
	// 900/910 doesn't exist at v0.48). A raw job.IsA(c.JobId(), job.PirateId,
	// ...) compare would misclassify a v0.48 GM as a Pirate. Resolve once to
	// this version's job Identity and branch on that; the other branches in
	// this if/elif chain are converted too (rather than left half-raw) since
	// they all test the same resolved jid.
	jid, jok := p.set().Job.Resolve(c.JobId())
	if jok && job.IsAIdentity(jid,
		job.Warrior,
		job.Fighter, job.Crusader, job.Hero,
		job.Page, job.Crusader, job.WhiteKnight,
		job.Spearman, job.DragonKnight, job.DarkKnight,
		job.DawnWarriorStage1, job.DawnWarriorStage2, job.DawnWarriorStage3, job.DawnWarriorStage4,
		job.AranStage1, job.AranStage2, job.AranStage3, job.AranStage4) {
		if job.IsCygnusIdentity(jid) {
			improvingHPSkillId = skill.DawnWarriorStage1ImprovedMaxHpIncreaseId
		} else {
			improvingHPSkillId = skill.WarriorImprovedMaxHpIncreaseId
		}
		resMax += 20
	} else if jok && job.IsAIdentity(jid,
		job.Magician,
		job.FirePoisonWizard, job.FirePoisonMagician, job.FirePoisonArchMagician,
		job.IceLightningWizard, job.IceLightningMagician, job.IceLightningArchMagician,
		job.Cleric, job.Priest, job.Bishop,
		job.BlazeWizardStage1, job.BlazeWizardStage2, job.BlazeWizardStage3, job.BlazeWizardStage4) {
		resMax += 6
	} else if jok && job.IsAIdentity(jid,
		job.Bowman,
		job.Hunter, job.Ranger, job.Bowmaster,
		job.Crossbowman, job.Sniper, job.Marksman,
		job.WindArcherStage1, job.WindArcherStage2, job.WindArcherStage3, job.WindArcherStage4,
		job.Rogue,
		job.Assassin, job.Hermit, job.NightLord,
		job.Bandit, job.ChiefBandit, job.Shadower,
		job.NightWalkerStage1, job.NightWalkerStage2, job.NightWalkerStage3, job.NightWalkerStage4) {
		resMax += 16
	} else if jok && job.IsAIdentity(jid,
		job.Pirate,
		job.Brawler, job.Marauder, job.Buccaneer,
		job.Gunslinger, job.Outlaw, job.Corsair,
		job.ThunderBreakerStage1, job.ThunderBreakerStage2, job.ThunderBreakerStage3, job.ThunderBreakerStage4) {
		if job.IsCygnusIdentity(jid) {
			improvingHPSkillId = skill.ThunderBreakerStage2ImprovedMaxHpIncreaseId
		} else {
			improvingHPSkillId = skill.BrawlerImproveMaxHpId
		}
		resMax += 18
	} else {
		resMax += 8
	}

	if improvingHPSkillId > 0 {
		improvingHPSkillLevel := c.GetSkillLevel(uint32(improvingHPSkillId))
		se, err := p.sdp.GetEffect(uint32(improvingHPSkillId), improvingHPSkillLevel)
		if err == nil {
			resMax = uint16(int16(resMax) + se.Y())
		}
	}
	return resMax, nil
}

func (p *ProcessorImpl) getMaxMpGrowth(c Model) (uint16, error) {
	if c.MaxMp() >= 30000 || c.HpMpUsed() > 9999 {
		return c.MaxMp(), errors.New("max ap to mp")
	}
	var improvingMPSkillId skill.Id
	resMax := c.MaxMp()
	// DIVERGENT (task-187 audit): same wire 500/510 GM/SuperGM-vs-Pirate/
	// Brawler collision as getMaxHpGrowth above -- resolve to Identity once
	// and branch on it (all sibling branches converted for the same reason).
	jid, jok := p.set().Job.Resolve(c.JobId())
	if jok && job.IsAIdentity(jid,
		job.Warrior,
		job.Fighter, job.Crusader, job.Hero,
		job.Page, job.Crusader, job.WhiteKnight,
		job.Spearman, job.DragonKnight, job.DarkKnight,
		job.DawnWarriorStage1, job.DawnWarriorStage2, job.DawnWarriorStage3, job.DawnWarriorStage4,
		job.AranStage1, job.AranStage2, job.AranStage3, job.AranStage4) {
		if job.IsAIdentity(jid, job.Crusader, job.WhiteKnight) {
			improvingMPSkillId = skill.WhiteKnightImprovingMpRecoveryId
		} else if job.IsAIdentity(jid, job.DawnWarriorStage3, job.DawnWarriorStage4) {
			improvingMPSkillId = skill.DawnWarriorStage3ImprovedMpRecoveryId
		}
		resMax += 2
	} else if jok && job.IsAIdentity(jid,
		job.Magician,
		job.FirePoisonWizard, job.FirePoisonMagician, job.FirePoisonArchMagician,
		job.IceLightningWizard, job.IceLightningMagician, job.IceLightningArchMagician,
		job.Cleric, job.Priest, job.Bishop,
		job.BlazeWizardStage1, job.BlazeWizardStage2, job.BlazeWizardStage3, job.BlazeWizardStage4) {
		if job.IsCygnusIdentity(jid) {
			improvingMPSkillId = skill.BlazeWizardStage1ImprovedMaxMpIncreaseId
		} else {
			improvingMPSkillId = skill.MagicianImprovedMaxMpIncreaseId
		}
		resMax += 18
	} else if jok && job.IsAIdentity(jid,
		job.Bowman,
		job.Hunter, job.Ranger, job.Bowmaster,
		job.Crossbowman, job.Sniper, job.Marksman,
		job.Rogue,
		job.Assassin, job.Hermit, job.NightLord,
		job.Bandit, job.ChiefBandit, job.Shadower,
		job.WindArcherStage1, job.WindArcherStage2, job.WindArcherStage3, job.WindArcherStage4,
		job.NightWalkerStage1, job.NightWalkerStage2, job.NightWalkerStage3, job.NightWalkerStage4) {
		resMax += 10
	} else if jok && job.IsAIdentity(jid,
		job.Pirate,
		job.Brawler, job.Marauder, job.Buccaneer,
		job.Gunslinger, job.Outlaw, job.Corsair,
		job.ThunderBreakerStage1, job.ThunderBreakerStage2, job.ThunderBreakerStage3, job.ThunderBreakerStage4) {
		resMax += 14
	} else {
		resMax += 6
	}

	// Use effective intelligence (includes buffs, equipment) with fallback to base
	intelligence := c.Intelligence()
	ch := channel.NewModel(c.WorldId(), 0)
	effectiveStats, err := effective_stats.RequestByCharacter(ch, c.Id())(p.l, p.ctx)
	if err == nil {
		intelligence = uint16(effectiveStats.Intelligence)
	} else {
		p.l.WithError(err).Warnf("Failed to fetch effective stats for character [%d], using base intelligence", c.Id())
	}
	resMax += uint16(math.Ceil(float64(intelligence) / 10))

	if improvingMPSkillId > 0 {
		improvingMPSkillLevel := c.GetSkillLevel(uint32(improvingMPSkillId))
		se, err := p.sdp.GetEffect(uint32(improvingMPSkillId), improvingMPSkillLevel)
		if err == nil {
			resMax = uint16(int16(resMax) + se.X())
		}
	}

	return resMax, nil
}

// enforceBounds adds change to current and clamps the result into
// [lowerBound, upperBound]. The intermediate sum is computed in int32
// so that uint16 saturation (e.g. current=30000 + change=+5000) cannot
// overflow into negative space and clamp to the lower bound.
func enforceBounds(change int16, current uint16, upperBound uint16, lowerBound uint16) uint16 {
	adjusted := int32(current) + int32(change)
	if adjusted < int32(lowerBound) {
		return lowerBound
	}
	if adjusted > int32(upperBound) {
		return upperBound
	}
	return uint16(adjusted)
}

// resolveEffectiveMax picks the max value to use for bounding HP/MP adjustments.
// If the effective-stats fetch failed OR returned zero, we fall back to the
// character's base max. Treating a zero response as authoritative clamps any
// HP/MP adjustment to zero and, for HP, emits a bogus DIED event on what was
// actually a routine regen tick. The zero-response path is logged at WARN so
// the underlying stats-service regression stays visible.
func resolveEffectiveMax(l logrus.FieldLogger, base uint16, effective uint32, fetchErr error, characterId uint32, statName string) uint16 {
	if fetchErr != nil {
		l.WithError(fetchErr).Debugf("Failed to fetch effective stats for character [%d], using base %s", characterId, statName)
		return base
	}
	if effective == 0 {
		l.Warnf("Effective stats for character [%d] reported %s=0; falling back to base %s=[%d] to avoid clamp-to-zero death", characterId, statName, statName, base)
		return base
	}
	// Defensive: even though atlas-effective-stats caps MaxHp/MaxMp at
	// 30000, a stale or untrusted upstream could return a value larger
	// than uint16 max which would silently wrap on cast. Clamp instead.
	if effective > math.MaxUint16 {
		return math.MaxUint16
	}
	return uint16(effective)
}

func (p *ProcessorImpl) ChangeHPAndEmit(transactionId uuid.UUID, channel channel.Model, characterId uint32, amount int16) error {
	return database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
		return message.Emit(outbox.EmitProvider(p.l, p.ctx, tx))(func(buf *message.Buffer) error {
			return p.WithTransaction(tx).ChangeHP(buf)(transactionId, channel, characterId, amount)
		})
	})
}

func (p *ProcessorImpl) ChangeHP(mb *message.Buffer) func(transactionId uuid.UUID, channel channel.Model, characterId uint32, amount int16) error {
	return func(transactionId uuid.UUID, channel channel.Model, characterId uint32, amount int16) error {
		var adjusted uint16
		txErr := database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
			c, err := p.WithTransaction(tx).GetById()(characterId)
			if err != nil {
				return err
			}

			// Use effective MaxHP when the stats service reports a positive value;
			// otherwise fall back to the character's base MaxHP to avoid clamping
			// a positive regen tick to zero (which would emit a spurious DIED).
			effectiveStats, err := effective_stats.RequestByCharacter(channel, c.Id())(p.l, p.ctx)
			maxHP := resolveEffectiveMax(p.l, c.MaxHp(), effectiveStats.MaxHp, err, c.Id(), "MaxHP")

			adjusted = enforceBounds(amount, c.Hp(), maxHP, 0)
			p.l.Debugf("Attempting to adjust character [%d] health by [%d] to [%d].", characterId, amount, adjusted)
			return dynamicUpdate(tx)(SetHealth(adjusted))(c)
		})
		if txErr != nil {
			return txErr
		}

		if adjusted == 0 {
			f, lerr := location.GetField(p.l, p.ctx, characterId)
			if lerr != nil {
				if errors.Is(lerr, location.ErrNotFound) {
					p.l.Warnf("ChangeHP: no atlas-maps location for [%d] (likely first login of new character); emitting DIED with zero map.", characterId)
				} else {
					p.l.WithError(lerr).Errorf("ChangeHP: atlas-maps lookup failed for [%d] (infrastructure error); emitting DIED with zero map.", characterId)
				}
			}
			_ = mb.Put(character2.EnvEventTopicCharacterStatus, diedEventProvider(transactionId, characterId, channel, f.MapId(), 0, character2.KillerTypeUnknown))
		}

		_ = mb.Put(character2.EnvEventTopicCharacterStatus, statChangedProvider(transactionId, channel, characterId, []stat.Type{stat.TypeHp}, nil))
		return nil
	}
}

func (p *ProcessorImpl) SetHPAndEmit(transactionId uuid.UUID, channel channel.Model, characterId uint32, amount uint16) error {
	return database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
		return message.Emit(outbox.EmitProvider(p.l, p.ctx, tx))(func(buf *message.Buffer) error {
			return p.WithTransaction(tx).SetHP(buf)(transactionId, channel, characterId, amount)
		})
	})
}

func (p *ProcessorImpl) SetHP(mb *message.Buffer) func(transactionId uuid.UUID, channel channel.Model, characterId uint32, amount uint16) error {
	return func(transactionId uuid.UUID, channel channel.Model, characterId uint32, amount uint16) error {
		var clamped uint16
		txErr := database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
			c, err := p.WithTransaction(tx).GetById()(characterId)
			if err != nil {
				return err
			}

			// Use effective MaxHP when the stats service reports a positive value;
			// otherwise fall back to the character's base MaxHP to avoid a
			// SetHP(>=0) being clamped to zero and emitting a spurious DIED.
			effectiveStats, err := effective_stats.RequestByCharacter(channel, c.Id())(p.l, p.ctx)
			maxHP := resolveEffectiveMax(p.l, c.MaxHp(), effectiveStats.MaxHp, err, c.Id(), "MaxHP")

			// Clamp amount between 0 and effective MaxHP
			clamped = amount
			if clamped > maxHP {
				clamped = maxHP
			}
			p.l.Debugf("Setting character [%d] health to [%d].", characterId, clamped)
			return dynamicUpdate(tx)(SetHealth(clamped))(c)
		})
		if txErr != nil {
			return txErr
		}

		if clamped == 0 {
			f, lerr := location.GetField(p.l, p.ctx, characterId)
			if lerr != nil {
				if errors.Is(lerr, location.ErrNotFound) {
					p.l.Warnf("SetHP: no atlas-maps location for [%d] (likely first login of new character); emitting DIED with zero map.", characterId)
				} else {
					p.l.WithError(lerr).Errorf("SetHP: atlas-maps lookup failed for [%d] (infrastructure error); emitting DIED with zero map.", characterId)
				}
			}
			_ = mb.Put(character2.EnvEventTopicCharacterStatus, diedEventProvider(transactionId, characterId, channel, f.MapId(), 0, character2.KillerTypeUnknown))
		}

		_ = mb.Put(character2.EnvEventTopicCharacterStatus, statChangedProvider(transactionId, channel, characterId, []stat.Type{stat.TypeHp}, nil))
		return nil
	}
}

func (p *ProcessorImpl) ChangeMPAndEmit(transactionId uuid.UUID, channel channel.Model, characterId uint32, amount int16) error {
	return database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
		return message.Emit(outbox.EmitProvider(p.l, p.ctx, tx))(func(buf *message.Buffer) error {
			return p.WithTransaction(tx).ChangeMP(buf)(transactionId, channel, characterId, amount)
		})
	})
}

func (p *ProcessorImpl) ChangeMP(mb *message.Buffer) func(transactionId uuid.UUID, channel channel.Model, characterId uint32, amount int16) error {
	return func(transactionId uuid.UUID, channel channel.Model, characterId uint32, amount int16) error {
		txErr := database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
			c, err := p.WithTransaction(tx).GetById()(characterId)
			if err != nil {
				return err
			}

			// Use effective MaxMP when the stats service reports a positive value;
			// otherwise fall back to the character's base MaxMP (same reasoning
			// as ChangeHP — a zero cap would clamp routine MP regen to 0).
			effectiveStats, err := effective_stats.RequestByCharacter(channel, c.Id())(p.l, p.ctx)
			maxMP := resolveEffectiveMax(p.l, c.MaxMp(), effectiveStats.MaxMp, err, c.Id(), "MaxMP")

			adjusted := enforceBounds(amount, c.Mp(), maxMP, 0)
			p.l.Debugf("Attempting to adjust character [%d] mana by [%d] to [%d].", characterId, amount, adjusted)
			return dynamicUpdate(tx)(SetMana(adjusted))(c)
		})
		if txErr != nil {
			return txErr
		}
		_ = mb.Put(character2.EnvEventTopicCharacterStatus, statChangedProvider(transactionId, channel, characterId, []stat.Type{stat.TypeMp}, nil))
		return nil
	}
}

func (p *ProcessorImpl) ClampHPAndEmit(transactionId uuid.UUID, channel channel.Model, characterId uint32, maxValue uint16) error {
	return database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
		return message.Emit(outbox.EmitProvider(p.l, p.ctx, tx))(func(buf *message.Buffer) error {
			return p.WithTransaction(tx).ClampHP(buf)(transactionId, channel, characterId, maxValue)
		})
	})
}

func (p *ProcessorImpl) ClampHP(mb *message.Buffer) func(transactionId uuid.UUID, channel channel.Model, characterId uint32, maxValue uint16) error {
	return func(transactionId uuid.UUID, channel channel.Model, characterId uint32, maxValue uint16) error {
		var clamped bool
		txErr := database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
			c, err := p.WithTransaction(tx).GetById()(characterId)
			if err != nil {
				return err
			}

			// Only clamp if current HP exceeds the new max value
			if c.Hp() > maxValue {
				p.l.Debugf("Clamping character [%d] HP from [%d] to [%d] (effective max decreased).", characterId, c.Hp(), maxValue)
				clamped = true
				return dynamicUpdate(tx)(SetHealth(maxValue))(c)
			}
			return nil
		})
		if txErr != nil {
			return txErr
		}

		if clamped {
			_ = mb.Put(character2.EnvEventTopicCharacterStatus, statChangedProvider(transactionId, channel, characterId, []stat.Type{stat.TypeHp}, nil))
		}
		return nil
	}
}

func (p *ProcessorImpl) ClampMPAndEmit(transactionId uuid.UUID, channel channel.Model, characterId uint32, maxValue uint16) error {
	return database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
		return message.Emit(outbox.EmitProvider(p.l, p.ctx, tx))(func(buf *message.Buffer) error {
			return p.WithTransaction(tx).ClampMP(buf)(transactionId, channel, characterId, maxValue)
		})
	})
}

func (p *ProcessorImpl) ClampMP(mb *message.Buffer) func(transactionId uuid.UUID, channel channel.Model, characterId uint32, maxValue uint16) error {
	return func(transactionId uuid.UUID, channel channel.Model, characterId uint32, maxValue uint16) error {
		var clamped bool
		txErr := database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
			c, err := p.WithTransaction(tx).GetById()(characterId)
			if err != nil {
				return err
			}

			// Only clamp if current MP exceeds the new max value
			if c.Mp() > maxValue {
				p.l.Debugf("Clamping character [%d] MP from [%d] to [%d] (effective max decreased).", characterId, c.Mp(), maxValue)
				clamped = true
				return dynamicUpdate(tx)(SetMana(maxValue))(c)
			}
			return nil
		})
		if txErr != nil {
			return txErr
		}

		if clamped {
			_ = mb.Put(character2.EnvEventTopicCharacterStatus, statChangedProvider(transactionId, channel, characterId, []stat.Type{stat.TypeMp}, nil))
		}
		return nil
	}
}

func (p *ProcessorImpl) ProcessLevelChangeAndEmit(transactionId uuid.UUID, channel channel.Model, characterId uint32, amount byte) error {
	return database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
		return message.Emit(outbox.EmitProvider(p.l, p.ctx, tx))(func(buf *message.Buffer) error {
			return p.WithTransaction(tx).ProcessLevelChange(buf)(transactionId, channel, characterId, amount)
		})
	})
}

func (p *ProcessorImpl) ProcessLevelChange(mb *message.Buffer) func(transactionId uuid.UUID, channel channel.Model, characterId uint32, amount byte) error {
	return func(transactionId uuid.UUID, channel channel.Model, characterId uint32, amount byte) error {
		var addedAP uint16
		var addedSP uint32
		var addedHP uint16
		var addedMP uint16
		var addedStr uint16
		var addedDex uint16
		sus := []stat.Type{stat.TypeAvailableAP, stat.TypeAvailableSP, stat.TypeHp, stat.TypeMaxHp, stat.TypeMp, stat.TypeMaxMp}

		var newMaxHP, newMaxMP uint16
		var newStr, newDex uint16
		var newInt uint16

		txErr := database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
			c, err := p.WithTransaction(tx).GetById(p.SkillModelDecorator)(characterId)
			if err != nil {
				return err
			}

			effectiveLevel := c.Level() - amount
			hpMPParams := p.resolveHPMPGainParams(c)

			for i := range amount {
				effectiveLevel = effectiveLevel + i + 1

				if appliesAutoAP(p.t) {
					if job.IsBeginner(c.JobId()) && effectiveLevel < 11 {
						if effectiveLevel < 6 {
							addedStr += 5
						} else {
							addedStr += 4
							addedDex += 1
						}
					} else {
						addedAP += computeOnLevelAddedAP(c.JobId(), effectiveLevel)
					}
				} else {
					addedAP += computeOnLevelAddedAP(c.JobId(), effectiveLevel)
				}

				addedSP += computeOnLevelAddedSP(c.JobId(), effectiveLevel)
				aHP, aMP := rollHPMPGain(hpMPParams)
				addedHP += aHP
				addedMP += aMP
			}

			p.l.Debugf("As a result of processing a level change of [%d]. Character [%d] will gain [%d] AP, [%d] SP, [%d] HP, and [%d] MP.", amount, characterId, addedAP, addedSP, addedHP, addedMP)
			// getSkillBook only branches on the Evan multi-book range
			// (2210-2218), which never collides with the audit's divergent
			// job set (500/510/900/910) -- an unresolved jid still yields
			// the correct "book 0" answer via the zero Identity fallback.
			jid, _ := p.set().Job.Resolve(c.JobId())
			sb := getSkillBook(jid)

			newMaxHP = c.MaxHp() + addedHP
			newMaxMP = c.MaxMp() + addedMP
			newInt = c.Intelligence()

			eufs := []EntityUpdateFunction{
				SetAP(c.AP() + addedAP),
				SetSP(c.SP(sb)+addedSP, uint32(sb)),
				SetHealth(newMaxHP),
				SetMaxHp(newMaxHP),
				SetMana(newMaxMP),
				SetMaxMp(newMaxMP),
			}

			if addedStr > 0 {
				newStr = c.Strength() + addedStr
				eufs = append(eufs, SetStrength(newStr))
				sus = append(sus, stat.TypeStrength)
			}
			if addedDex > 0 {
				newDex = c.Dexterity() + addedDex
				eufs = append(eufs, SetDexterity(newDex))
				sus = append(sus, stat.TypeDexterity)
			}

			return dynamicUpdate(tx)(eufs...)(c)
		})
		if txErr != nil {
			return txErr
		}

		values := map[string]interface{}{
			"max_hp":       newMaxHP,
			"max_mp":       newMaxMP,
			"intelligence": newInt,
		}
		if addedStr > 0 {
			values["strength"] = newStr
		}
		if addedDex > 0 {
			values["dexterity"] = newDex
		}

		_ = mb.Put(character2.EnvEventTopicCharacterStatus, statChangedProvider(transactionId, channel, characterId, sus, values))
		return nil
	}
}

// version-stable per task-187 audit (audit/README.md, divergences.csv):
// Cygnus (job type 1xxx) is not part of the audit's divergent job set (only
// wire 500/510/900/910 GM<->Pirate remap across the provisioned GMS
// versions) -- job.IsCygnus(jobId) is safe to leave raw-Id-keyed here.
func computeOnLevelAddedAP(jobId job.Id, level byte) uint16 {
	toGain := uint16(5)
	if job.IsCygnus(jobId) {
		if level > 10 {
			if level <= 17 {
				toGain += 2
			} else if level < 77 {
				toGain += 1
			}
		}
	}
	return toGain
}

// version-stable per task-187 audit (audit/README.md, divergences.csv):
// Beginner/Noblesse/Legend/Evan roots are not part of the audit's divergent
// job set -- job.IsBeginner(jobId) is safe to leave raw-Id-keyed here.
func computeOnLevelAddedSP(jobId job.Id, effectiveLevel byte) uint32 {
	if job.IsBeginner(jobId) {
		if effectiveLevel >= 2 && effectiveLevel <= 7 {
			return 1
		}
		return 0
	}
	return 3
}

type hpMPGainParams struct {
	hpLower uint16
	hpUpper uint16
	mpLower uint16
	mpUpper uint16
	hpBonus int16
	mpBonus int16
}

func (p *ProcessorImpl) resolveHPMPGainParams(c Model) hpMPGainParams {
	var params hpMPGainParams
	var improvingHPSkillId skill.Id
	var improvingMPSkillId skill.Id

	// DIVERGENT (task-187 audit, the bug this function's v0.48 GM test
	// guards against): c.JobId() is a version-specific wire id -- wire
	// 500/510 is Pirate/Brawler at v0.61+ but GM/SuperGM at v0.48 (job
	// 900/910 doesn't exist until v0.61). A raw job.IsA(c.JobId(),
	// job.PirateId, ...) compare matches BOTH a v0.48 GM and a v0.61+
	// Pirate on wire 500, and since the Pirate branch sits after the
	// GM/SuperGM branch in the original chain, a v0.48 GM (which never
	// matches the raw job.GmId/SuperGmId==900/910 check) fell through to
	// the Pirate branch and got 22/28 HP instead of 30000/30000. Resolve
	// c.JobId() to this version's job Identity once and branch on that
	// (GM/SuperGM ordered before Pirate, as in the original); every sibling
	// branch in this chain is converted too, since they all test the same
	// resolved jid and a half-resolved/half-raw chain would be incoherent.
	jid, jok := p.set().Job.Resolve(c.JobId())

	if jok && job.IsBeginnerIdentity(jid) {
		params.hpLower, params.hpUpper = 12, 16
		params.mpLower, params.mpUpper = 10, 12
	} else if jok && job.IsAIdentity(jid,
		job.Warrior,
		job.Fighter, job.Crusader, job.Hero,
		job.Page, job.Crusader, job.WhiteKnight,
		job.Spearman, job.DragonKnight, job.DarkKnight,
		job.DawnWarriorStage1, job.DawnWarriorStage2, job.DawnWarriorStage3, job.DawnWarriorStage4) {
		if job.IsCygnusIdentity(jid) {
			improvingHPSkillId = skill.DawnWarriorStage1ImprovedMaxHpIncreaseId
		} else {
			improvingHPSkillId = skill.WarriorImprovedMaxHpIncreaseId
		}
		if job.IsAIdentity(jid, job.Crusader, job.WhiteKnight) {
			improvingMPSkillId = skill.WhiteKnightImprovingMpRecoveryId
		} else if job.IsAIdentity(jid, job.DawnWarriorStage3, job.DawnWarriorStage4) {
			improvingMPSkillId = skill.DawnWarriorStage3ImprovedMpRecoveryId
		}
		params.hpLower, params.hpUpper = 24, 28
		params.mpLower, params.mpUpper = 4, 6
	} else if jok && job.IsAIdentity(jid,
		job.Magician,
		job.FirePoisonWizard, job.FirePoisonMagician, job.FirePoisonArchMagician,
		job.IceLightningWizard, job.IceLightningMagician, job.IceLightningArchMagician,
		job.Cleric, job.Priest, job.Bishop,
		job.BlazeWizardStage1, job.BlazeWizardStage2, job.BlazeWizardStage3, job.BlazeWizardStage4) {
		if job.IsCygnusIdentity(jid) {
			improvingMPSkillId = skill.BlazeWizardStage1ImprovedMaxMpIncreaseId
		} else {
			improvingMPSkillId = skill.MagicianImprovedMaxMpIncreaseId
		}
		params.hpLower, params.hpUpper = 10, 14
		params.mpLower, params.mpUpper = 22, 24
	} else if jok && job.IsAIdentity(jid,
		job.Bowman,
		job.Hunter, job.Ranger, job.Bowmaster,
		job.Crossbowman, job.Sniper, job.Marksman,
		job.Rogue,
		job.Assassin, job.Hermit, job.NightLord,
		job.Bandit, job.ChiefBandit, job.Shadower,
		job.WindArcherStage1, job.WindArcherStage2, job.WindArcherStage3, job.WindArcherStage4,
		job.NightWalkerStage1, job.NightWalkerStage2, job.NightWalkerStage3, job.NightWalkerStage4) {
		params.hpLower, params.hpUpper = 20, 24
		params.mpLower, params.mpUpper = 14, 16
	} else if jok && job.IsAIdentity(jid, job.Gm, job.SuperGm) {
		params.hpLower, params.hpUpper = 30000, 30000
		params.mpLower, params.mpUpper = 30000, 30000
	} else if jok && job.IsAIdentity(jid,
		job.Pirate,
		job.Brawler, job.Marauder, job.Buccaneer,
		job.Gunslinger, job.Outlaw, job.Corsair,
		job.ThunderBreakerStage1, job.ThunderBreakerStage2, job.ThunderBreakerStage3, job.ThunderBreakerStage4) {
		if job.IsCygnusIdentity(jid) {
			improvingHPSkillId = skill.ThunderBreakerStage2ImprovedMaxHpIncreaseId
		} else {
			improvingHPSkillId = skill.BrawlerImproveMaxHpId
		}
		params.hpLower, params.hpUpper = 22, 28
		params.mpLower, params.mpUpper = 18, 23
	} else if jok && job.IsAIdentity(jid, job.AranStage1, job.AranStage2, job.AranStage3, job.AranStage4) {
		params.hpLower, params.hpUpper = 44, 48
		params.mpLower, params.mpUpper = 4, 8
	}

	if improvingHPSkillId > 0 {
		improvingHPSkillLevel := c.GetSkillLevel(uint32(improvingHPSkillId))
		se, err := p.sdp.GetEffect(uint32(improvingHPSkillId), improvingHPSkillLevel)
		if err == nil {
			params.hpBonus = se.X()
		}
	}
	if improvingMPSkillId > 0 {
		improvingMPSkillLevel := c.GetSkillLevel(uint32(improvingMPSkillId))
		se, err := p.sdp.GetEffect(uint32(improvingMPSkillId), improvingMPSkillLevel)
		if err == nil {
			params.mpBonus = se.X()
		}
	}
	return params
}

func rollHPMPGain(params hpMPGainParams) (uint16, uint16) {
	randBound := func(lower uint16, upper uint16) uint16 {
		return uint16(rand.Float32()*float32(upper-lower+1)) + lower
	}
	addedHP := uint16(int16(randBound(params.hpLower, params.hpUpper)) + params.hpBonus)
	addedMP := uint16(int16(randBound(params.mpLower, params.mpUpper)) + params.mpBonus)
	return addedHP, addedMP
}

func (p *ProcessorImpl) ProcessJobChangeAndEmit(transactionId uuid.UUID, channel channel.Model, characterId uint32, jobId job.Id) error {
	return database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
		return message.Emit(outbox.EmitProvider(p.l, p.ctx, tx))(func(buf *message.Buffer) error {
			return p.WithTransaction(tx).ProcessJobChange(buf)(transactionId, channel, characterId, jobId)
		})
	})
}

func (p *ProcessorImpl) ProcessJobChange(mb *message.Buffer) func(transactionId uuid.UUID, channel channel.Model, characterId uint32, jobId job.Id) error {
	return func(transactionId uuid.UUID, channel channel.Model, characterId uint32, jobId job.Id) error {
		var addedAP uint16
		var addedSP uint32
		var addedHP uint16
		var addedMP uint16
		var newMaxHP, newMaxMP uint16
		var newInt uint16

		randBoundFunc := func(lower uint16, upper uint16) uint16 {
			return uint16(rand.Float32()*float32(upper-lower+1)) + lower
		}

		txErr := database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
			c, err := p.WithTransaction(tx).GetById(p.SkillModelDecorator)(characterId)
			if err != nil {
				return err
			}

			// DIVERGENT (task-187 audit): jobId here is the version-specific
			// wire id of the job being changed TO -- wire 500 is Pirate at
			// v0.61+ but GM at v0.48. A raw job.IsA(jobId, job.PirateId, ...)
			// compare below would misclassify a v0.48 GM job-change as a
			// Pirate job-change (100-150 HP/25-50 MP) instead of falling
			// through to the generic non-beginner award. Resolve jobId to
			// this version's job Identity once and branch on that for the
			// whole chain (kept coherent rather than half-raw).
			jid, jok := p.set().Job.Resolve(jobId)

			// TODO award job change AP is this only Cygnus?
			if jok && job.IsCygnusIdentity(jid) {
				addedAP = 7
			}

			addedSP = 1
			if jok && job.IsAIdentity(jid, job.Evan, job.EvanStage1, job.EvanStage2, job.EvanStage3, job.EvanStage4, job.EvanStage5, job.EvanStage6, job.EvanStage7, job.EvanStage8, job.EvanStage9, job.EvanStage10) {
				addedAP += 2
			} else if jok && job.IsFourthJobIdentity(jid) {
				addedSP += 2
			}

			if jok && job.IsAIdentity(jid, job.Warrior, job.DawnWarriorStage1, job.AranStage1) {
				addedHP = randBoundFunc(200, 250)
			} else if jok && job.IsAIdentity(jid, job.Magician, job.BlazeWizardStage1, job.EvanStage1) {
				addedMP = randBoundFunc(100, 150)
			} else if jok && job.IsAIdentity(jid, job.Bowman, job.Rogue, job.Pirate, job.WindArcherStage1, job.NightWalkerStage1, job.ThunderBreakerStage1) {
				addedHP = randBoundFunc(100, 150)
				addedMP = randBoundFunc(25, 50)
			} else if jok && job.IsAIdentity(jid,
				job.Fighter, job.Crusader, job.Hero,
				job.Page, job.Crusader, job.WhiteKnight,
				job.Spearman, job.DragonKnight, job.DarkKnight,
				job.DawnWarriorStage2, job.DawnWarriorStage3, job.DawnWarriorStage4,
				job.AranStage2, job.AranStage3, job.AranStage4) {
				addedHP = randBoundFunc(300, 350)
			} else if jok && job.IsAIdentity(jid,
				job.FirePoisonWizard, job.FirePoisonMagician, job.FirePoisonArchMagician,
				job.IceLightningWizard, job.IceLightningMagician, job.IceLightningArchMagician,
				job.Cleric, job.Priest, job.Bishop,
				job.BlazeWizardStage2, job.BlazeWizardStage3, job.BlazeWizardStage4,
				job.EvanStage2, job.EvanStage3, job.EvanStage4, job.EvanStage5, job.EvanStage6, job.EvanStage7, job.EvanStage8, job.EvanStage9, job.EvanStage10) {
				addedMP = randBoundFunc(450, 500)
			} else if jok && !job.IsBeginnerIdentity(jid) {
				addedHP = randBoundFunc(300, 350)
				addedMP = randBoundFunc(150, 200)
			}

			newMaxHP = c.MaxHp() + addedHP
			newMaxMP = c.MaxMp() + addedMP
			newInt = c.Intelligence()

			p.l.Debugf("As a result of processing a job change to [%d]. Character [%d] will gain [%d] AP, [%d] SP, [%d] HP, and [%d] MP.", jobId, characterId, addedAP, addedSP, addedHP, addedMP)
			// getSkillBook only branches on the Evan multi-book range
			// (2210-2218), disjoint from the divergent job set above -- an
			// unresolved curJid still yields the correct "book 0" answer.
			curJid, _ := p.set().Job.Resolve(c.JobId())
			sb := getSkillBook(curJid)
			return dynamicUpdate(tx)(SetAP(c.AP()+addedAP), SetSP(c.SP(sb)+addedSP, uint32(sb)), SetHealth(newMaxHP), SetMaxHp(newMaxHP), SetMana(newMaxMP), SetMaxMp(newMaxMP))(c)
		})
		if txErr != nil {
			return txErr
		}

		values := map[string]interface{}{
			"max_hp":       newMaxHP,
			"max_mp":       newMaxMP,
			"intelligence": newInt,
		}

		_ = mb.Put(character2.EnvEventTopicCharacterStatus, statChangedProvider(transactionId, channel, characterId, []stat.Type{stat.TypeAvailableAP, stat.TypeAvailableSP, stat.TypeHp, stat.TypeMaxHp, stat.TypeMp, stat.TypeMaxMp}, values))
		return nil
	}
}

// getSkillBook takes a job Identity (task-187): callers resolve the
// character's/target's version-specific wire job id to this version-blind
// Identity before calling in. The Evan multi-book range (2210-2218) itself
// is version-stable (not part of the audit's divergent set), but the
// signature is Identity-typed for uniformity with its callers, some of
// which already hold a resolved jid for other (divergent) branches in the
// same function.
func getSkillBook(jid job.Identity) int {
	return job.GetSkillBookIdentity(jid)
}

func (p *ProcessorImpl) UpdateAndEmit(transactionId uuid.UUID, characterId uint32, input RestModel) error {
	return database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
		return message.Emit(outbox.EmitProvider(p.l, p.ctx, tx))(func(buf *message.Buffer) error {
			return p.WithTransaction(tx).Update(buf)(transactionId, characterId, input)
		})
	})
}

// fieldChange represents a validated field change with event emission capability
type fieldChange struct {
	updateFunc  EntityUpdateFunction
	eventFunc   func() error
	shouldApply bool
}

func (p *ProcessorImpl) Update(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, input RestModel) error {
	return func(transactionId uuid.UUID, characterId uint32, input RestModel) error {
		return database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
			c, err := p.WithTransaction(tx).GetById()(characterId)
			if err != nil {
				return err
			}

			// Validate and prepare all field changes
			changes := []fieldChange{}

			// Name validation and update
			if input.Name != "" && input.Name != c.Name() {
				if ok, err := p.IsValidName(input.Name); !ok || err != nil {
					if err != nil {
						return err
					}
					return errors.New("invalid or duplicate name")
				}
				changes = append(changes, fieldChange{
					updateFunc:  SetName(input.Name),
					shouldApply: true,
					eventFunc: func() error {
						return mb.Put(character2.EnvEventTopicCharacterStatus, nameChangedEventProvider(transactionId, characterId, c.WorldId(), c.Name(), input.Name))
					},
				})
			}

			// Hair validation and update
			if input.Hair != 0 && input.Hair != c.Hair() {
				if !p.isValidHair(input.Hair) {
					return errors.New("invalid hair ID")
				}
				changes = append(changes, fieldChange{
					updateFunc:  SetHair(input.Hair),
					shouldApply: true,
					eventFunc: func() error {
						return mb.Put(character2.EnvEventTopicCharacterStatus, hairChangedEventProvider(transactionId, characterId, c.WorldId(), c.Hair(), input.Hair))
					},
				})
			}

			// Face validation and update
			if input.Face != 0 && input.Face != c.Face() {
				if !p.isValidFace(input.Face) {
					return errors.New("invalid face ID")
				}
				changes = append(changes, fieldChange{
					updateFunc:  SetFace(input.Face),
					shouldApply: true,
					eventFunc: func() error {
						return mb.Put(character2.EnvEventTopicCharacterStatus, faceChangedEventProvider(transactionId, characterId, c.WorldId(), c.Face(), input.Face))
					},
				})
			}

			// Gender validation and update
			if input.Gender != c.Gender() {
				if !p.isValidGender(input.Gender) {
					return errors.New("invalid gender value")
				}
				changes = append(changes, fieldChange{
					updateFunc:  SetGender(input.Gender),
					shouldApply: true,
					eventFunc: func() error {
						return mb.Put(character2.EnvEventTopicCharacterStatus, genderChangedEventProvider(transactionId, characterId, c.WorldId(), c.Gender(), input.Gender))
					},
				})
			}

			// Skin color validation and update
			if input.SkinColor != 0 && input.SkinColor != c.SkinColor() {
				if !p.isValidSkinColor(input.SkinColor) {
					return errors.New("invalid skin color value")
				}
				changes = append(changes, fieldChange{
					updateFunc:  SetSkinColor(input.SkinColor),
					shouldApply: true,
					eventFunc: func() error {
						return mb.Put(character2.EnvEventTopicCharacterStatus, skinColorChangedEventProvider(transactionId, characterId, c.WorldId(), c.SkinColor(), input.SkinColor))
					},
				})
			}

			// GM validation and update
			// Gm is a pointer: nil = field absent (no change requested);
			// non-nil = explicit set, including 0 (demotion).
			if input.Gm != nil && *input.Gm != c.GM() {
				newGmVal := *input.Gm
				if !p.isValidGm(newGmVal) {
					return errors.New("invalid GM value")
				}
				changes = append(changes, fieldChange{
					updateFunc:  SetGm(newGmVal),
					shouldApply: true,
					eventFunc: func() error {
						oldGm := c.GM() != 0
						newGm := newGmVal != 0
						return mb.Put(character2.EnvEventTopicCharacterStatus, gmChangedEventProvider(transactionId, characterId, c.WorldId(), oldGm, newGm))
					},
				})
			}

			// If no updates are needed, return early
			if len(changes) == 0 {
				return nil
			}

			// Apply all updates
			updates := []EntityUpdateFunction{}
			for _, change := range changes {
				if change.shouldApply {
					updates = append(updates, change.updateFunc)
				}
			}

			err = dynamicUpdate(tx)(updates...)(c)
			if err != nil {
				return err
			}

			// Emit events for all changes
			for _, change := range changes {
				if change.shouldApply {
					if err := change.eventFunc(); err != nil {
						return err
					}
				}
			}

			return nil
		})
	}
}

// Validation helper methods
func (p *ProcessorImpl) isValidHair(hair uint32) bool {
	// Basic hair ID validation - typical range for hair IDs
	return hair >= 30000 && hair <= 35000
}

func (p *ProcessorImpl) isValidFace(face uint32) bool {
	// Basic face ID validation - typical range for face IDs
	return face >= 20000 && face <= 25000
}

func (p *ProcessorImpl) isValidGender(gender byte) bool {
	// Gender must be 0 (male) or 1 (female)
	return gender == 0 || gender == 1
}

func (p *ProcessorImpl) isValidSkinColor(skinColor byte) bool {
	// Skin color typically ranges from 0-9
	return skinColor >= 0 && skinColor <= 9
}

func (p *ProcessorImpl) isValidGm(gm int) bool {
	// GM level must be non-negative. 0 = not GM, 1+ = GM level
	return gm >= 0
}

func (p *ProcessorImpl) ResetStatsAndEmit(transactionId uuid.UUID, characterId uint32, channel channel.Model) error {
	return database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
		return message.Emit(outbox.EmitProvider(p.l, p.ctx, tx))(func(buf *message.Buffer) error {
			return p.WithTransaction(tx).ResetStats(buf)(transactionId, characterId, channel)
		})
	})
}

func (p *ProcessorImpl) ResetStats(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, channel channel.Model) error {
	return func(transactionId uuid.UUID, characterId uint32, channel channel.Model) error {
		const baseStat uint16 = 4

		txErr := database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
			c, err := p.WithTransaction(tx).GetById()(characterId)
			if err != nil {
				return err
			}

			// Calculate AP to return from stats above base value
			returnedAP := uint16(0)
			if c.Strength() > baseStat {
				returnedAP += c.Strength() - baseStat
			}
			if c.Dexterity() > baseStat {
				returnedAP += c.Dexterity() - baseStat
			}
			if c.Intelligence() > baseStat {
				returnedAP += c.Intelligence() - baseStat
			}
			if c.Luck() > baseStat {
				returnedAP += c.Luck() - baseStat
			}

			p.l.Debugf("Resetting stats for character [%d]. Returning [%d] AP. STR: %d->%d, DEX: %d->%d, INT: %d->%d, LUK: %d->%d",
				characterId, returnedAP,
				c.Strength(), baseStat,
				c.Dexterity(), baseStat,
				c.Intelligence(), baseStat,
				c.Luck(), baseStat)

			return dynamicUpdate(tx)(
				SetStrength(baseStat),
				SetDexterity(baseStat),
				SetIntelligence(baseStat),
				SetLuck(baseStat),
				SetAP(c.AP()+returnedAP),
			)(c)
		})
		if txErr != nil {
			return txErr
		}

		values := map[string]interface{}{
			"strength":     baseStat,
			"dexterity":    baseStat,
			"intelligence": baseStat,
			"luck":         baseStat,
		}

		_ = mb.Put(character2.EnvEventTopicCharacterStatus, statChangedProvider(transactionId, channel, characterId, []stat.Type{stat.TypeAvailableAP, stat.TypeStrength, stat.TypeDexterity, stat.TypeIntelligence, stat.TypeLuck}, values))
		return nil
	}
}

func (p *ProcessorImpl) RebalanceAPAndEmit(transactionId uuid.UUID, characterId uint32, channel channel.Model, targets []RebalanceTarget) error {
	return database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
		return message.Emit(outbox.EmitProvider(p.l, p.ctx, tx))(func(buf *message.Buffer) error {
			return p.WithTransaction(tx).RebalanceAP(buf)(transactionId, characterId, channel, targets)
		})
	})
}

func (p *ProcessorImpl) RebalanceAP(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, channel channel.Model, targets []RebalanceTarget) error {
	return func(transactionId uuid.UUID, characterId uint32, channel channel.Model, targets []RebalanceTarget) error {
		var beforeStr, beforeDex, beforeInt, beforeLuk, beforeAP uint16
		var result rebalanceResult

		txErr := database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
			c, err := p.WithTransaction(tx).GetById()(characterId)
			if err != nil {
				return err
			}
			beforeStr, beforeDex, beforeInt, beforeLuk, beforeAP = c.Strength(), c.Dexterity(), c.Intelligence(), c.Luck(), c.AP()
			result, err = computeRebalance(beforeStr, beforeDex, beforeInt, beforeLuk, beforeAP, targets)
			if err != nil {
				return err
			}
			return dynamicUpdate(tx)(
				SetStrength(result.Str),
				SetDexterity(result.Dex),
				SetIntelligence(result.Int),
				SetLuck(result.Luk),
				SetAP(result.Unallocated),
			)(c)
		})
		if txErr != nil {
			p.l.WithError(txErr).Errorf("Could not rebalance AP for character [%d].", characterId)
			return txErr
		}

		p.l.Infof("Rebalanced character [%d] AP. Before STR=%d DEX=%d INT=%d LUK=%d AP=%d -> after STR=%d DEX=%d INT=%d LUK=%d AP=%d, targets=%+v",
			characterId,
			beforeStr, beforeDex, beforeInt, beforeLuk, beforeAP,
			result.Str, result.Dex, result.Int, result.Luk, result.Unallocated,
			targets)

		values := map[string]interface{}{
			"strength":     result.Str,
			"dexterity":    result.Dex,
			"intelligence": result.Int,
			"luck":         result.Luk,
		}
		_ = mb.Put(character2.EnvEventTopicCharacterStatus, statChangedProvider(transactionId, channel, characterId, []stat.Type{stat.TypeAvailableAP, stat.TypeStrength, stat.TypeDexterity, stat.TypeIntelligence, stat.TypeLuck}, values))
		return nil
	}
}

func (p *ProcessorImpl) TransferAPAndEmit(transactionId uuid.UUID, characterId uint32, channel channel.Model, from string, to string) error {
	return message.Emit(producer.ProviderImpl(p.l)(p.ctx))(func(buf *message.Buffer) error {
		return p.TransferAP(buf)(transactionId, characterId, channel, from, to)
	})
}

// transferApRejection is a non-nil sentinel carrying the machine-readable
// error code + detail for a rejected transfer; it is emitted as an ERROR
// status event, not returned as a Go error.
type transferApRejection struct {
	code   string
	detail string
}

// TransferAP moves one already-spent AP From -> To (AP Reset item 5050000),
// validating both ends of the swap against the point-reset policy tables
// (point_reset.go). The source decrement is applied to a set of running
// values first, then the target is validated/applied against that
// post-source state — this handles From==To naturally (no leaked source
// decrement on a target-side rejection) and keeps the whole operation a
// single dynamicUpdate. Any rejection returns nil (not a Go error) with a
// typed ERROR status event buffered instead of a STAT_CHANGED.
func (p *ProcessorImpl) TransferAP(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, channel channel.Model, from string, to string) error {
	return func(transactionId uuid.UUID, characterId uint32, channel channel.Model, from string, to string) error {
		var rejection *transferApRejection
		var stats []stat.Type
		values := map[string]interface{}{}

		txErr := database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
			c, err := p.WithTransaction(tx).GetById()(characterId)
			if err != nil {
				return err
			}
			// DIVERGENT (task-187 audit): resolve c.JobId() once for every
			// point-reset policy lookup below -- see point_reset.go's
			// pointResetPolicyRows/pointResetMinHpRows/pointResetMinMpRows
			// DIVERGENT notes (wire 500/510/520 Pirate/Brawler/Gunslinger at
			// v0.61+ collide with wire 500/510 GM/SuperGM at v0.48).
			jid, _ := p.set().Job.Resolve(c.JobId())
			policy := pointResetPolicyFor(jid)

			// Running values: source applied first, then target validated
			// against the post-source state (handles From==To naturally).
			newStr, newDex, newInt, newLuk := c.Strength(), c.Dexterity(), c.Intelligence(), c.Luck()
			newMaxHp, newMaxMp := c.MaxHp(), c.MaxMp()
			newHp, newMp := c.Hp(), c.Mp()
			newHpMpUsed := c.HpMpUsed()

			primary := func(ability string) *uint16 {
				switch ability {
				case CommandDistributeApAbilityStrength:
					return &newStr
				case CommandDistributeApAbilityDexterity:
					return &newDex
				case CommandDistributeApAbilityIntelligence:
					return &newInt
				case CommandDistributeApAbilityLuck:
					return &newLuk
				}
				return nil
			}

			// Source arm.
			switch from {
			case CommandDistributeApAbilityStrength, CommandDistributeApAbilityDexterity,
				CommandDistributeApAbilityIntelligence, CommandDistributeApAbilityLuck:
				src := primary(from)
				if *src < pointResetPrimaryFloor+1 {
					rejection = &transferApRejection{code: character2.StatusEventErrorTypeStatAtMinimum, detail: from}
					return nil
				}
				*src = *src - 1
			case CommandDistributeApAbilityHp:
				if newHpMpUsed < 1 {
					rejection = &transferApRejection{code: character2.StatusEventErrorTypeInsufficientHpMpApUsed, detail: from}
					return nil
				}
				if int(newMaxHp)-int(policy.takeHp) < pointResetMinHp(jid, c.Level()) {
					rejection = &transferApRejection{code: character2.StatusEventErrorTypePoolBelowJobMinimum, detail: from}
					return nil
				}
				newMaxHp -= policy.takeHp
				if int(newHp)-int(policy.takeHp) < 1 {
					newHp = 1
				} else {
					newHp -= policy.takeHp
				}
				newHpMpUsed--
			case CommandDistributeApAbilityMp:
				if newHpMpUsed < 1 {
					rejection = &transferApRejection{code: character2.StatusEventErrorTypeInsufficientHpMpApUsed, detail: from}
					return nil
				}
				// Magicians lose an INT-scaled amount of MaxMP (client parity);
				// every other branch loses the fixed policy.takeMp. Only
				// magicians need the effective-INT fetch, so it is scoped to
				// this branch (mirrors the effective-stats fetch in ChangeHP/
				// ChangeMP, which likewise run inside the transaction).
				takeMp := policy.takeMp
				if isPointResetMagician(jid) {
					// Effective INT (base + equipment) drives the loss; on an
					// effective-stats failure fall back to base INT, and log —
					// a silent degradation would reintroduce the very desync
					// this scaling fixes.
					effectiveInt := c.Intelligence()
					es, esErr := effective_stats.RequestByCharacter(channel, c.Id())(p.l, p.ctx)
					if esErr != nil {
						p.l.WithError(esErr).Warnf("Failed to fetch effective stats for character [%d]; magician MP-reset loss falls back to base INT [%d].", c.Id(), effectiveInt)
					} else if es.Intelligence > 0 {
						if es.Intelligence > math.MaxUint16 {
							effectiveInt = math.MaxUint16
						} else {
							effectiveInt = uint16(es.Intelligence)
						}
					}
					takeMp = pointResetMagicianTakeMp(effectiveInt)
				}
				if int(newMaxMp)-int(takeMp) < pointResetMinMp(jid, c.Level()) {
					rejection = &transferApRejection{code: character2.StatusEventErrorTypePoolBelowJobMinimum, detail: from}
					return nil
				}
				// MaxMp guard: pointResetMinMp can be negative at very low
				// levels (pirate/bowman/thief lines have negative offsets), so
				// the floor check above can pass with takeMp > newMaxMp. Clamp
				// to 0 to keep the uint16 subtraction from underflowing.
				if int(newMaxMp)-int(takeMp) < 0 {
					newMaxMp = 0
				} else {
					newMaxMp -= takeMp
				}
				if int(newMp)-int(takeMp) < 0 {
					newMp = 0
				} else {
					newMp -= takeMp
				}
				newHpMpUsed--
			default:
				rejection = &transferApRejection{code: character2.StatusEventErrorTypeApTransferInvalidTarget, detail: from}
				return nil
			}

			// Target arm (validated against post-source running values).
			switch to {
			case CommandDistributeApAbilityStrength, CommandDistributeApAbilityDexterity,
				CommandDistributeApAbilityIntelligence, CommandDistributeApAbilityLuck:
				dst := primary(to)
				if *dst+1 > pointResetPrimaryCap {
					rejection = &transferApRejection{code: character2.StatusEventErrorTypeStatAtMaximum, detail: to}
					return nil
				}
				*dst = *dst + 1
			case CommandDistributeApAbilityHp:
				if newMaxHp >= pointResetPoolCap {
					rejection = &transferApRejection{code: character2.StatusEventErrorTypeStatAtMaximum, detail: to}
					return nil
				}
				newMaxHp += policy.gainHp
				if newMaxHp > pointResetPoolCap {
					newMaxHp = pointResetPoolCap
				}
				newHpMpUsed++
			case CommandDistributeApAbilityMp:
				if newMaxMp >= pointResetPoolCap {
					rejection = &transferApRejection{code: character2.StatusEventErrorTypeStatAtMaximum, detail: to}
					return nil
				}
				newMaxMp += policy.gainMp
				if newMaxMp > pointResetPoolCap {
					newMaxMp = pointResetPoolCap
				}
				newHpMpUsed++
			default:
				rejection = &transferApRejection{code: character2.StatusEventErrorTypeApTransferInvalidTarget, detail: to}
				return nil
			}

			// Apply everything in one dynamicUpdate. remainingAp untouched (FR-11).
			mods := make([]EntityUpdateFunction, 0, 8)
			if newStr != c.Strength() {
				mods = append(mods, SetStrength(newStr))
				stats = append(stats, stat.TypeStrength)
				values["strength"] = newStr
			}
			if newDex != c.Dexterity() {
				mods = append(mods, SetDexterity(newDex))
				stats = append(stats, stat.TypeDexterity)
				values["dexterity"] = newDex
			}
			if newInt != c.Intelligence() {
				mods = append(mods, SetIntelligence(newInt))
				stats = append(stats, stat.TypeIntelligence)
				values["intelligence"] = newInt
			}
			if newLuk != c.Luck() {
				mods = append(mods, SetLuck(newLuk))
				stats = append(stats, stat.TypeLuck)
				values["luck"] = newLuk
			}
			if newMaxHp != c.MaxHp() {
				mods = append(mods, SetMaxHp(newMaxHp))
				stats = append(stats, stat.TypeMaxHp)
				values["max_hp"] = newMaxHp
			}
			if newMaxMp != c.MaxMp() {
				mods = append(mods, SetMaxMp(newMaxMp))
				stats = append(stats, stat.TypeMaxMp)
				values["max_mp"] = newMaxMp
			}
			if newHp != c.Hp() {
				mods = append(mods, SetHealth(newHp))
				stats = append(stats, stat.TypeHp)
				values["hp"] = newHp
			}
			if newMp != c.Mp() {
				mods = append(mods, SetMana(newMp))
				stats = append(stats, stat.TypeMp)
				values["mp"] = newMp
			}
			if newHpMpUsed != c.HpMpUsed() {
				mods = append(mods, SetHpMpUsed(newHpMpUsed))
			}
			if len(mods) == 0 {
				// From==To primary transfer nets to zero (e.g. STR->STR): no
				// columns to persist, but the caller still gets a successful
				// (empty) STAT_CHANGED below so the saga step completes and
				// the client unlocks.
				return nil
			}
			return dynamicUpdate(tx)(mods...)(c)
		})
		if txErr != nil {
			p.l.WithError(txErr).Errorf("Could not transfer AP for character [%d].", characterId)
			return txErr
		}

		if rejection != nil {
			p.l.WithFields(logrus.Fields{"character_id": characterId, "from": from, "to": to}).
				Warnf("Rejected AP transfer: [%s] detail [%s].", rejection.code, rejection.detail)
			_ = mb.Put(character2.EnvEventTopicCharacterStatus, apTransferErrorStatusEventProvider(transactionId, characterId, channel.WorldId(), rejection.code, rejection.detail))
			return nil
		}

		_ = mb.Put(character2.EnvEventTopicCharacterStatus, statChangedProvider(transactionId, channel, characterId, stats, values))
		return nil
	}
}
