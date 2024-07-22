package session

import (
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
		return model.SliceProviderToProviderAdapter[Model](AllInTenantProvider(tenant), CharacterIdPreciselyOneFilter(characterId))
	}
}

// IfPresentByCharacterId executes an Operator if a session exists for the characterId
func IfPresentByCharacterId(tenant tenant.Model) func(characterId uint32, f model.Operator[Model]) {
	return func(characterId uint32, f model.Operator[Model]) {
		model.IfPresent(ByCharacterIdModelProvider(tenant)(characterId), f)
	}
}

func CharacterIdFilter(referenceId uint32) model.Filter[Model] {
	return func(model Model) bool {
		return model.CharacterId() == referenceId
	}
}

// CharacterIdPreciselyOneFilter a filter which yields true when the characterId matches the one in the session
func CharacterIdPreciselyOneFilter(characterId uint32) model.PreciselyOneFilter[Model] {
	return func(models []Model) (Model, error) {
		return model.First(model.FixedSliceProvider(models), CharacterIdFilter(characterId))
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

//
//func SetWorldId(worldId byte) func(id uuid.UUID) Model {
//	return func(id uuid.UUID) Model {
//		s := Model{}
//		var ok bool
//		if s, ok = GetRegistry().Get(id); ok {
//			s = s.setWorldId(worldId)
//			GetRegistry().Update(s)
//			return s
//		}
//		return s
//	}
//}
//
//func SetChannelId(channelId byte) func(id uuid.UUID) Model {
//	return func(id uuid.UUID) Model {
//		s := Model{}
//		var ok bool
//		if s, ok = GetRegistry().Get(id); ok {
//			s = s.setChannelId(channelId)
//			GetRegistry().Update(s)
//			return s
//		}
//		return s
//	}
//}

func SessionCreated(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(s Model) {
	return func(s Model) {
		emitCreatedStatusEvent(l, span, tenant)(s.SessionId(), s.AccountId(), s.CharacterId(), s.WorldId(), s.ChannelId())
	}
}
