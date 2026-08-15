package scheduling

import (
	"atlas-events/event/definition"
	"atlas-events/event/occurrence"
	"atlas-events/event/registry"
	"atlas-events/event/transition"
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// Default retry policy applied when the caller never overrides it via
// SetMaxAttempts/SetBackoff. Production wiring (NewPoller) always overrides
// backoff from Config with a non-zero default (poller.go); a bare
// NewProcessor defaults to zero backoff — immediate retry — since a
// processor built directly (as every test in this package does) has no
// Config to derive one from.
const (
	defaultMaxAttempts = 5
	defaultBackoff     = 0
)

// errNoHandler marks a work row whose definition type has no registered
// handler as a permanent, non-retryable failure (design §5.2): no number of
// attempts will make a handler appear, so the row goes straight to FAILED on
// the first try instead of bouncing through PENDING/backoff until
// MaxAttempts is exhausted.
type errNoHandler struct{ theType string }

func (e errNoHandler) Error() string {
	return fmt.Sprintf("no handler for type %s", e.theType)
}

// Executor is the work function ExecuteOne invokes for a claimed row. The
// zero value is nil, which selects the real dispatch — the
// registry/definition/occurrence path in dispatch(). Tests supply an
// override via NewProcessorWithExecutor so they can assert the §5.2 outcome
// state machine without registering a real registry.Handler.
type Executor func(Model) error

// Processor is the generic entry point onto scheduled-work claiming,
// lease reclaim and execution. Unlike definition.Processor/occurrence.Processor
// it is NOT constructed from a tenant-scoped context: the poller is the one
// deliberate exception to tenant filtering (design §4.2) and must see every
// tenant's due work. ExecuteOne re-enters a tenant-scoped context for the one
// row it is about to run, before invoking any handler, so a handler never
// runs unfiltered (FR-N13, FR-N14).
type Processor interface {
	ClaimBatch(instanceId string, limit int) ([]Model, error)
	Reclaim(lease time.Duration) (int64, error)
	ExecuteOne(m Model) error
}

type ProcessorImpl struct {
	l           logrus.FieldLogger
	ctx         context.Context
	db          *gorm.DB
	executor    Executor
	maxAttempts int
	backoff     time.Duration
}

// NewProcessor constructs a poller-facing Processor. ctx carries no tenant —
// it is the background service lifecycle context, not a request context.
func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) *ProcessorImpl {
	return &ProcessorImpl{
		l:           l,
		ctx:         ctx,
		db:          db,
		maxAttempts: defaultMaxAttempts,
		backoff:     defaultBackoff,
	}
}

// NewProcessorWithExecutor is the test-only constructor that substitutes
// executor for the real registry/definition/occurrence dispatch, so tests
// can assert the outcome state machine (design §5.2) without a registered
// Handler.
func NewProcessorWithExecutor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB, executor Executor) *ProcessorImpl {
	p := NewProcessor(l, ctx, db)
	p.executor = executor
	return p
}

var _ Processor = (*ProcessorImpl)(nil)

// SetMaxAttempts overrides the retry ceiling ExecuteOne's outcome policy
// applies (design §5.2, FR-S9).
func (p *ProcessorImpl) SetMaxAttempts(n int) { p.maxAttempts = n }

// SetBackoff overrides the delay ExecuteOne applies to a row that errored
// and still has attempts remaining.
func (p *ProcessorImpl) SetBackoff(d time.Duration) { p.backoff = d }

