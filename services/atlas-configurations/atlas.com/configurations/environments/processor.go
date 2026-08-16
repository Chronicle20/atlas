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

// ErrInvalidPhase is returned when Create/Update is called with a phase
// that is not one of the four PRD FR-5.1 values. The phase is published on
// the compacted topic and decoded by every Task 20 subscriber, so an
// unvalidated value would poison every pod's registry projection until the
// key is overwritten - it must be rejected here, before persistence, not
// merely at the point something downstream fails to parse it.
var ErrInvalidPhase = errors.New("environments: phase must be one of PROVISIONING, ACTIVE, DEACTIVATING, DELETED")

// ErrIllegalPhaseTransition is returned when Update would move a record's
// phase somewhere other than forward-by-one (or a no-op) along PRD FR-5.1's
// literal chain: PROVISIONING -> ACTIVE -> DEACTIVATING -> DELETED. Neither
// design.md nor prd.md defines any additional transition rule (branching,
// abort-from-PROVISIONING-direct-to-DELETED, etc.) beyond that chain
// (task-232 fix round 1) - this enforces exactly what FR-5.1 states and
// nothing more.
var ErrIllegalPhaseTransition = errors.New("environments: phase transition is not legal (PRD FR-5.1: PROVISIONING -> ACTIVE -> DEACTIVATING -> DELETED, no skipping, no reverting)")

// phaseOrder is PRD FR-5.1's lifecycle, literally in the order the
// requirement states it.
var phaseOrder = []string{
	envlib.PhaseProvisioning,
	envlib.PhaseActive,
	envlib.PhaseDeactivating,
	envlib.PhaseDeleted,
}

func phaseIndex(phase string) int {
	for i, p := range phaseOrder {
		if p == phase {
			return i
		}
	}
	return -1
}

func validatePhase(phase string) error {
	if phaseIndex(phase) == -1 {
		return ErrInvalidPhase
	}
	return nil
}

// validatePhaseTransition allows only a no-op (re-PATCHing the same phase)
// or advancing exactly one step along phaseOrder. from/to are assumed
// already validatePhase-checked.
func validatePhaseTransition(from, to string) error {
	fi, ti := phaseIndex(from), phaseIndex(to)
	if ti == fi || ti == fi+1 {
		return nil
	}
	return ErrIllegalPhaseTransition
}

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
	if err := validatePhase(input.Phase); err != nil {
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
	if err := validatePhase(input.Phase); err != nil {
		return RestModel{}, err
	}
	existing, err := p.GetByName(name)
	if err != nil {
		return RestModel{}, err
	}
	if err := validatePhaseTransition(existing.Phase, input.Phase); err != nil {
		return RestModel{}, err
	}
	// The record's own name field always follows the URL/path identity, not
	// whatever the body happened to carry.
	input.Name = name

	// PATCH is partial: a caller that supplies only {"phase": "ACTIVE"} must
	// not zero out the columns it left out of the body. RestModel has no
	// pointer/omitempty fields (its wire shape also serves GET, which needs
	// every field present), so a field the caller omitted decodes as its Go
	// zero value - indistinguishable from an explicit empty value. Fall back
	// to the already-fetched existing record for any field still at its zero
	// value, so an update touches only what was actually supplied.
	if input.Baseline == "" {
		input.Baseline = existing.Baseline
	}
	if input.Namespace == "" {
		input.Namespace = existing.Namespace
	}
	if input.Tenant == "" {
		input.Tenant = existing.Tenant
	}
	if input.Overrides == nil {
		input.Overrides = existing.Overrides
	}

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
