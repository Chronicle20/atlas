package services

import (
	"atlas-configurations/outbox"
	"atlas-configurations/services/service"
	"context"
	"encoding/json"
	"errors"
	"os"
	"time"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	outboxlib "github.com/Chronicle20/atlas/libs/atlas-outbox"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// EnvServiceStatusTopic names the env var carrying the Kafka topic that
// service config CRUD events are enqueued onto. When unset (e.g. in unit
// tests that don't exercise the publish path), Enqueue is skipped.
const EnvServiceStatusTopic = "EVENT_TOPIC_CONFIGURATION_SERVICE_STATUS"

// serviceOutboxKey returns the outbox message key for a service. The
// "service:" prefix prevents collisions with tenant keys on a shared
// topic should the two ever be merged.
func serviceOutboxKey(id uuid.UUID) []byte {
	return []byte("service:" + id.String())
}

// enqueueServiceStatus inserts the outbox row for a service config change
// inside the caller's transaction. When the topic env var is unset, the
// call is a no-op so unit tests don't have to set it.
func enqueueServiceStatus(tx *gorm.DB, id uuid.UUID, config any) error {
	topic := os.Getenv(EnvServiceStatusTopic)
	if topic == "" {
		return nil
	}
	var value []byte
	if config != nil {
		v, err := outbox.NewServiceEnvelope(id, config, time.Now())
		if err != nil {
			return err
		}
		value = v
	}
	return outboxlib.Enqueue(tx, outboxlib.Message{
		Topic: topic,
		Key:   serviceOutboxKey(id),
		Value: value,
	})
}

type ServiceType string

const (
	ServiceTypeLogin   = ServiceType("login-service")
	ServiceTypeChannel = ServiceType("channel-service")
	ServiceTypeDrops   = ServiceType("drops-service")
)

var validServiceTypes = map[ServiceType]bool{
	ServiceTypeLogin:   true,
	ServiceTypeChannel: true,
	ServiceTypeDrops:   true,
}

func IsValidServiceType(t string) bool {
	return validServiceTypes[ServiceType(t)]
}

type Processor struct {
	l   logrus.FieldLogger
	ctx context.Context
	db  *gorm.DB
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) *Processor {
	p := &Processor{
		l:   l,
		ctx: ctx,
		db:  db,
	}
	return p
}

func (p *Processor) ByIdProvider(id uuid.UUID) model.Provider[interface{}] {
	return model.Map(Make)(byIdEntityProvider(p.ctx)(id)(p.db))
}

func (p *Processor) AllProvider() model.Provider[[]interface{}] {
	return model.SliceMap(Make)(allEntityProvider(p.ctx)(p.db))()
}

func Make(e Entity) (interface{}, error) {
	if e.Type == ServiceTypeLogin {
		var rm service.LoginRestModel
		err := json.Unmarshal(e.Data, &rm)
		if err != nil {
			return nil, err
		}
		rm.Id = e.Id.String()
		rm.Type = string(e.Type)
		return rm, nil
	} else if e.Type == ServiceTypeChannel {
		var rm service.ChannelRestModel
		err := json.Unmarshal(e.Data, &rm)
		if err != nil {
			return nil, err
		}
		rm.Id = e.Id.String()
		rm.Type = string(e.Type)
		return rm, nil
	} else if e.Type == ServiceTypeDrops {
		var rm service.GenericRestModel
		err := json.Unmarshal(e.Data, &rm)
		if err != nil {
			return nil, err
		}
		rm.Id = e.Id.String()
		rm.Type = string(e.Type)
		return rm, nil
	}
	return nil, errors.New("invalid service type")
}

func (p *Processor) GetAll() ([]interface{}, error) {
	return p.AllProvider()()
}

func (p *Processor) GetById(id uuid.UUID) (interface{}, error) {
	return p.ByIdProvider(id)()
}

func (p *Processor) Create(input service.InputRestModel) (uuid.UUID, error) {
	serviceType := ServiceType(input.Type)
	if !IsValidServiceType(input.Type) {
		return uuid.Nil, errors.New("invalid service type")
	}

	// Marshal the input to JSON for storage
	res, err := json.Marshal(input)
	if err != nil {
		return uuid.Nil, err
	}
	rm := &json.RawMessage{}
	err = rm.UnmarshalJSON(res)
	if err != nil {
		return uuid.Nil, err
	}

	// Use ID from input if provided and valid, otherwise generate a new one
	var serviceId uuid.UUID
	if input.Id != "" {
		parsedId, parseErr := uuid.Parse(input.Id)
		if parseErr != nil {
			// If the provided ID is invalid, log and generate a new one
			p.l.WithError(parseErr).Warnf("Invalid UUID provided [%s], generating new one.", input.Id)
			serviceId = uuid.New()
		} else {
			// Use the provided UUID (including nil UUID if that's what was requested)
			serviceId = parsedId
		}
	} else {
		serviceId = uuid.New()
	}

	err = database.ExecuteTransaction(p.db, func(db *gorm.DB) error {
		if err := create(p.ctx, serviceId, serviceType, *rm)(db); err != nil {
			return err
		}
		return enqueueServiceStatus(db, serviceId, input)
	})
	if err != nil {
		return uuid.Nil, err
	}
	return serviceId, nil
}

func (p *Processor) UpdateById(serviceId uuid.UUID, input service.InputRestModel) error {
	serviceType := ServiceType(input.Type)
	if !IsValidServiceType(input.Type) {
		return errors.New("invalid service type")
	}

	// Marshal the input to JSON for storage
	res, err := json.Marshal(input)
	if err != nil {
		return err
	}
	rm := &json.RawMessage{}
	err = rm.UnmarshalJSON(res)
	if err != nil {
		return err
	}

	return database.ExecuteTransaction(p.db, func(db *gorm.DB) error {
		if err := update(p.ctx, serviceId, serviceType, *rm)(db); err != nil {
			return err
		}
		return enqueueServiceStatus(db, serviceId, input)
	})
}

func (p *Processor) DeleteById(serviceId uuid.UUID) error {
	return database.ExecuteTransaction(p.db, func(db *gorm.DB) error {
		if err := delete(p.ctx, serviceId)(db); err != nil {
			return err
		}
		// Tombstone: nil config → nil value, suitable for log-compacted topic.
		return enqueueServiceStatus(db, serviceId, nil)
	})
}