// ClaimBatch atomically moves up to `limit` due rows PENDING -> PROCESSING and
// stamps claimedBy/claimedAt (FR-S6). SKIP LOCKED means a second replica takes
// the NEXT rows rather than blocking, so both keep working and a stuck row is
// isolated to its own claimer (FR-N6) — the same idiom already proven in
// libs/atlas-outbox/drainer.go:223 and the saga orchestrator's
// saga/store.go:242.
//
// SQLite silently drops the clause.Locking clause instead of erroring
// (gorm.io/driver/sqlite@v1.6.0/sqlite.go:124-129), so this method builds and
// runs correctly under the in-memory test harness, but SQLite tests prove
// only the state machine below — nothing about row locking under contention.
// That is covered by poller_integration_test.go against real Postgres.
func (p *ProcessorImpl) ClaimBatch(instanceId string, limit int) ([]Model, error) {
	var claimedEntities []Entity
	err := database.ExecuteTransaction(p.db, func(tx *gorm.DB) error {
		var rows []Entity
		if err := tx.WithContext(database.WithoutTenantFilter(p.ctx)).
			Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
			Where("state = ? AND execute_at <= ?", StatePending, time.Now()).
			Order("execute_at ASC").Limit(limit).Find(&rows).Error; err != nil {
			return err
		}
		if len(rows) == 0 {
			return nil
		}

		ids := make([]uuid.UUID, 0, len(rows))
		for _, r := range rows {
			ids = append(ids, r.ID)
		}
		now := time.Now()
		if err := tx.WithContext(database.WithoutTenantFilter(p.ctx)).
			Model(&Entity{}).Where("id IN ?", ids).
			Updates(map[string]any{
				"state": StateProcessing, "claimed_by": instanceId, "claimed_at": now,
			}).Error; err != nil {
			return err
		}

		claimedEntities = rows
		return nil
	})
	if err != nil {
		return nil, err
	}

	claimed := make([]Model, 0, len(claimedEntities))
	for _, e := range claimedEntities {
		m, err := Make(e)
		if err != nil {
			return nil, err
		}
		claimed = append(claimed, m)
	}
	return claimed, nil
}

// Reclaim moves rows abandoned by a dead replica — state PROCESSING and
// claimed_at older than lease — back to PENDING, incrementing attempts
// (FR-S7). Like ClaimBatch this deliberately sees every tenant's stuck work.
func (p *ProcessorImpl) Reclaim(lease time.Duration) (int64, error) {
	cutoff := time.Now().Add(-lease)
	res := p.db.WithContext(database.WithoutTenantFilter(p.ctx)).
		Model(&Entity{}).
		Where("state = ? AND claimed_at < ?", StateProcessing, cutoff).
		Updates(map[string]any{
			"state":      StatePending,
			"attempts":   gorm.Expr("attempts + 1"),
			"claimed_by": "",
			"claimed_at": nil,
		})
	return res.RowsAffected, res.Error
}

// ExecuteOne dispatches the claimed row m and writes the resulting state
// transition per the design §5.2 outcome table. If NewProcessorWithExecutor
// supplied a test executor, that function stands in for the real dispatch —
// the outcome policy below is identical either way.
func (p *ProcessorImpl) ExecuteOne(m Model) error {
	var workErr error
	if p.executor != nil {
		workErr = p.executor(m)
	} else {
		workErr = p.dispatch(m)
	}
	return p.applyOutcome(m, workErr)
}

// tenantContext rebuilds a tenant-scoped context for m's owning tenant. This
// is the "re-enters a tenant-scoped context per claimed row" step design §4.2
// requires before any handler or tenant-filtered query runs for this row.
func (p *ProcessorImpl) tenantContext(m Model) context.Context {
	t, _ := tenant.Create(m.TenantId(), m.TenantRegion(), m.TenantMajor(), m.TenantMinor())
	return tenant.WithContext(p.ctx, t)
}

