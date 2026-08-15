package environments

import (
	"atlas-configurations/outbox"
	"context"
	"encoding/json"
	"errors"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	envlib "github.com/Chronicle20/atlas/libs/atlas-env"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	outboxlib "github.com/Chronicle20/atlas/libs/atlas-outbox"
)

// EnvEnvironmentStatusTopic names the env var carrying the Kafka topic that
// environment CRUD events (and heartbeats) are enqueued onto. When unset
// (unit tests), Enqueue is skipped - matches EnvServiceStatusTopic /
// EnvTenantStatusTopic.
const EnvEnvironmentStatusTopic = "EVENT_TOPIC_CONFIGURATION_ENVIRONMENT_STATUS"

// ErrNameRequired is returned when Create/Update is called with an empty
// environment name. The empty env.Id is the legacy "not environment-aware"
// value (libs/atlas-env); it must never be persisted as an environment
// record.
var ErrNameRequired = errors.New("environments: name is required")

// ErrInvalidName is returned when the environment name fails env.Valid
// (task-232 P2 ingest validation).
var ErrInvalidName = errors.New("environments: name is not a well-formed environment id")

func environmentOutboxKey(name string) []byte {
	return []byte("environment:" + name)
}

// toRecord builds the env.Record wire shape (libs/atlas-env, Task 18) from
// a RestModel. This is the only place that shape is assembled, so the
// envelope's field names can never drift from RestModel's independently.
func toRecord(rm RestModel) envlib.Record {
	return envlib.Record{
		Name:      envlib.Id(rm.Name),
		Baseline:  envlib.Id(rm.Baseline),
		Namespace: rm.Namespace,
		Tenant:    rm.Tenant,
		Overrides: rm.Overrides,
		Phase:     rm.Phase,
	}
}

func enqueueEnvironmentStatus(tx *gorm.DB, name string, config any) error {
	topic := os.Getenv(EnvEnvironmentStatusTopic)
	if topic == "" {
		return nil
	}
	var value []byte
	if config != nil {
		v, err := outbox.NewEnvironmentEnvelope(name, config, time.Now())
		if err != nil {
			return err
		}
		value = v
	}
	return outboxlib.Enqueue(tx, outboxlib.Message{
		Topic: topic,
		Key:   environmentOutboxKey(name),
		Value: value,
	})
}

type Processor interface {
	ByNameProvider(name string) model.Provider[RestModel]
	AllProvider(page model.Page) model.Provider[model.Paged[RestModel]]
	GetByName(name string) (RestModel, error)
	Create(input RestModel) (RestModel, error)
	UpdateByName(name string, input RestModel) (RestModel, error)
	// Republish re-emits the persisted record for id unchanged, so
	// consumers can treat topic arrival as liveness (heartbeat.go).
	Republish(id envlib.Id) error
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	db  *gorm.DB
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) Processor {
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
		db:  db,
	}
}

var _ Processor = (*ProcessorImpl)(nil)

func (p *ProcessorImpl) ByNameProvider(name string) model.Provider[RestModel] {
	return model.Map(Make)(byNameEntityProvider(p.ctx)(name)(p.db))
}

func (p *ProcessorImpl) AllProvider(page model.Page) model.Provider[model.Paged[RestModel]] {
	return model.MapPaged(Make)(getAll(p.ctx, page)(p.db))(model.ParallelMap())
}

func Make(e Entity) (RestModel, error) {
	var overrides map[string]string
	if err := json.Unmarshal(e.Overrides, &overrides); err != nil {
		return RestModel{}, err
	}
	return RestModel{
		Id:        e.Id.String(),
		Name:      e.Name,
		Baseline:  e.Baseline,
		Namespace: e.Namespace,
		Tenant:    e.Tenant,
		Overrides: overrides,
		Phase:     e.Phase,
	}, nil
}

func (p *ProcessorImpl) GetByName(name string) (RestModel, error) {
	return p.ByNameProvider(name)()
}

func validateName(name string) error {
	if name == "" {
		return ErrNameRequired
	}
	if !envlib.Valid(envlib.Id(name)) {
		return ErrInvalidName
	}
	return nil
}

func (p *ProcessorImpl) Create(input RestModel) (RestModel, error) {
	if err := validateName(input.Name); err != nil {
		return RestModel{}, err
	}

	overrides, err := json.Marshal(input.Overrides)
	if err != nil {
		return RestModel{}, err
	}

	id := uuid.New()
	err = database.ExecuteTransaction(p.db, func(db *gorm.DB) error {
		if err := create(p.ctx, id, input.Name, input.Baseline, input.Namespace, input.Tenant, overrides, input.Phase)(db); err != nil {
			return err
		}
		return enqueueEnvironmentStatus(db, input.Name, toRecord(input))
	})
	if err != nil {
		return RestModel{}, err
	}

	input.Id = id.String()
	return input, nil
}

func (p *ProcessorImpl) UpdateByName(name string, input RestModel) (RestModel, error) {
	if err := validateName(name); err != nil {
		return RestModel{}, err
	}
	// The record's own name field always follows the URL/path identity, not
	// whatever the body happened to carry.
	input.Name = name

	overrides, err := json.Marshal(input.Overrides)
	if err != nil {
		return RestModel{}, err
	}

	err = database.ExecuteTransaction(p.db, func(db *gorm.DB) error {
		if err := update(p.ctx, name, input.Baseline, input.Namespace, input.Tenant, overrides, input.Phase)(db); err != nil {
			return err
		}
		return enqueueEnvironmentStatus(db, name, toRecord(input))
	})
	if err != nil {
		return RestModel{}, err
	}
	return input, nil
}

func (p *ProcessorImpl) Republish(id envlib.Id) error {
	name := string(id)
	rm, err := p.GetByName(name)
	if err != nil {
		return err
	}
	return database.ExecuteTransaction(p.db, func(db *gorm.DB) error {
		return enqueueEnvironmentStatus(db, name, toRecord(rm))
	})
}
