package character

import (
	"atlas-character/data/portal"
	skill3 "atlas-character/data/skill"
	"atlas-character/database"
	"atlas-character/drop"
	"atlas-character/kafka/message"
	character2 "atlas-character/kafka/message/character"
	"atlas-character/kafka/producer"
	skill2 "atlas-character/skill"
	"context"
	"errors"
	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/field"
	"github.com/Chronicle20/atlas-constants/job"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/skill"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"math"
	"math/rand"
	"regexp"
	"time"
)

var blockedNameErr = errors.New("blocked name")
var invalidLevelErr = errors.New("invalid level")

const (
	CommandDistributeApAbilityStrength     = "STRENGTH"
	CommandDistributeApAbilityDexterity    = "DEXTERITY"
	CommandDistributeApAbilityIntelligence = "INTELLIGENCE"
	CommandDistributeApAbilityLuck         = "LUCK"
	CommandDistributeApAbilityHp           = "HP"
	CommandDistributeApAbilityMp           = "MP"
)

type Processor interface {
	WithTransaction(tx *gorm.DB) Processor
	ByIdProvider(decorators ...model.Decorator[Model]) func(id uint32) model.Provider[Model]
	GetById(decorators ...model.Decorator[Model]) func(id uint32) (Model, error)
	GetForAccountInWorld(decorators ...model.Decorator[Model]) func(accountId uint32, worldId world.Id) ([]Model, error)
	GetForMapInWorld(decorators ...model.Decorator[Model]) func(worldId world.Id, mapId _map.Id) ([]Model, error)
	GetForName(decorators ...model.Decorator[Model]) func(name string) ([]Model, error)
	GetAll(decorators ...model.Decorator[Model]) ([]Model, error)
	SkillModelDecorator(m Model) Model
	IsValidName(name string) (bool, error)
	CreateAndEmit(transactionId uuid.UUID, input Model) (Model, error)
	Create(mb *message.Buffer) func(transactionId uuid.UUID, input Model) (Model, error)
	DeleteAndEmit(transactionId uuid.UUID, characterId uint32) error
	Delete(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32) error
	LoginAndEmit(transactionId uuid.UUID, characterId uint32, channel channel.Model) error
	Login(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, channel channel.Model) error
	LogoutAndEmit(transactionId uuid.UUID, characterId uint32, channel channel.Model) error
	Logout(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, channel channel.Model) error
	ChangeChannelAndEmit(transactionId uuid.UUID, characterId uint32, currentChannel channel.Model, oldChannelId channel.Id) error
	ChangeChannel(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, currentChannel channel.Model, oldChannelId channel.Id) error
	ChangeMapAndEmit(transactionId uuid.UUID, characterId uint32, field field.Model, portalId uint32) error
	ChangeMap(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, field field.Model, portalId uint32) error
	ChangeJobAndEmit(transactionId uuid.UUID, characterId uint32, channel channel.Model, jobId job.Id) error
	ChangeJob(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, channel channel.Model, jobId job.Id) error
	ChangeHairAndEmit(transactionId uuid.UUID, characterId uint32, channel channel.Model, styleId uint32) error
	ChangeHair(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, channel channel.Model, styleId uint32) error
	ChangeFaceAndEmit(transactionId uuid.UUID, characterId uint32, channel channel.Model, styleId uint32) error
	ChangeFace(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, channel channel.Model, styleId uint32) error
	ChangeSkinAndEmit(transactionId uuid.UUID, characterId uint32, channel channel.Model, styleId byte) error
	ChangeSkin(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, channel channel.Model, styleId byte) error
	AwardExperienceAndEmit(transactionId uuid.UUID, characterId uint32, channel channel.Model, experience []ExperienceModel) error
	AwardExperience(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, channel channel.Model, experience []ExperienceModel) error
	AwardLevelAndEmit(transactionId uuid.UUID, characterId uint32, channel channel.Model, level byte) error
	AwardLevel(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, channel channel.Model, level byte) error
	Move(characterId uint32, x int16, y int16, stance byte) error
	RequestChangeMeso(transactionId uuid.UUID, characterId uint32, amount int32, actorId uint32, actorType string) error
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
	ProcessLevelChangeAndEmit(transactionId uuid.UUID, channel channel.Model, characterId uint32, amount byte) error
	ProcessLevelChange(mb *message.Buffer) func(transactionId uuid.UUID, channel channel.Model, characterId uint32, amount byte) error
	ProcessJobChangeAndEmit(transactionId uuid.UUID, channel channel.Model, characterId uint32, jobId job.Id) error
	ProcessJobChange(mb *message.Buffer) func(transactionId uuid.UUID, channel channel.Model, characterId uint32, jobId job.Id) error
	UpdateAndEmit(transactionId uuid.UUID, characterId uint32, input RestModel) error
	Update(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, input RestModel) error
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

func (p *ProcessorImpl) WithTransaction(tx *gorm.DB) Processor {
	return &ProcessorImpl{
		l:   p.l,
		ctx: p.ctx,
		db:  tx,
		t:   p.t,
		pp:  p.pp,
		sp:  p.sp,
	}
}

func (p *ProcessorImpl) ByIdProvider(decorators ...model.Decorator[Model]) func(id uint32) model.Provider[Model] {
	return func(id uint32) model.Provider[Model] {
		mp := model.Map(modelFromEntity)(getById(p.t.Id(), id)(p.db))
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
		mp := model.SliceMap(modelFromEntity)(getForAccountInWorld(p.t.Id(), accountId, worldId)(p.db))(model.ParallelMap())
		return model.SliceMap(model.Decorate(decorators))(mp)(model.ParallelMap())()
	}
}

func (p *ProcessorImpl) GetForMapInWorld(decorators ...model.Decorator[Model]) func(worldId world.Id, mapId _map.Id) ([]Model, error) {
	return func(worldId world.Id, mapId _map.Id) ([]Model, error) {
		mp := model.SliceMap(modelFromEntity)(getForMapInWorld(p.t.Id(), worldId, mapId)(p.db))(model.ParallelMap())
		return model.SliceMap(model.Decorate[Model](decorators))(mp)(model.ParallelMap())()
	}
}

func (p *ProcessorImpl) GetForName(decorators ...model.Decorator[Model]) func(name string) ([]Model, error) {
	return func(name string) ([]Model, error) {
		mp := model.SliceMap[entity, Model](modelFromEntity)(getForName(p.t.Id(), name)(p.db))(model.ParallelMap())
		return model.SliceMap(model.Decorate[Model](decorators))(mp)(model.ParallelMap())()
	}
}

func (p *ProcessorImpl) GetAll(decorators ...model.Decorator[Model]) ([]Model, error) {
	mp := model.SliceMap(modelFromEntity)(getAll(p.t.Id())(p.db))(model.ParallelMap())
	return model.SliceMap(model.Decorate[Model](decorators))(mp)(model.ParallelMap())()
}

func (p *ProcessorImpl) SkillModelDecorator(m Model) Model {
	ms, err := p.sp.GetByCharacterId(m.Id())
	if err != nil {
		return m
	}
	return CloneModel(m).SetSkills(ms).Build()
}

func (p *ProcessorImpl) IsValidName(name string) (bool, error) {
	m, err := regexp.MatchString("[A-Za-z0-9\u3040-\u309F\u30A0-\u30FF\u4E00-\u9FAF]{3,12}", name)
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

func (p *ProcessorImpl) CreateAndEmit(transactionId uuid.UUID, input Model) (Model, error) {
	var output Model
	err := message.Emit(producer.ProviderImpl(p.l)(p.ctx))(func(buf *message.Buffer) error {
		var err error
		output, err = p.Create(buf)(transactionId, input)
		if err != nil {
			// Emit creation failed event on error
			_ = buf.Put(character2.EnvEventTopicCharacterStatus, creationFailedEventProvider(transactionId, input.WorldId(), input.Name(), err.Error()))
		}
		return err
	})
	return output, err
}

func (p *ProcessorImpl) Create(mb *message.Buffer) func(transactionId uuid.UUID, input Model) (Model, error) {
	return func(transactionId uuid.UUID, input Model) (Model, error) {
		ok, err := p.IsValidName(input.Name())
		if err != nil {
			p.l.WithError(err).Errorf("Error validating name [%s] during character creation.", input.Name())
			return Model{}, err
		}
		if !ok {
			p.l.Infof("Attempting to create a character with an invalid name [%s].", input.Name())
			return Model{}, blockedNameErr
		}
		if input.Level() < 1 || input.Level() > 200 {
			p.l.Infof("Attempting to create character with an invalid level [%d].", input.Level())
			return Model{}, invalidLevelErr
		}

		var res Model
		txErr := database.ExecuteTransaction(p.db, func(tx *gorm.DB) error {
			res, err = create(tx, p.t.Id(), input.accountId, input.worldId, input.name, input.level, input.strength, input.dexterity, input.intelligence, input.luck, input.maxHp, input.maxMp, input.jobId, input.gender, input.hair, input.face, input.skinColor, input.mapId, input.gm)
			if err != nil {
				p.l.WithError(err).Errorf("Error persisting character in database.")
				tx.Rollback()
				return err
			}
			return mb.Put(character2.EnvEventTopicCharacterStatus, createdEventProvider(transactionId, res.Id(), res.WorldId(), res.Name()))
		})
		if txErr != nil {
			p.l.WithError(txErr).Errorf("Error persisting character in database.")
			return Model{}, txErr
		}
		return res, nil
	}
}

func (p *ProcessorImpl) DeleteAndEmit(transactionId uuid.UUID, characterId uint32) error {
	return message.Emit(producer.ProviderImpl(p.l)(p.ctx))(func(buf *message.Buffer) error {
		return p.Delete(buf)(transactionId, characterId)
	})
}

func (p *ProcessorImpl) Delete(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32) error {
	return func(transactionId uuid.UUID, characterId uint32) error {
		txErr := database.ExecuteTransaction(p.db, func(tx *gorm.DB) error {
			c, err := p.GetById()(characterId)
			if err != nil {
				return err
			}

			err = delete(tx, p.t.Id(), characterId)
			if err != nil {
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

func (p *ProcessorImpl) LoginAndEmit(transactionId uuid.UUID, characterId uint32, channel channel.Model) error {
	return message.Emit(producer.ProviderImpl(p.l)(p.ctx))(func(buf *message.Buffer) error {
		return p.Login(buf)(transactionId, characterId, channel)
	})
}

func (p *ProcessorImpl) Login(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, channel channel.Model) error {
	return func(transactionId uuid.UUID, characterId uint32, channel channel.Model) error {
		return model.For(p.ByIdProvider()(characterId), func(c Model) error {
			return mb.Put(character2.EnvEventTopicCharacterStatus, loginEventProvider(transactionId, c.Id(), field.NewBuilder(channel.WorldId(), channel.Id(), c.MapId()).Build()))
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
			return mb.Put(character2.EnvEventTopicCharacterStatus, logoutEventProvider(transactionId, c.Id(), field.NewBuilder(channel.WorldId(), channel.Id(), c.MapId()).Build()))
		})
	}
}

func (p *ProcessorImpl) ChangeChannelAndEmit(transactionId uuid.UUID, characterId uint32, currentChannel channel.Model, oldChannelId channel.Id) error {
	return message.Emit(producer.ProviderImpl(p.l)(p.ctx))(func(buf *message.Buffer) error {
		return p.ChangeChannel(buf)(transactionId, characterId, currentChannel, oldChannelId)
	})
}

func (p *ProcessorImpl) ChangeChannel(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, currentChannel channel.Model, oldChannelId channel.Id) error {
	return func(transactionId uuid.UUID, characterId uint32, currentChannel channel.Model, oldChannelId channel.Id) error {
		return model.For(p.ByIdProvider()(characterId), func(c Model) error {
			oldField := field.NewBuilder(c.WorldId(), oldChannelId, c.MapId()).Build()
			newField := field.NewBuilder(currentChannel.WorldId(), currentChannel.Id(), c.MapId()).Build()
			return mb.Put(character2.EnvEventTopicCharacterStatus, changeChannelEventProvider(transactionId, c.Id(), oldField, newField))
		})
	}
}

func (p *ProcessorImpl) ChangeMapAndEmit(transactionId uuid.UUID, characterId uint32, field field.Model, portalId uint32) error {
	return message.Emit(producer.ProviderImpl(p.l)(p.ctx))(func(buf *message.Buffer) error {
		return p.ChangeMap(buf)(transactionId, characterId, field, portalId)
	})
}

func (p *ProcessorImpl) ChangeMap(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, field field.Model, portalId uint32) error {
	return func(transactionId uuid.UUID, characterId uint32, field field.Model, portalId uint32) error {
		cmf := dynamicUpdate(p.db)(SetMapId(field.MapId()))(p.t.Id())
		papf := p.positionAtPortal(field.MapId(), portalId)
		amcf := announceMapChangedWithBuffer(mb)(transactionId, field, portalId)
		return model.For(p.ByIdProvider()(characterId), model.ThenOperator(cmf, model.Operators(papf, amcf)))
	}
}

func (p *ProcessorImpl) positionAtPortal(mapId _map.Id, portalId uint32) model.Operator[Model] {
	return func(c Model) error {
		por, err := p.pp.GetInMapById(mapId, portalId)
		if err != nil {
			return err
		}
		GetTemporalRegistry().UpdatePosition(c.Id(), por.X(), por.Y())
		return nil
	}
}

func announceMapChangedWithBuffer(mb *message.Buffer) func(transactionId uuid.UUID, newField field.Model, portalId uint32) model.Operator[Model] {
	return func(transactionId uuid.UUID, newField field.Model, portalId uint32) model.Operator[Model] {
		return func(c Model) error {
			oldField := field.NewBuilder(newField.WorldId(), newField.ChannelId(), c.MapId()).Build()
			return mb.Put(character2.EnvEventTopicCharacterStatus, mapChangedEventProvider(transactionId, c.Id(), oldField, newField, portalId))
		}
	}
}

func announceMapChanged(provider producer.Provider) func(transactionId uuid.UUID, newField field.Model, portalId uint32) model.Operator[Model] {
	return func(transactionId uuid.UUID, newField field.Model, portalId uint32) model.Operator[Model] {
		return func(c Model) error {
			oldField := field.NewBuilder(newField.WorldId(), newField.ChannelId(), c.MapId()).Build()
			return provider(character2.EnvEventTopicCharacterStatus)(mapChangedEventProvider(transactionId, c.Id(), oldField, newField, portalId))
		}
	}
}

func (p *ProcessorImpl) ChangeJobAndEmit(transactionId uuid.UUID, characterId uint32, channel channel.Model, jobId job.Id) error {
	return message.Emit(producer.ProviderImpl(p.l)(p.ctx))(func(buf *message.Buffer) error {
		return p.ChangeJob(buf)(transactionId, characterId, channel, jobId)
	})
}

func (p *ProcessorImpl) ChangeJob(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, channel channel.Model, jobId job.Id) error {
	return func(transactionId uuid.UUID, characterId uint32, channel channel.Model, jobId job.Id) error {
		p.l.Debugf("Attempting to set character [%d] job to [%d].", characterId, jobId)
		txErr := database.ExecuteTransaction(p.db, func(tx *gorm.DB) error {
			c, err := p.WithTransaction(tx).GetById()(characterId)
			if err != nil {
				return err
			}
			err = dynamicUpdate(tx)(SetJob(jobId))(p.t.Id())(c)
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
		_ = mb.Put(character2.EnvEventTopicCharacterStatus, statChangedProvider(transactionId, channel, characterId, []string{"JOB"}))
		return nil
	}
}

func (p *ProcessorImpl) ChangeHairAndEmit(transactionId uuid.UUID, characterId uint32, channel channel.Model, styleId uint32) error {
	return message.Emit(producer.ProviderImpl(p.l)(p.ctx))(func(buf *message.Buffer) error {
		return p.ChangeHair(buf)(transactionId, characterId, channel, styleId)
	})
}

func (p *ProcessorImpl) ChangeHair(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, channel channel.Model, styleId uint32) error {
	return func(transactionId uuid.UUID, characterId uint32, channel channel.Model, styleId uint32) error {
		p.l.Debugf("Attempting to set character [%d] hair to [%d].", characterId, styleId)
		var oldHair uint32
		txErr := database.ExecuteTransaction(p.db, func(tx *gorm.DB) error {
			c, err := p.WithTransaction(tx).GetById()(characterId)
			if err != nil {
				return err
			}
			oldHair = c.Hair()
			err = dynamicUpdate(tx)(SetHair(styleId))(p.t.Id())(c)
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
		_ = mb.Put(character2.EnvEventTopicCharacterStatus, statChangedProvider(transactionId, channel, characterId, []string{"HAIR"}))
		return nil
	}
}

func (p *ProcessorImpl) ChangeFaceAndEmit(transactionId uuid.UUID, characterId uint32, channel channel.Model, styleId uint32) error {
	return message.Emit(producer.ProviderImpl(p.l)(p.ctx))(func(buf *message.Buffer) error {
		return p.ChangeFace(buf)(transactionId, characterId, channel, styleId)
	})
}

func (p *ProcessorImpl) ChangeFace(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, channel channel.Model, styleId uint32) error {
	return func(transactionId uuid.UUID, characterId uint32, channel channel.Model, styleId uint32) error {
		p.l.Debugf("Attempting to set character [%d] face to [%d].", characterId, styleId)
		var oldFace uint32
		txErr := database.ExecuteTransaction(p.db, func(tx *gorm.DB) error {
			c, err := p.WithTransaction(tx).GetById()(characterId)
			if err != nil {
				return err
			}
			oldFace = c.Face()
			err = dynamicUpdate(tx)(SetFace(styleId))(p.t.Id())(c)
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
		_ = mb.Put(character2.EnvEventTopicCharacterStatus, statChangedProvider(transactionId, channel, characterId, []string{"FACE"}))
		return nil
	}
}

func (p *ProcessorImpl) ChangeSkinAndEmit(transactionId uuid.UUID, characterId uint32, channel channel.Model, styleId byte) error {
	return message.Emit(producer.ProviderImpl(p.l)(p.ctx))(func(buf *message.Buffer) error {
		return p.ChangeSkin(buf)(transactionId, characterId, channel, styleId)
	})
}

func (p *ProcessorImpl) ChangeSkin(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, channel channel.Model, styleId byte) error {
	return func(transactionId uuid.UUID, characterId uint32, channel channel.Model, styleId byte) error {
		p.l.Debugf("Attempting to set character [%d] skin to [%d].", characterId, styleId)
		var oldSkin byte
		txErr := database.ExecuteTransaction(p.db, func(tx *gorm.DB) error {
			c, err := p.WithTransaction(tx).GetById()(characterId)
			if err != nil {
				return err
			}
			oldSkin = c.SkinColor()
			err = dynamicUpdate(tx)(SetSkinColor(styleId))(p.t.Id())(c)
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
		_ = mb.Put(character2.EnvEventTopicCharacterStatus, statChangedProvider(transactionId, channel, characterId, []string{"SKIN_COLOR"}))
		return nil
	}
}

type ExperienceModel struct{
	experienceType string
	amount         uint32
	attr1          uint32
}

func NewExperienceModel(experienceType string, amount uint32, attr1 uint32) ExperienceModel {
	return ExperienceModel{experienceType, amount, attr1}
}

func (p *ProcessorImpl) AwardExperienceAndEmit(transactionId uuid.UUID, characterId uint32, channel channel.Model, experience []ExperienceModel) error {
	return message.Emit(producer.ProviderImpl(p.l)(p.ctx))(func(buf *message.Buffer) error {
		return p.AwardExperience(buf)(transactionId, characterId, channel, experience)
	})
}

func (p *ProcessorImpl) AwardExperience(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, channel channel.Model, experience []ExperienceModel) error {
	return func(transactionId uuid.UUID, characterId uint32, channel channel.Model, experience []ExperienceModel) error {
		amount := uint32(0)
		for _, e := range experience {
			amount += e.amount
		}

		p.l.Debugf("Attempting to award character [%d] [%d] experience.", characterId, amount)
		awardedLevels := byte(0)
		current := uint32(0)
		txErr := database.ExecuteTransaction(p.db, func(tx *gorm.DB) error {
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

			err = dynamicUpdate(tx)(SetExperience(current))(p.t.Id())(c)
			if err != nil {
				return err
			}
			return nil
		})
		if txErr != nil {
			p.l.WithError(txErr).Errorf("Could not award character [%d] [%d] experience.", characterId, amount)
			return txErr
		}

		_ = mb.Put(character2.EnvEventTopicCharacterStatus, experienceChangedEventProvider(transactionId, characterId, channel, experience, current))
		_ = mb.Put(character2.EnvEventTopicCharacterStatus, statChangedProvider(transactionId, channel, characterId, []string{"EXPERIENCE"}))
		if awardedLevels > 0 {
			_ = mb.Put(character2.EnvCommandTopic, awardLevelCommandProvider(transactionId, characterId, channel, awardedLevels))
		}
		return nil
	}
}

func (p *ProcessorImpl) AwardLevelAndEmit(transactionId uuid.UUID, characterId uint32, channel channel.Model, amount byte) error {
	return message.Emit(producer.ProviderImpl(p.l)(p.ctx))(func(buf *message.Buffer) error {
		return p.AwardLevel(buf)(transactionId, characterId, channel, amount)
	})
}

func (p *ProcessorImpl) AwardLevel(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, channel channel.Model, amount byte) error {
	return func(transactionId uuid.UUID, characterId uint32, channel channel.Model, amount byte) error {
		p.l.Debugf("Attempting to award character [%d] [%d] level(s).", characterId, amount)
		actual := amount
		current := byte(0)
		txErr := database.ExecuteTransaction(p.db, func(tx *gorm.DB) error {
			c, err := p.WithTransaction(tx).GetById()(characterId)
			if err != nil {
				return err
			}

			if c.Level()+amount > MaxLevel {
				p.l.Debugf("Awarding [%d] level(s) would cause character [%d] to go over cap [%d]. Setting change to [%d].", amount, characterId, MaxLevel, actual)
				actual = MaxLevel - c.Level()
			}
			current = c.Level() + actual

			err = dynamicUpdate(tx)(SetLevel(current))(p.t.Id())(c)
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
		_ = mb.Put(character2.EnvEventTopicCharacterStatus, statChangedProvider(transactionId, channel, characterId, []string{"LEVEL"}))
		return nil
	}
}

func (p *ProcessorImpl) Move(characterId uint32, x int16, y int16, stance byte) error {
	GetTemporalRegistry().Update(characterId, x, y, stance)
	return nil
}

func (p *ProcessorImpl) RequestChangeMeso(transactionId uuid.UUID, characterId uint32, amount int32, actorId uint32, actorType string) error {
	return database.ExecuteTransaction(p.db, func(tx *gorm.DB) error {
		c, err := p.WithTransaction(tx).GetById()(characterId)
		if err != nil {
			p.l.WithError(err).Errorf("Unable to retrieve character [%d] who is having their meso adjusted.", characterId)
			return err
		}
		if int64(c.Meso())+int64(amount) < 0 {
			p.l.Debugf("Request for character [%d] would leave their meso negative. Amount [%d]. Existing [%d].", characterId, amount, c.Meso())
			return producer.ProviderImpl(p.l)(p.ctx)(character2.EnvEventTopicCharacterStatus)(notEnoughMesoErrorStatusEventProvider(transactionId, characterId, c.WorldId(), amount))
		}
		if amount > 0 && uint32(amount) > (math.MaxUint32-c.Meso()) {
			p.l.Errorf("Transaction for character [%d] would result in a uint32 overflow. Rejecting transaction.", characterId)
			return err
		}

		err = dynamicUpdate(tx)(SetMeso(uint32(int64(c.Meso()) + int64(amount))))(p.t.Id())(c)
		_ = producer.ProviderImpl(p.l)(p.ctx)(character2.EnvEventTopicCharacterStatus)(mesoChangedStatusEventProvider(transactionId, characterId, c.WorldId(), amount, actorId, actorType))
		return producer.ProviderImpl(p.l)(p.ctx)(character2.EnvEventTopicCharacterStatus)(statChangedProvider(transactionId, channel.NewModel(c.WorldId(), 0), characterId, []string{"MESO"}))
	})
}

func (p *ProcessorImpl) AttemptMesoPickUp(transactionId uuid.UUID, field field.Model, characterId uint32, dropId uint32, meso uint32) error {
	txErr := database.ExecuteTransaction(p.db, func(tx *gorm.DB) error {
		c, err := p.WithTransaction(tx).GetById()(characterId)
		if err != nil {
			p.l.WithError(err).Errorf("Unable to retrieve character [%d] who is having their meso adjusted.", characterId)
			return err
		}
		if meso > (math.MaxUint32 - c.Meso()) {
			p.l.Errorf("Transaction for character [%d] would result in a uint32 overflow. Rejecting transaction.", characterId)
			return err
		}

		err = dynamicUpdate(tx)(SetMeso(uint32(int64(c.Meso()) + int64(meso))))(p.t.Id())(c)
		return producer.ProviderImpl(p.l)(p.ctx)(character2.EnvEventTopicCharacterStatus)(statChangedProvider(transactionId, channel.NewModel(field.WorldId(), field.ChannelId()), characterId, []string{"MESO"}))
	})
	if txErr != nil {
		return txErr
	}
	return drop.NewProcessor(p.l, p.ctx).RequestPickUp(field, dropId, characterId)
}

func (p *ProcessorImpl) RequestDropMeso(transactionId uuid.UUID, field field.Model, characterId uint32, amount uint32) error {
	txErr := database.ExecuteTransaction(p.db, func(tx *gorm.DB) error {
		c, err := p.WithTransaction(tx).GetById()(characterId)
		if err != nil {
			p.l.WithError(err).Errorf("Unable to retrieve character [%d] who is having their meso adjusted.", characterId)
			return err
		}
		if int64(c.Meso())-int64(amount) < 0 {
			p.l.Debugf("Request for character [%d] would leave their meso negative. Amount [%d]. Existing [%d].", characterId, amount, c.Meso())
			return producer.ProviderImpl(p.l)(p.ctx)(character2.EnvEventTopicCharacterStatus)(notEnoughMesoErrorStatusEventProvider(transactionId, characterId, c.WorldId(), int32(amount)))
		}

		return dynamicUpdate(tx)(SetMeso(c.Meso() - amount))(p.t.Id())(c)
	})
	if txErr != nil {
		return txErr
	}

	tc := GetTemporalRegistry().GetById(characterId)

	_ = producer.ProviderImpl(p.l)(p.ctx)(character2.EnvEventTopicCharacterStatus)(statChangedProvider(transactionId, channel.NewModel(field.WorldId(), field.ChannelId()), characterId, []string{"MESO"}))
	// TODO determine appropriate drop type and mod
	_ = drop.NewProcessor(p.l, p.ctx).CreateForMesos(field, amount, 2, tc.X(), tc.Y(), characterId)
	return nil
}

func (p *ProcessorImpl) RequestChangeFame(transactionId uuid.UUID, characterId uint32, amount int8, actorId uint32, actorType string) error {
	return database.ExecuteTransaction(p.db, func(tx *gorm.DB) error {
		c, err := p.WithTransaction(tx).GetById()(characterId)
		if err != nil {
			p.l.WithError(err).Errorf("Unable to retrieve character [%d] who is having their fame adjusted.", characterId)
			return err
		}

		total := c.Fame() + int16(amount)
		err = dynamicUpdate(tx)(SetFame(total))(p.t.Id())(c)
		_ = producer.ProviderImpl(p.l)(p.ctx)(character2.EnvEventTopicCharacterStatus)(fameChangedStatusEventProvider(transactionId, characterId, c.WorldId(), amount, actorId, actorType))
		return producer.ProviderImpl(p.l)(p.ctx)(character2.EnvEventTopicCharacterStatus)(statChangedProvider(transactionId, channel.NewModel(c.WorldId(), 0), characterId, []string{"FAME"}))
	})
}

type Distribution struct {
	Ability string
	Amount  int8
}

func (p *ProcessorImpl) RequestDistributeAp(transactionId uuid.UUID, characterId uint32, distributions []Distribution) error {
	return database.ExecuteTransaction(p.db, func(tx *gorm.DB) error {
		c, err := p.WithTransaction(tx).GetById()(characterId)
		if err != nil {
			_ = producer.ProviderImpl(p.l)(p.ctx)(character2.EnvEventTopicCharacterStatus)(statChangedProvider(transactionId, channel.NewModel(c.WorldId(), 0), characterId, []string{}))
			return err
		}
		if c.AP() < uint16(len(distributions)) {
			_ = producer.ProviderImpl(p.l)(p.ctx)(character2.EnvEventTopicCharacterStatus)(statChangedProvider(transactionId, channel.NewModel(c.WorldId(), 0), characterId, []string{}))
			return errors.New("not enough ap")
		}

		var eufs = make([]EntityUpdateFunction, 0)
		var stat = make([]string, 0)

		spent := uint16(0)
		for _, d := range distributions {
			switch d.Ability {
			case CommandDistributeApAbilityStrength:
				eufs = append(eufs, SetStrength(uint16(int16(c.Strength())+int16(d.Amount))))
				stat = append(stat, "STRENGTH")
				break
			case CommandDistributeApAbilityDexterity:
				eufs = append(eufs, SetDexterity(uint16(int16(c.Dexterity())+int16(d.Amount))))
				stat = append(stat, "DEXTERITY")
				break
			case CommandDistributeApAbilityIntelligence:
				eufs = append(eufs, SetIntelligence(uint16(int16(c.Intelligence())+int16(d.Amount))))
				stat = append(stat, "INTELLIGENCE")
				break
			case CommandDistributeApAbilityLuck:
				eufs = append(eufs, SetLuck(uint16(int16(c.Luck())+int16(d.Amount))))
				stat = append(stat, "LUCK")
				break
			case CommandDistributeApAbilityHp:
				hpGrowth, err := p.getMaxHpGrowth(c)
				if err != nil {
					return err
				}
				eufs = append(eufs, SetMaxHP(uint16(int16(hpGrowth)*int16(d.Amount))))
				eufs = append(eufs, SetHPMPUsed(c.HPMPUsed()+int(d.Amount)))
				stat = append(stat, "MAX_HP")
				break
			case CommandDistributeApAbilityMp:
				mpGrowth, err := p.getMaxMpGrowth(c)
				if err != nil {
					return err
				}
				eufs = append(eufs, SetMaxMP(uint16(int16(mpGrowth)*int16(d.Amount))))
				eufs = append(eufs, SetHPMPUsed(c.HPMPUsed()+int(d.Amount)))
				stat = append(stat, "MAX_MP")
				break
			}
			spent = uint16(int16(spent) + int16(d.Amount))
		}

		if len(eufs) == 0 {
			_ = producer.ProviderImpl(p.l)(p.ctx)(character2.EnvEventTopicCharacterStatus)(statChangedProvider(transactionId, channel.NewModel(c.WorldId(), 0), characterId, []string{}))
			return errors.New("invalid ability")
		}

		eufs = append(eufs, SetAP(c.AP()-spent))
		stat = append(stat, "AVAILABLE_AP")

		err = dynamicUpdate(tx)(eufs...)(p.t.Id())(c)
		if err != nil {
			_ = producer.ProviderImpl(p.l)(p.ctx)(character2.EnvEventTopicCharacterStatus)(statChangedProvider(transactionId, channel.NewModel(c.WorldId(), 0), characterId, []string{"AVAILABLE_AP"}))
			return err
		}

		_ = producer.ProviderImpl(p.l)(p.ctx)(character2.EnvEventTopicCharacterStatus)(statChangedProvider(transactionId, channel.NewModel(c.WorldId(), 0), characterId, stat))
		return nil
	})
}

func (p *ProcessorImpl) RequestDistributeSp(transactionId uuid.UUID, characterId uint32, skillId uint32, amount int8) error {
	var c Model
	txErr := database.ExecuteTransaction(p.db, func(tx *gorm.DB) error {
		var err error
		c, err = p.WithTransaction(tx).GetById(p.SkillModelDecorator)(characterId)
		if err != nil {
			return err
		}
		sjid, ok := job.FromSkillId(skill.Id(skillId))
		if !ok {
			return errors.New("unable to locate job from skill")
		}
		sb := getSkillBook(sjid.Id())
		if c.SP(sb) < uint32(amount) {
			return errors.New("not enough sp")
		}
		return dynamicUpdate(tx)(SetSP(c.SP(sb)-uint32(amount), uint32(sb)))(p.t.Id())(c)
	})
	if txErr != nil {
		return txErr
	}
	_ = producer.ProviderImpl(p.l)(p.ctx)(character2.EnvEventTopicCharacterStatus)(statChangedProvider(transactionId, channel.NewModel(c.WorldId(), 0), characterId, []string{"AVAILABLE_SP"}))

	if val := c.GetSkill(skillId); val.Id() != skillId {
		_ = skill2.NewProcessor(p.l, p.ctx).RequestCreate(characterId, skillId, byte(amount), 0, time.Time{})
	} else {
		_ = skill2.NewProcessor(p.l, p.ctx).RequestUpdate(characterId, skillId, val.Level()+byte(amount), val.MasterLevel(), val.Expiration())
	}
	return nil
}

func (p *ProcessorImpl) getMaxHpGrowth(c Model) (uint16, error) {
	if c.MaxHP() >= 30000 || c.HPMPUsed() > 9999 {
		return c.MaxHP(), errors.New("max ap to hp")
	}
	var improvingHPSkillId skill.Id
	resMax := c.MaxHP()
	if job.IsA(c.JobId(),
		job.WarriorId,
		job.FighterId, job.CrusaderId, job.HeroId,
		job.PageId, job.CrusaderId, job.WhiteKnightId,
		job.SpearmanId, job.DragonKnightId, job.DarkKnightId,
		job.DawnWarriorStage1Id, job.DawnWarriorStage2Id, job.DawnWarriorStage3Id, job.DawnWarriorStage4Id,
		job.AranStage1Id, job.AranStage2Id, job.AranStage3Id, job.AranStage4Id) {
		if job.IsCygnus(c.JobId()) {
			improvingHPSkillId = skill.DawnWarriorStage1ImprovedMaxHpIncreaseId
		} else {
			improvingHPSkillId = skill.WarriorImprovedMaxHpIncreaseId
		}
		resMax += 20
	} else if job.IsA(c.JobId(),
		job.MagicianId,
		job.FirePoisonWizardId, job.FirePoisonMagicianId, job.FirePoisonArchMagicianId,
		job.IceLightningWizardId, job.IceLightningMagicianId, job.IceLightningArchMagicianId,
		job.ClericId, job.PriestId, job.BishopId,
		job.BlazeWizardStage1Id, job.BlazeWizardStage2Id, job.BlazeWizardStage3Id, job.BlazeWizardStage4Id) {
		resMax += 6
	} else if job.IsA(c.JobId(),
		job.BowmanId,
		job.HunterId, job.RangerId, job.BowmasterId,
		job.CrossbowmanId, job.SniperId, job.MarksmanId,
		job.WindArcherStage1Id, job.WindArcherStage2Id, job.WindArcherStage3Id, job.WindArcherStage4Id,
		job.RogueId,
		job.AssassinId, job.HermitId, job.NightLordId,
		job.BanditId, job.ChiefBanditId, job.ShadowerId,
		job.NightWalkerStage1Id, job.NightWalkerStage2Id, job.NightWalkerStage3Id, job.NightWalkerStage4Id) {
		resMax += 16
	} else if job.IsA(c.JobId(),
		job.PirateId,
		job.BrawlerId, job.MarauderId, job.BuccaneerId,
		job.GunslingerId, job.OutlawId, job.CorsairId,
		job.ThunderBreakerStage1Id, job.ThunderBreakerStage2Id, job.ThunderBreakerStage3Id, job.ThunderBreakerStage4Id) {
		if job.IsCygnus(c.JobId()) {
			improvingHPSkillId = skill.ThunderBreakerStage2ImprovedMaxHpIncreaseId
		} else {
			improvingHPSkillId = skill.BrawlerImproveMaxHpId
		}
		resMax += 18
	} else {
		resMax += 8
	}

	if improvingHPSkillId > 0 {
		var improvingHPSkillLevel = c.GetSkillLevel(uint32(improvingHPSkillId))
		se, err := p.sdp.GetEffect(uint32(improvingHPSkillId), improvingHPSkillLevel)
		if err == nil {
			resMax = uint16(int16(resMax) + se.Y())
		}
	}
	return resMax, nil
}

func (p *ProcessorImpl) getMaxMpGrowth(c Model) (uint16, error) {
	if c.MaxMP() >= 30000 || c.HPMPUsed() > 9999 {
		return c.MaxMP(), errors.New("max ap to mp")
	}
	var improvingMPSkillId skill.Id
	resMax := c.MaxMP()
	if job.IsA(c.JobId(),
		job.WarriorId,
		job.FighterId, job.CrusaderId, job.HeroId,
		job.PageId, job.CrusaderId, job.WhiteKnightId,
		job.SpearmanId, job.DragonKnightId, job.DarkKnightId,
		job.DawnWarriorStage1Id, job.DawnWarriorStage2Id, job.DawnWarriorStage3Id, job.DawnWarriorStage4Id,
		job.AranStage1Id, job.AranStage2Id, job.AranStage3Id, job.AranStage4Id) {
		if job.IsA(c.JobId(), job.CrusaderId, job.WhiteKnightId) {
			improvingMPSkillId = skill.WhiteKnightImprovingMpRecoveryId
		} else if job.IsA(c.JobId(), job.DawnWarriorStage3Id, job.DawnWarriorStage4Id) {
			improvingMPSkillId = skill.DawnWarriorStage3ImprovedMpRecoveryId
		}
		resMax += 2
	} else if job.IsA(c.JobId(),
		job.MagicianId,
		job.FirePoisonWizardId, job.FirePoisonMagicianId, job.FirePoisonArchMagicianId,
		job.IceLightningWizardId, job.IceLightningMagicianId, job.IceLightningArchMagicianId,
		job.ClericId, job.PriestId, job.BishopId,
		job.BlazeWizardStage1Id, job.BlazeWizardStage2Id, job.BlazeWizardStage3Id, job.BlazeWizardStage4Id) {
		if job.IsCygnus(c.JobId()) {
			improvingMPSkillId = skill.BlazeWizardStage1ImprovedMaxMpIncreaseId
		} else {
			improvingMPSkillId = skill.MagicianImprovedMaxMpIncreaseId
		}
		resMax += 18
	} else if job.IsA(c.JobId(),
		job.BowmanId,
		job.HunterId, job.RangerId, job.BowmasterId,
		job.CrossbowmanId, job.SniperId, job.MarksmanId,
		job.WindArcherStage1Id, job.WindArcherStage2Id, job.WindArcherStage3Id, job.WindArcherStage4Id,
		job.RogueId,
		job.AssassinId, job.HermitId, job.NightLordId,
		job.BanditId, job.ChiefBanditId, job.ShadowerId,
		job.NightWalkerStage1Id, job.NightWalkerStage2Id, job.NightWalkerStage3Id, job.NightWalkerStage4Id) {
		resMax += 10
	} else if job.IsA(
		job.PirateId,
		job.BrawlerId, job.MarauderId, job.BuccaneerId,
		job.GunslingerId, job.OutlawId, job.CorsairId,
		job.ThunderBreakerStage1Id, job.ThunderBreakerStage2Id, job.ThunderBreakerStage3Id, job.ThunderBreakerStage4Id) {
		resMax += 14
	} else {
		resMax += 6
	}
	// TODO this needs to incorporate computed total intelligence (buffs, weapons, etc)
	resMax += uint16(math.Ceil(float64(c.Intelligence()) / 10))

	if improvingMPSkillId > 0 {
		var improvingMPSkillLevel = c.GetSkillLevel(uint32(improvingMPSkillId))
		se, err := p.sdp.GetEffect(uint32(improvingMPSkillId), improvingMPSkillLevel)
		if err == nil {
			resMax = uint16(int16(resMax) + se.X())
		}
	}

	return resMax, nil
}

func enforceBounds(change int16, current uint16, upperBound uint16, lowerBound uint16) uint16 {
	var adjusted = int16(current) + change
	return uint16(math.Min(math.Max(float64(adjusted), float64(lowerBound)), float64(upperBound)))
}

func (p *ProcessorImpl) ChangeHPAndEmit(transactionId uuid.UUID, channel channel.Model, characterId uint32, amount int16) error {
	return message.Emit(producer.ProviderImpl(p.l)(p.ctx))(func(buf *message.Buffer) error {
		return p.ChangeHP(buf)(transactionId, channel, characterId, amount)
	})
}

func (p *ProcessorImpl) ChangeHP(mb *message.Buffer) func(transactionId uuid.UUID, channel channel.Model, characterId uint32, amount int16) error {
	return func(transactionId uuid.UUID, channel channel.Model, characterId uint32, amount int16) error {
		txErr := database.ExecuteTransaction(p.db, func(tx *gorm.DB) error {
			c, err := p.WithTransaction(tx).GetById()(characterId)
			if err != nil {
				return err
			}
			// TODO consider effective (temporary) Max HP.
			adjusted := enforceBounds(amount, c.HP(), c.MaxHP(), 0)
			p.l.Debugf("Attempting to adjust character [%d] health by [%d] to [%d].", characterId, amount, adjusted)
			return dynamicUpdate(tx)(SetHealth(adjusted))(p.t.Id())(c)
		})
		if txErr != nil {
			return txErr
		}
		// TODO need to emit event when character dies.
		_ = mb.Put(character2.EnvEventTopicCharacterStatus, statChangedProvider(transactionId, channel, characterId, []string{"HP"}))
		return nil
	}
}

func (p *ProcessorImpl) SetHPAndEmit(transactionId uuid.UUID, channel channel.Model, characterId uint32, amount uint16) error {
	return message.Emit(producer.ProviderImpl(p.l)(p.ctx))(func(buf *message.Buffer) error {
		return p.SetHP(buf)(transactionId, channel, characterId, amount)
	})
}

func (p *ProcessorImpl) SetHP(mb *message.Buffer) func(transactionId uuid.UUID, channel channel.Model, characterId uint32, amount uint16) error {
	return func(transactionId uuid.UUID, channel channel.Model, characterId uint32, amount uint16) error {
		txErr := database.ExecuteTransaction(p.db, func(tx *gorm.DB) error {
			c, err := p.WithTransaction(tx).GetById()(characterId)
			if err != nil {
				return err
			}
			// Clamp amount between 0 and MaxHP
			clamped := amount
			if clamped > c.MaxHP() {
				clamped = c.MaxHP()
			}
			p.l.Debugf("Setting character [%d] health to [%d].", characterId, clamped)
			return dynamicUpdate(tx)(SetHealth(clamped))(p.t.Id())(c)
		})
		if txErr != nil {
			return txErr
		}
		_ = mb.Put(character2.EnvEventTopicCharacterStatus, statChangedProvider(transactionId, channel, characterId, []string{"HP"}))
		return nil
	}
}

func (p *ProcessorImpl) ChangeMPAndEmit(transactionId uuid.UUID, channel channel.Model, characterId uint32, amount int16) error {
	return message.Emit(producer.ProviderImpl(p.l)(p.ctx))(func(buf *message.Buffer) error {
		return p.ChangeMP(buf)(transactionId, channel, characterId, amount)
	})
}

func (p *ProcessorImpl) ChangeMP(mb *message.Buffer) func(transactionId uuid.UUID, channel channel.Model, characterId uint32, amount int16) error {
	return func(transactionId uuid.UUID, channel channel.Model, characterId uint32, amount int16) error {
		txErr := database.ExecuteTransaction(p.db, func(tx *gorm.DB) error {
			c, err := p.WithTransaction(tx).GetById()(characterId)
			if err != nil {
				return err
			}
			// TODO consider effective (temporary) Max MP.
			adjusted := enforceBounds(amount, c.MP(), c.MaxMP(), 0)
			p.l.Debugf("Attempting to adjust character [%d] mana by [%d] to [%d].", characterId, amount, adjusted)
			return dynamicUpdate(tx)(SetMana(adjusted))(p.t.Id())(c)
		})
		if txErr != nil {
			return txErr
		}
		_ = mb.Put(character2.EnvEventTopicCharacterStatus, statChangedProvider(transactionId, channel, characterId, []string{"MP"}))
		return nil
	}
}

func (p *ProcessorImpl) ProcessLevelChangeAndEmit(transactionId uuid.UUID, channel channel.Model, characterId uint32, amount byte) error {
	return message.Emit(producer.ProviderImpl(p.l)(p.ctx))(func(buf *message.Buffer) error {
		return p.ProcessLevelChange(buf)(transactionId, channel, characterId, amount)
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
		var sus = []string{"AVAILABLE_AP", "AVAILABLE_SP", "HP", "MAX_HP", "MP", "MAX_MP"}

		txErr := database.ExecuteTransaction(p.db, func(tx *gorm.DB) error {
			c, err := p.WithTransaction(tx).GetById(p.SkillModelDecorator)(characterId)
			if err != nil {
				return err
			}

			effectiveLevel := c.Level() - amount

			for i := range amount {
				effectiveLevel = effectiveLevel + i + 1

				if p.t.Region() == "GMS" && p.t.MajorVersion() == 83 {
					// TODO properly define this range. For these versions, Beginner, Noblesse, and Legend AP are auto assigned.
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

				addedSP += computeOnLevelAddedSP(c.JobId())
				// TODO could potentially pre-compute HP and MP so you don't incur loop cost
				aHP, aMP := p.computeOnLevelAddedHPandMP(c)
				addedHP += aHP
				addedMP += aMP
			}

			p.l.Debugf("As a result of processing a level change of [%d]. Character [%d] will gain [%d] AP, [%d] SP, [%d] HP, and [%d] MP.", amount, characterId, addedAP, addedSP, addedHP, addedMP)
			sb := getSkillBook(c.JobId())

			var eufs = []EntityUpdateFunction{
				SetAP(c.AP() + addedAP),
				SetSP(c.SP(sb)+addedSP, uint32(sb)),
				SetHealth(c.MaxHP() + addedHP),
				SetMaxHP(c.MaxHP() + addedHP),
				SetMana(c.MaxMP() + addedMP),
				SetMaxMP(c.MaxMP() + addedMP),
			}

			if addedStr > 0 {
				eufs = append(eufs, SetStrength(c.Strength()+addedStr))
				sus = append(sus, "STRENGTH")
			}
			if addedDex > 0 {
				eufs = append(eufs, SetDexterity(c.Dexterity()+addedDex))
				sus = append(sus, "DEXTERITY")
			}

			return dynamicUpdate(tx)(eufs...)(p.t.Id())(c)
		})
		if txErr != nil {
			return txErr
		}
		_ = mb.Put(character2.EnvEventTopicCharacterStatus, statChangedProvider(transactionId, channel, characterId, sus))
		return nil
	}
}

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

func computeOnLevelAddedSP(jobId job.Id) uint32 {
	// TODO need to account for 6 beginner skill levels
	if job.IsBeginner(jobId) {
		return 0
	}
	return 3
}

func (p *ProcessorImpl) computeOnLevelAddedHPandMP(c Model) (uint16, uint16) {
	var addedHP uint16
	var addedMP uint16
	var improvingHPSkillId skill.Id
	var improvingMPSkillId skill.Id

	randBoundFunc := func(lower uint16, upper uint16) uint16 {
		return uint16(rand.Float32()*float32(upper-lower+1)) + lower
	}

	if job.IsBeginner(c.JobId()) {
		addedHP = randBoundFunc(12, 16)
		addedMP = randBoundFunc(10, 12)
	} else if job.IsA(c.JobId(),
		job.WarriorId,
		job.FighterId, job.CrusaderId, job.HeroId,
		job.PageId, job.CrusaderId, job.WhiteKnightId,
		job.SpearmanId, job.DragonKnightId, job.DarkKnightId,
		job.DawnWarriorStage1Id, job.DawnWarriorStage2Id, job.DawnWarriorStage3Id, job.DawnWarriorStage4Id) {
		if job.IsCygnus(c.JobId()) {
			improvingHPSkillId = skill.DawnWarriorStage1ImprovedMaxHpIncreaseId
		} else {
			improvingHPSkillId = skill.WarriorImprovedMaxHpIncreaseId
		}
		if job.IsA(c.JobId(), job.CrusaderId, job.WhiteKnightId) {
			improvingMPSkillId = skill.WhiteKnightImprovingMpRecoveryId
		} else if job.IsA(c.JobId(), job.DawnWarriorStage3Id, job.DawnWarriorStage4Id) {
			improvingMPSkillId = skill.DawnWarriorStage3ImprovedMpRecoveryId
		}
		addedHP = randBoundFunc(24, 28)
		addedMP = randBoundFunc(4, 6)
	} else if job.IsA(c.JobId(),
		job.MagicianId,
		job.FirePoisonWizardId, job.FirePoisonMagicianId, job.FirePoisonArchMagicianId,
		job.IceLightningWizardId, job.IceLightningMagicianId, job.IceLightningArchMagicianId,
		job.ClericId, job.PriestId, job.BishopId,
		job.BlazeWizardStage1Id, job.BlazeWizardStage2Id, job.BlazeWizardStage3Id, job.BlazeWizardStage4Id) {
		if job.IsCygnus(c.JobId()) {
			improvingMPSkillId = skill.BlazeWizardStage1ImprovedMaxMpIncreaseId
		} else {
			improvingMPSkillId = skill.MagicianImprovedMaxMpIncreaseId
		}
		addedHP = randBoundFunc(10, 14)
		addedMP = randBoundFunc(22, 24)
	} else if job.IsA(c.JobId(),
		job.BowmanId,
		job.HunterId, job.RangerId, job.BowmasterId,
		job.CrossbowmanId, job.SniperId, job.MarksmanId,
		job.WindArcherStage1Id, job.WindArcherStage2Id, job.WindArcherStage3Id, job.WindArcherStage4Id,
		job.RogueId,
		job.AssassinId, job.HermitId, job.NightLordId,
		job.BanditId, job.ChiefBanditId, job.ShadowerId,
		job.NightWalkerStage1Id, job.NightWalkerStage2Id, job.NightWalkerStage3Id, job.NightWalkerStage4Id) {
		addedHP = randBoundFunc(20, 24)
		addedMP = randBoundFunc(14, 16)
	} else if job.IsA(c.JobId(), job.GmId, job.SuperGmId) {
		addedHP = 30000
		addedMP = 30000
	} else if job.IsA(
		job.PirateId,
		job.BrawlerId, job.MarauderId, job.BuccaneerId,
		job.GunslingerId, job.OutlawId, job.CorsairId,
		job.ThunderBreakerStage1Id, job.ThunderBreakerStage2Id, job.ThunderBreakerStage3Id, job.ThunderBreakerStage4Id) {
		if job.IsCygnus(c.JobId()) {
			improvingHPSkillId = skill.ThunderBreakerStage2ImprovedMaxHpIncreaseId
		} else {
			improvingHPSkillId = skill.BrawlerImproveMaxHpId
		}
		addedHP = randBoundFunc(22, 28)
		addedMP = randBoundFunc(18, 23)
	} else if job.IsA(c.JobId(), job.AranStage1Id, job.AranStage2Id, job.AranStage3Id, job.AranStage4Id) {
		addedHP = randBoundFunc(44, 48)
		addedMP = randBoundFunc(4, 8)
	}

	if improvingHPSkillId > 0 {
		var improvingHPSkillLevel = c.GetSkillLevel(uint32(improvingHPSkillId))
		se, err := p.sdp.GetEffect(uint32(improvingHPSkillId), improvingHPSkillLevel)
		if err == nil {
			addedHP = uint16(int16(addedHP) + se.X())
		}
	}
	if improvingMPSkillId > 0 {
		var improvingMPSkillLevel = c.GetSkillLevel(uint32(improvingMPSkillId))
		se, err := p.sdp.GetEffect(uint32(improvingMPSkillId), improvingMPSkillLevel)
		if err == nil {
			addedMP = uint16(int16(addedMP) + se.X())
		}
	}
	return addedHP, addedMP
}

func (p *ProcessorImpl) ProcessJobChangeAndEmit(transactionId uuid.UUID, channel channel.Model, characterId uint32, jobId job.Id) error {
	return message.Emit(producer.ProviderImpl(p.l)(p.ctx))(func(buf *message.Buffer) error {
		return p.ProcessJobChange(buf)(transactionId, channel, characterId, jobId)
	})
}

func (p *ProcessorImpl) ProcessJobChange(mb *message.Buffer) func(transactionId uuid.UUID, channel channel.Model, characterId uint32, jobId job.Id) error {
	return func(transactionId uuid.UUID, channel channel.Model, characterId uint32, jobId job.Id) error {
		var addedAP uint16
		var addedSP uint32
		var addedHP uint16
		var addedMP uint16

		randBoundFunc := func(lower uint16, upper uint16) uint16 {
			return uint16(rand.Float32()*float32(upper-lower+1)) + lower
		}

		txErr := database.ExecuteTransaction(p.db, func(tx *gorm.DB) error {
			c, err := p.WithTransaction(tx).GetById(p.SkillModelDecorator)(characterId)
			if err != nil {
				return err
			}

			// TODO award job change AP is this only Cygnus?
			if job.IsCygnus(jobId) {
				addedAP = 7
			}

			addedSP = 1
			if job.IsA(jobId, job.EvanId, job.EvanStage1Id, job.EvanStage2Id, job.EvanStage3Id, job.EvanStage4Id, job.EvanStage5Id, job.EvanStage6Id, job.EvanStage7Id, job.EvanStage8Id, job.EvanStage9Id, job.EvanStage10Id) {
				addedAP += 2
			} else if job.IsFourthJob(jobId) {
				addedSP += 2
			}

			if job.IsA(jobId, job.WarriorId, job.DawnWarriorStage1Id, job.AranStage1Id) {
				addedHP = randBoundFunc(200, 250)
			} else if job.IsA(jobId, job.MagicianId, job.BlazeWizardStage1Id, job.EvanStage1Id) {
				addedMP = randBoundFunc(100, 150)
			} else if job.IsA(jobId, job.BowmanId, job.RogueId, job.PirateId, job.WindArcherStage1Id, job.NightWalkerStage1Id, job.ThunderBreakerStage1Id) {
				addedHP = randBoundFunc(100, 150)
				addedMP = randBoundFunc(25, 50)
			} else if job.IsA(jobId,
				job.FighterId, job.CrusaderId, job.HeroId,
				job.PageId, job.CrusaderId, job.WhiteKnightId,
				job.SpearmanId, job.DragonKnightId, job.DarkKnightId,
				job.DawnWarriorStage2Id, job.DawnWarriorStage3Id, job.DawnWarriorStage4Id,
				job.AranStage2Id, job.AranStage3Id, job.AranStage4Id) {
				addedHP = randBoundFunc(300, 350)
			} else if job.IsA(jobId,
				job.FirePoisonWizardId, job.FirePoisonMagicianId, job.FirePoisonArchMagicianId,
				job.IceLightningWizardId, job.IceLightningMagicianId, job.IceLightningArchMagicianId,
				job.ClericId, job.PriestId, job.BishopId,
				job.BlazeWizardStage2Id, job.BlazeWizardStage3Id, job.BlazeWizardStage4Id,
				job.EvanStage2Id, job.EvanStage3Id, job.EvanStage4Id, job.EvanStage5Id, job.EvanStage6Id, job.EvanStage7Id, job.EvanStage8Id, job.EvanStage9Id, job.EvanStage10Id) {
				addedMP = randBoundFunc(450, 500)
			} else if !job.IsBeginner(jobId) {
				addedHP = randBoundFunc(300, 350)
				addedMP = randBoundFunc(150, 200)
			}

			p.l.Debugf("As a result of processing a job change to [%d]. Character [%d] will gain [%d] AP, [%d] SP, [%d] HP, and [%d] MP.", jobId, characterId, addedAP, addedSP, addedHP, addedMP)
			sb := getSkillBook(c.JobId())
			return dynamicUpdate(tx)(SetAP(c.AP()+addedAP), SetSP(c.SP(sb)+addedSP, uint32(sb)), SetHealth(c.MaxHP()+addedHP), SetMaxHP(c.MaxHP()+addedHP), SetMana(c.MaxMP()+addedMP), SetMaxMP(c.MaxMP()+addedMP))(p.t.Id())(c)
		})
		if txErr != nil {
			return txErr
		}
		_ = mb.Put(character2.EnvEventTopicCharacterStatus, statChangedProvider(transactionId, channel, characterId, []string{"AVAILABLE_AP", "AVAILABLE_SP", "HP", "MAX_HP", "MP", "MAX_MP"}))
		return nil
	}
}

func getSkillBook(jobId job.Id) int {
	if jobId >= job.EvanStage2Id && jobId <= job.EvanStage10Id {
		return int(jobId - 2209)
	}
	return 0
}

func (p *ProcessorImpl) UpdateAndEmit(transactionId uuid.UUID, characterId uint32, input RestModel) error {
	return message.Emit(producer.ProviderImpl(p.l)(p.ctx))(func(buf *message.Buffer) error {
		return p.Update(buf)(transactionId, characterId, input)
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
		return database.ExecuteTransaction(p.db, func(tx *gorm.DB) error {
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
			// Only update GM if the input explicitly provides a different value
			// We skip the update if input.Gm is 0 and current GM is non-zero, as this likely means
			// the client didn't intend to change GM status (zero value in request)
			if input.Gm != c.GM() && !(input.Gm == 0 && c.GM() != 0) {
				if !p.isValidGm(input.Gm) {
					return errors.New("invalid GM value")
				}
				changes = append(changes, fieldChange{
					updateFunc:  SetGm(input.Gm),
					shouldApply: true,
					eventFunc: func() error {
						// Convert int to bool for GM status
						oldGm := c.GM() != 0
						newGm := input.Gm != 0
						return mb.Put(character2.EnvEventTopicCharacterStatus, gmChangedEventProvider(transactionId, characterId, c.WorldId(), oldGm, newGm))
					},
				})
			}

			// MapId update
			if input.MapId != 0 && input.MapId != c.MapId() {
				changes = append(changes, fieldChange{
					updateFunc:  SetMapId(input.MapId),
					shouldApply: true,
					eventFunc: func() error {
						// Create field models for old and new map locations
						// Note: We need to get the current channel ID from the context or use a default
						// For now, using channel ID 0 as a placeholder - this should be updated based on actual channel context
						oldField := field.NewBuilder(c.WorldId(), 0, c.MapId()).Build()
						newField := field.NewBuilder(c.WorldId(), 0, input.MapId).Build()
						return mb.Put(character2.EnvEventTopicCharacterStatus, mapChangedEventProvider(transactionId, characterId, oldField, newField, 0))
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

			err = dynamicUpdate(tx)(updates...)(p.t.Id())(c)
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

