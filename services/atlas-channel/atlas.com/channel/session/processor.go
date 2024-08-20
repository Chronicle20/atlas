package session

import (
	"atlas-channel/kafka/producer"
	"atlas-channel/socket/writer"
	"atlas-channel/tenant"
	"errors"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
)

func AllInTenantProvider(tenant tenant.Model) func() ([]Model, error) {
	return func() ([]Model, error) {
		return GetRegistry().GetInTenant(tenant.Id), nil
	}
}

func ByCharacterIdModelProvider(tenant tenant.Model) func(characterId uint32) model.Provider[Model] {
	return func(characterId uint32) model.Provider[Model] {
		return model.FirstProvider[Model](AllInTenantProvider(tenant), CharacterIdFilter(characterId))
	}
}

// IfPresentByCharacterId executes an Operator if a session exists for the characterId
func IfPresentByCharacterId(tenant tenant.Model, worldId byte, channelId byte) func(characterId uint32, f model.Operator[Model]) {
	return func(characterId uint32, f model.Operator[Model]) {
		s, err := ByCharacterIdModelProvider(tenant)(characterId)()
		if err != nil {
			return
		}
		if s.WorldId() != worldId || s.ChannelId() != channelId {
			return
		}
		_ = f(s)
	}
}

func CharacterIdFilter(referenceId uint32) model.Filter[Model] {
	return func(model Model) bool {
		return model.CharacterId() == referenceId
	}
}

// GetByCharacterId gets a session (if one exists) for the given characterId
func GetByCharacterId(tenant tenant.Model) func(characterId uint32) (Model, error) {
	return func(characterId uint32) (Model, error) {
		return ByCharacterIdModelProvider(tenant)(characterId)()
	}
}

func ForEachByCharacterId(tenant tenant.Model) func(provider model.Provider[[]uint32], f model.Operator[Model]) error {
	return func(provider model.Provider[[]uint32], f model.Operator[Model]) error {
		return model.ForEachSlice(model.SliceMap[uint32, Model](provider, GetByCharacterId(tenant)), f, model.ParallelExecute())
	}
}

func Announce(l logrus.FieldLogger) func(writerProducer writer.Producer) func(writerName string) func(s Model, bodyProducer writer.BodyProducer) error {
	return func(writerProducer writer.Producer) func(writerName string) func(s Model, bodyProducer writer.BodyProducer) error {
		return func(writerName string) func(s Model, bodyProducer writer.BodyProducer) error {
			return func(s Model, bodyProducer writer.BodyProducer) error {
				w, err := writerProducer(l, writerName)
				if err != nil {
					return err
				}

				if lock, ok := GetRegistry().GetLock(s.Tenant().Id, s.SessionId()); ok {
					lock.Lock()
					err = s.announceEncrypted(w(l)(bodyProducer))
					lock.Unlock()
					return err
				}
				return errors.New("invalid session")
			}
		}
	}
}

func SetAccountId(accountId uint32) func(tenantId uuid.UUID, id uuid.UUID) Model {
	return func(tenantId uuid.UUID, id uuid.UUID) Model {
		s := Model{}
		var ok bool
		if s, ok = GetRegistry().Get(tenantId, id); ok {
			s = s.setAccountId(accountId)
			GetRegistry().Update(s)
			return s
		}
		return s
	}
}

func SetCharacterId(characterId uint32) func(tenantId uuid.UUID, id uuid.UUID) Model {
	return func(tenantId uuid.UUID, id uuid.UUID) Model {
		s := Model{}
		var ok bool
		if s, ok = GetRegistry().Get(tenantId, id); ok {
			s = s.setCharacterId(characterId)
			GetRegistry().Update(s)
			return s
		}
		return s
	}
}

func SetMapId(mapId uint32) func(tenantId uuid.UUID, id uuid.UUID) Model {
	return func(tenantId uuid.UUID, id uuid.UUID) Model {
		s := Model{}
		var ok bool
		if s, ok = GetRegistry().Get(tenantId, id); ok {
			s = s.setMapId(mapId)
			GetRegistry().Update(s)
			return s
		}
		return s
	}
}

func SetGm(gm bool) func(tenantId uuid.UUID, id uuid.UUID) Model {
	return func(tenantId uuid.UUID, id uuid.UUID) Model {
		s := Model{}
		var ok bool
		if s, ok = GetRegistry().Get(tenantId, id); ok {
			s = s.setGm(gm)
			GetRegistry().Update(s)
			return s
		}
		return s
	}
}

func UpdateLastRequest() func(tenantId uuid.UUID, id uuid.UUID) Model {
	return func(tenantId uuid.UUID, id uuid.UUID) Model {
		s := Model{}
		var ok bool
		if s, ok = GetRegistry().Get(tenantId, id); ok {
			s = s.updateLastRequest()
			GetRegistry().Update(s)
			return s
		}
		return s
	}
}

func EmitCreated(kp producer.Provider, tenant tenant.Model) func(s Model) {
	return func(s Model) {
		_ = kp(EnvEventTopicSessionStatus)(createdStatusEventProvider(tenant, s.SessionId(), s.AccountId(), s.CharacterId(), s.WorldId(), s.ChannelId()))
	}
}

func Teardown(l logrus.FieldLogger) func() {
	return func() {
		span := opentracing.StartSpan("teardown")
		defer span.Finish()
		tenant.ForAll(DestroyAll(l, span, GetRegistry()))
	}
}
