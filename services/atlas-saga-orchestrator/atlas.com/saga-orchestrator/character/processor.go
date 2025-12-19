package character

import (
	"atlas-saga-orchestrator/data/portal"
	"atlas-saga-orchestrator/kafka/message"
	character2 "atlas-saga-orchestrator/kafka/message/character"
	"atlas-saga-orchestrator/kafka/producer"
	"context"
	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/field"
	"github.com/Chronicle20/atlas-constants/job"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-model/model"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

type Processor interface {
	WarpRandomAndEmit(transactionId uuid.UUID, characterId uint32, field field.Model) error
	WarpRandom(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, field field.Model) error
	WarpToPortalAndEmit(transactionId uuid.UUID, characterId uint32, field field.Model, pp model.Provider[uint32]) error
	WarpToPortal(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, field field.Model, pp model.Provider[uint32]) error
	AwardExperienceAndEmit(transactionId uuid.UUID, worldId world.Id, characterId uint32, channelId channel.Id, distributions []character2.ExperienceDistributions) error
	AwardExperience(mb *message.Buffer) func(transactionId uuid.UUID, worldId world.Id, characterId uint32, channelId channel.Id, distributions []character2.ExperienceDistributions) error
	AwardLevelAndEmit(transactionId uuid.UUID, worldId world.Id, characterId uint32, channelId channel.Id, amount byte) error
	AwardLevel(mb *message.Buffer) func(transactionId uuid.UUID, worldId world.Id, characterId uint32, channelId channel.Id, amount byte) error
	AwardMesosAndEmit(transactionId uuid.UUID, worldId world.Id, characterId uint32, channelId channel.Id, actorId uint32, actorType string, amount int32) error
	AwardMesos(mb *message.Buffer) func(transactionId uuid.UUID, worldId world.Id, characterId uint32, channelId channel.Id, actorId uint32, actorType string, amount int32) error
	ChangeJobAndEmit(transactionId uuid.UUID, worldId world.Id, characterId uint32, channelId channel.Id, jobId job.Id) error
	ChangeJob(mb *message.Buffer) func(transactionId uuid.UUID, worldId world.Id, characterId uint32, channelId channel.Id, jobId job.Id) error
	RequestCreateCharacter(transactionId uuid.UUID, accountId uint32, worldId byte, name string, level byte, strength uint16, dexterity uint16, intelligence uint16, luck uint16, hp uint16, mp uint16, jobId job.Id, gender byte, face uint32, hair uint32, skin byte, mapId _map.Id) error
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	t   tenant.Model
	p   producer.Provider
	pp  portal.Processor
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
		t:   tenant.MustFromContext(ctx),
		p:   producer.ProviderImpl(l)(ctx),
		pp:  portal.NewProcessor(l, ctx),
	}
}

func (p *ProcessorImpl) WarpRandomAndEmit(transactionId uuid.UUID, characterId uint32, field field.Model) error {
	return message.Emit(p.p)(func(mb *message.Buffer) error {
		return p.WarpRandom(mb)(transactionId, characterId, field)
	})
}

func (p *ProcessorImpl) WarpRandom(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, field field.Model) error {
	return func(transactionId uuid.UUID, characterId uint32, field field.Model) error {
		return p.WarpToPortal(mb)(transactionId, characterId, field, p.pp.RandomSpawnPointIdProvider(field.MapId()))
	}
}

func (p *ProcessorImpl) WarpToPortalAndEmit(transactionId uuid.UUID, characterId uint32, field field.Model, pp model.Provider[uint32]) error {
	return message.Emit(p.p)(func(mb *message.Buffer) error {
		return p.WarpToPortal(mb)(transactionId, characterId, field, pp)
	})
}

func (p *ProcessorImpl) WarpToPortal(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, field field.Model, pp model.Provider[uint32]) error {
	return func(transactionId uuid.UUID, characterId uint32, field field.Model, pp model.Provider[uint32]) error {
		portalId, err := pp()
		if err != nil {
			return err
		}
		return mb.Put(character2.EnvCommandTopic, ChangeMapProvider(transactionId, characterId, field, portalId))
	}
}

