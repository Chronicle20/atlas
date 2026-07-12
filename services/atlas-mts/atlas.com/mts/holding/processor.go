package holding

import (
	"context"
	"fmt"
	"time"

	"atlas-mts/saga"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// SagaEmitter abstracts the saga-command emission so the take-home flow can be
// exercised without a live Kafka broker. The production implementation is the
// saga package's Processor; tests inject a capturing stub. Mirrors the listing
// package's SagaEmitter (Task 4.1).
type SagaEmitter interface {
	Create(s saga.Saga) error
}

// takeHomeSagaBaseTimeout and takeHomeSagaPerStepTimeout define the
// step-count-scaled timeout for the take-home (WithdrawFromMts) saga. The
// orchestrator processes saga steps serially over Kafka, so the timeout budget
// must grow with the number of effective steps; a flat timeout rolls back
// legitimate multi-step sagas (see the preset-creation timeout bug,
// bug_preset_creation_saga_flat_timeout).
//
// The per-step budget must cover ONE full cross-service Kafka round-trip under a
// stressed broker (seconds, not ms) — matching the list/buy flows, which moved to
// 15s/step after an observed ~11s step tripped a 1s/step budget and fired
// compensation while the step was still in flight (listing/processor.go). Take-home
// (release + accept_to_character) is the same failure family, so it uses the same
// per-step budget rather than the old flat 1s.
const (
	takeHomeSagaBaseTimeout    = 10 * time.Second
	takeHomeSagaPerStepTimeout = 15 * time.Second
)

// Processor exposes the REST-facing operations over take-home holdings plus the
// take-home initiation flow (TakeHome).
type Processor interface {
	GetAll() model.Provider[[]Model]
	GetById(id string) (Model, error)
	// GetBySerial resolves a holding by its per-(tenant, world) ITC serial (the
	// client's nITCSN). It is the resolver the channel take-home ITC_OPERATION arm
	// uses to translate the wire's uint32 serial into the UUID-keyed holding.
	GetBySerial(worldId world.Id, sn uint32) (Model, error)
	Create(m Model) (Model, error)
	GetByOwner(worldId world.Id, ownerId uint32) ([]Model, error)
	GetByCharacter(ownerId uint32) ([]Model, error)
	// TakeHome initiates the owner's withdrawal of a holding into inventory by
	// emitting a WithdrawFromMts saga keyed by a fresh transaction id. It returns
	// that transaction id. It does NOT soft-delete the holding row directly — the
	// saga's ReleaseFromMtsHolding custody command soft-deletes it (idempotently on
	// replay) and AcceptToCharacter grants the item to inventory.
	TakeHome(holdingId string, characterId uint32, worldId world.Id, inventoryType byte, slot int16) (uuid.UUID, error)
	// Release soft-deletes the holding row by id in one local DB transaction
	// (idempotent on replay), capturing the pre-delete snapshot so the consumer can
	// gate the ITEM_TAKEN_HOME event. It is the custody ReleaseFromMtsHolding logic.
	Release(holdingId string) (ReleaseResult, error)
	// RestoreHolding un-soft-deletes the holding row by id (the compensating inverse
	// of Release) in one local DB transaction, idempotent on replay.
	RestoreHolding(holdingId string) error
}

type ProcessorImpl struct {
	l       logrus.FieldLogger
	ctx     context.Context
	db      *gorm.DB
	emitter SagaEmitter
}

// Option mutates a ProcessorImpl during construction.
type Option func(*ProcessorImpl)

// WithSagaEmitter overrides the saga emitter (default: the real saga Processor).
// Tests inject a capturing stub so the saga can be asserted without Kafka.
func WithSagaEmitter(e SagaEmitter) Option {
	return func(p *ProcessorImpl) {
		p.emitter = e
	}
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB, opts ...Option) Processor {
	p := &ProcessorImpl{l: l, ctx: ctx, db: db}
	p.emitter = saga.NewProcessor(l, ctx)
	for _, opt := range opts {
		opt(p)
	}
	return p
}

func (p *ProcessorImpl) GetAll() model.Provider[[]Model] {
	return model.SliceMap(modelFromEntity)(getAll()(p.db.WithContext(p.ctx)))()
}

func (p *ProcessorImpl) GetById(id string) (Model, error) {
	return GetById(id)(p.db.WithContext(p.ctx))()
}

// GetBySerial resolves a holding by its per-(tenant, world) ITC serial.
func (p *ProcessorImpl) GetBySerial(worldId world.Id, sn uint32) (Model, error) {
	return GetBySerial(worldId, sn)(p.db.WithContext(p.ctx))()
}

// Create persists a new holding and returns the stored Model (with its assigned
// surrogate id).
func (p *ProcessorImpl) Create(m Model) (Model, error) {
	return CreateHolding(p.db.WithContext(p.ctx), m)
}

// GetByOwner returns the holdings for a character in a world. The signature
// mirrors the getByOwner provider exactly.
func (p *ProcessorImpl) GetByOwner(worldId world.Id, ownerId uint32) ([]Model, error) {
	return model.SliceMap(modelFromEntity)(getByOwner(worldId, ownerId)(p.db.WithContext(p.ctx)))()()
}

// GetByCharacter returns all holdings for a character (owner) across worlds. The
// signature mirrors the getByCharacter provider exactly.
func (p *ProcessorImpl) GetByCharacter(ownerId uint32) ([]Model, error) {
	return model.SliceMap(modelFromEntity)(getByCharacter(ownerId)(p.db.WithContext(p.ctx)))()()
}

// TakeHome is the server-authoritative take-home initiation flow. It builds and
// emits a WithdrawFromMts saga that the orchestrator expands into
// release_from_mts_holding + accept_to_character: the holding row is soft-deleted
// on release (idempotently — a replay finds it already released and re-acks
// without re-granting) and the item is granted to the character's inventory;
// compensation re-creates the holding if the accept fails.
//
// TakeHome does NOT soft-delete the holding row here — that side effect belongs
// to the saga's ReleaseFromMtsHolding custody command, which is the idempotency
// boundary on replay.
//
// slot is advisory: WithdrawFromMtsPayload carries no Slot field, so the
// inventory grant (AcceptToCharacter) assigns a free slot during expansion. The
// parameter is accepted for the REST contract but not propagated to the saga.
func (p *ProcessorImpl) TakeHome(holdingId string, characterId uint32, worldId world.Id, inventoryType byte, slot int16) (uuid.UUID, error) {
	hid, err := uuid.Parse(holdingId)
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid holding id %q: %w", holdingId, err)
	}

	transactionId := uuid.New()

	builder := saga.NewBuilder().
		SetTransactionId(transactionId).
		SetSagaType(saga.MtsOperation).
		SetInitiatedBy(fmt.Sprintf("character_%d", characterId))

	// WithdrawFromMts (composite): the orchestrator expands it into
	// release_from_mts_holding + accept_to_character. release_from_mts_holding
	// soft-deletes the holding by id (idempotent on replay); accept_to_character
	// grants the item to the character's inventory.
	builder.AddStep("withdraw_from_mts", saga.Pending, saga.WithdrawFromMts, saga.WithdrawFromMtsPayload{
		TransactionId: transactionId,
		CharacterId:   characterId,
		WorldId:       worldId,
		HoldingId:     hid,
		InventoryType: inventoryType,
	})

	// Timeout MUST be set explicitly and scaled for the effective step count.
	// N=2: the WithdrawFromMts composite expands to 2 steps
	// (release_from_mts_holding + accept_to_character). A flat timeout rolls back a
	// legitimate multi-step saga under a stressed broker.
	const withdrawFromMtsExpandedSteps = 2 // release_from_mts_holding + accept_to_character
	builder.SetTimeout(takeHomeSagaBaseTimeout + time.Duration(withdrawFromMtsExpandedSteps)*takeHomeSagaPerStepTimeout)

	if err := p.emitter.Create(builder.Build()); err != nil {
		return uuid.Nil, err
	}
	return transactionId, nil
}