// dispatch is the real work function: read the definition, resolve its
// handler from the registry, and drive Evaluate/Start (TRIGGER_EVALUATION) or
// Advance (OCCURRENCE_TRANSITION). It never switches on a type constant
// (FR-X3) — registry.Get is the only seam into event-specific behavior.
func (p *ProcessorImpl) dispatch(m Model) error {
	tctx := p.tenantContext(m)

	dp := definition.NewProcessor(p.l, tctx, p.db)
	d, err := dp.GetById(m.DefinitionId())
	if err != nil {
		return err
	}
	if !d.Enabled() {
		// Definition disabled: COMPLETED, no occurrence (design §5.2).
		return nil
	}

	h, ok := registry.Get(d.Type())
	if !ok {
		return errNoHandler{theType: d.Type()}
	}

	rd := registry.Definition{
		Id:            d.Id(),
		Type:          d.Type(),
		Name:          d.Name(),
		Enabled:       d.Enabled(),
		Configuration: d.Configuration(),
	}
	w := registry.Work{Id: m.Id(), Type: m.Type(), Context: m.Context()}
	op := occurrence.NewProcessor(p.l, tctx, p.db)
	triggerRef := m.Id().String()

	switch m.Type() {
	case WorkTypeTriggerEvaluation:
		seed, err := h.Evaluate(tctx, rd, w)
		if err != nil {
			return err
		}
		if seed == nil {
			// Ordinary "no occurrence" outcome (FR-B7, FR-B8): completes the row.
			return nil
		}

		occ, err := op.CreateFromSeed(d, *seed, triggerRef)
		if err != nil {
			if errors.Is(err, occurrence.ErrConcurrencyKeyTaken) {
				// Someone else already created it — treat as success.
				return nil
			}
			return err
		}

		prog, err := h.Start(tctx, toRegistryOccurrence(occ))
		if err != nil {
			return err
		}
		_, err = op.ApplyProgress(occ, prog, transition.TriggerTypeOccurrenceStart, triggerRef)
		return err

	case WorkTypeOccurrenceTransition:
		occ, err := op.GetById(m.OccurrenceId())
		if err != nil {
			return err
		}
		prog, err := h.Advance(tctx, toRegistryOccurrence(occ), w)
		if err != nil {
			return err
		}
		_, err = op.ApplyProgress(occ, prog, transition.TriggerTypeScheduledWork, triggerRef)
		return err

	default:
		return fmt.Errorf("scheduling: unknown work type [%s]", m.Type())
	}
}

// toRegistryOccurrence narrows an occurrence.Model to the read-only view a
// registry.Handler receives.
func toRegistryOccurrence(o occurrence.Model) registry.Occurrence {
	return registry.Occurrence{
		Id:           o.Id(),
		DefinitionId: o.DefinitionId(),
		Type:         o.Type(),
		Stage:        o.Stage(),
		Context:      o.Context(),
		WorldId:      o.WorldId(),
		ChannelId:    o.ChannelId(),
		VoyageId:     o.VoyageId(),
		StartedAt:    o.StartedAt(),
	}
}

// applyOutcome writes the row transition ExecuteOne resolved for m, per the
// design §5.2 outcome table. It always re-enters m's tenant-scoped context
// (design §4.2) for the write, even though pinning on id makes the tenant
// filter redundant here — every write path in this package stays
// consistently tenant-scoped.
func (p *ProcessorImpl) applyOutcome(m Model, workErr error) error {
	db := p.db.WithContext(p.tenantContext(m))

	if workErr == nil {
		return db.Model(&Entity{}).Where("id = ?", m.Id()).
			Update("state", StateCompleted).Error
	}

	var nh errNoHandler
	if errors.As(workErr, &nh) {
		return db.Model(&Entity{}).Where("id = ?", m.Id()).
			Updates(map[string]any{"state": StateFailed, "last_error": workErr.Error()}).Error
	}

	attempts := m.Attempts() + 1
	if attempts < p.maxAttempts {
		return db.Model(&Entity{}).Where("id = ?", m.Id()).
			Updates(map[string]any{
				"state":      StatePending,
				"attempts":   attempts,
				"execute_at": time.Now().Add(p.backoff),
				"last_error": workErr.Error(),
			}).Error
	}
	return db.Model(&Entity{}).Where("id = ?", m.Id()).
		Updates(map[string]any{
			"state":      StateFailed,
			"attempts":   attempts,
			"last_error": workErr.Error(),
		}).Error
}