func (p *ProcessorImpl) AwardExperienceAndEmit(transactionId uuid.UUID, worldId world.Id, characterId uint32, channelId channel.Id, distributions []character2.ExperienceDistributions) error {
	return message.Emit(p.p)(func(mb *message.Buffer) error {
		return p.AwardExperience(mb)(transactionId, worldId, characterId, channelId, distributions)
	})
}

func (p *ProcessorImpl) AwardExperience(mb *message.Buffer) func(transactionId uuid.UUID, worldId world.Id, characterId uint32, channelId channel.Id, distributions []character2.ExperienceDistributions) error {
	return func(transactionId uuid.UUID, worldId world.Id, characterId uint32, channelId channel.Id, distributions []character2.ExperienceDistributions) error {
		return mb.Put(character2.EnvCommandTopic, AwardExperienceProvider(transactionId, worldId, characterId, channelId, distributions))
	}
}

func (p *ProcessorImpl) AwardLevelAndEmit(transactionId uuid.UUID, worldId world.Id, characterId uint32, channelId channel.Id, amount byte) error {
	return message.Emit(p.p)(func(mb *message.Buffer) error {
		return p.AwardLevel(mb)(transactionId, worldId, characterId, channelId, amount)
	})
}

func (p *ProcessorImpl) AwardLevel(mb *message.Buffer) func(transactionId uuid.UUID, worldId world.Id, characterId uint32, channelId channel.Id, amount byte) error {
	return func(transactionId uuid.UUID, worldId world.Id, characterId uint32, channelId channel.Id, amount byte) error {
		return mb.Put(character2.EnvCommandTopic, AwardLevelProvider(transactionId, worldId, characterId, channelId, amount))
	}
}

func (p *ProcessorImpl) AwardMesosAndEmit(transactionId uuid.UUID, worldId world.Id, characterId uint32, channelId channel.Id, actorId uint32, actorType string, amount int32) error {
	return message.Emit(p.p)(func(mb *message.Buffer) error {
		return p.AwardMesos(mb)(transactionId, worldId, characterId, channelId, actorId, actorType, amount)
	})
}

func (p *ProcessorImpl) AwardMesos(mb *message.Buffer) func(transactionId uuid.UUID, worldId world.Id, characterId uint32, channelId channel.Id, actorId uint32, actorType string, amount int32) error {
	return func(transactionId uuid.UUID, worldId world.Id, characterId uint32, channelId channel.Id, actorId uint32, actorType string, amount int32) error {
		return mb.Put(character2.EnvCommandTopic, AwardMesosProvider(transactionId, worldId, characterId, channelId, actorId, actorType, amount))
	}
}

func (p *ProcessorImpl) ChangeJobAndEmit(transactionId uuid.UUID, worldId world.Id, characterId uint32, channelId channel.Id, jobId job.Id) error {
	return message.Emit(p.p)(func(mb *message.Buffer) error {
		return p.ChangeJob(mb)(transactionId, worldId, characterId, channelId, jobId)
	})
}

func (p *ProcessorImpl) ChangeJob(mb *message.Buffer) func(transactionId uuid.UUID, worldId world.Id, characterId uint32, channelId channel.Id, jobId job.Id) error {
	return func(transactionId uuid.UUID, worldId world.Id, characterId uint32, channelId channel.Id, jobId job.Id) error {
		return mb.Put(character2.EnvCommandTopic, ChangeJobProvider(transactionId, worldId, characterId, channelId, jobId))
	}
}

func (p *ProcessorImpl) RequestCreateCharacter(transactionId uuid.UUID, accountId uint32, worldId byte, name string, level byte, strength uint16, dexterity uint16, intelligence uint16, luck uint16, hp uint16, mp uint16, jobId job.Id, gender byte, face uint32, hair uint32, skin byte, mapId _map.Id) error {
	return message.Emit(p.p)(func(mb *message.Buffer) error {
		return mb.Put(character2.EnvCommandTopic, RequestCreateCharacterProvider(transactionId, accountId, worldId, name, level, strength, dexterity, intelligence, luck, hp, mp, jobId, gender, face, hair, skin, mapId))
	})
}
