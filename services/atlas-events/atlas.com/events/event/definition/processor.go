package definition

import (
	"atlas-events/event/registry"
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// Processor is the generic entry point onto event definition persistence.
// Every method is tenant-scoped through the *gorm.DB attached tenant callbacks
// (via WithContext); none of them switch on a type constant (FR-X3) — a
// definition's type is resolved to behavior only through registry.Get.
type Processor interface {
	GetById(id uuid.UUID) (Model, error)
	GetAllPaged(page model.Page) (model.Paged[Model], error)
	GetByType(theType string) ([]Model, error)
	GetEnabledByType(theType string) ([]Model, error)
	SetEnabled(id uuid.UUID, enabled bool) (Model, error)
	Create(m Model) (Model, error)
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	t   tenant.Model
	db  *gorm.DB
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) Processor {
	t := tenant.MustFromContext(ctx)

	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
		t:   t,
		db:  db,
	}
}

var _ Processor = (*ProcessorImpl)(nil)

func (p *ProcessorImpl) GetById(id uuid.UUID) (Model, error) {
	return model.Map(Make)(getByIdProvider(id)(p.db.WithContext(p.ctx)))()
}

func (p *ProcessorImpl) GetAllPaged(page model.Page) (model.Paged[Model], error) {
	ep := getAllPagedProvider(page)(p.db.WithContext(p.ctx))
	return model.MapPaged(Make)(ep)(model.ParallelMap())()
}

func (p *ProcessorImpl) GetByType(theType string) ([]Model, error) {
	return model.SliceMap(Make)(getByTypeProvider(theType)(p.db.WithContext(p.ctx)))(model.ParallelMap())()
}

func (p *ProcessorImpl) GetEnabledByType(theType string) ([]Model, error) {
	return model.SliceMap(Make)(getEnabledByTypeProvider(theType)(p.db.WithContext(p.ctx)))(model.ParallelMap())()
}

// validateConfiguration resolves the handler registered for m.Type() and asks
// it to validate m.Configuration() (FR-D6). A type with no registered handler
// is rejected the same way. This is the single place the FR-D6 rule lives —
// every path that can create an event definition (the HTTP Processor.Create
// and the seeder's DefinitionSubdomain.BulkCreate) must call it before the
// row is ever written, so there is no path by which an unvalidatable
// definition reaches the table to fail later at trigger time.
func validateConfiguration(m Model) error {
	h, ok := registry.Get(m.Type())
	if !ok {
		return fmt.Errorf("no handler registered for event type [%s]", m.Type())
	}
	if err := h.ValidateConfiguration(m.Configuration()); err != nil {
		return fmt.Errorf("configuration rejected for event type [%s]: %w", m.Type(), err)
	}
	return nil
}

// Create validates m.Configuration() (FR-D6) before persisting it.
func (p *ProcessorImpl) Create(m Model) (Model, error) {
	p.l.Debugf("Creating event definition [%s] of type [%s].", m.Name(), m.Type())

	if err := validateConfiguration(m); err != nil {
		return Model{}, err
	}

	result, err := create(p.db.WithContext(p.ctx))(p.t.Id())(m)
	if err != nil {
		p.l.WithError(err).Errorf("Failed to create event definition [%s].", m.Name())
		return Model{}, err
	}
	return result, nil
}

// SetEnabled flips the enabled flag and nothing else (FR-D5) — disabling a
// definition must never touch its occurrences, and this processor has no path
// that could.
func (p *ProcessorImpl) SetEnabled(id uuid.UUID, enabled bool) (Model, error) {
	p.l.Debugf("Setting event definition [%s] enabled=[%t].", id, enabled)

	if err := setEnabled(p.db.WithContext(p.ctx))(id)(enabled); err != nil {
		p.l.WithError(err).Errorf("Failed to set enabled on event definition [%s].", id)
		return Model{}, err
	}
	return p.GetById(id)
}

// singleOccurrence reports whether theType's registered handler names a
// concurrency key that is a per-type constant rather than one that varies
// with the work context (FR-UI4): true ⇒ at most one occurrence can ever
// exist, so the UI may render live occurrence state directly on the
// definition row.
//
// This asks the handler directly (registry.Handler.ConcurrencyKeyIsConstant)
// rather than probing ConcurrencyKey with two distinct payloads and comparing
// results, which R33-4 found unsound: json.Unmarshal silently ignores
// unknown fields, so a handler that decodes the probe into a typed struct
// (e.g. CRIMSON_BALROG's WorkContext) gets the SAME zero value back for every
// unrecognized probe — collapsing two distinct probes to equal keys and
// misreporting a varying handler as constant. An unregistered type is
// treated as "varies" (false) — the safe default is to link out to the
// filtered occurrence list rather than imply a single state that may not
// hold.
func singleOccurrence(_ context.Context, theType string) bool {
	h, ok := registry.Get(theType)
	if !ok {
		return false
	}
	return h.ConcurrencyKeyIsConstant()
}
