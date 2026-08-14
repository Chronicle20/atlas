package pending_change

import (
	"atlas-character/character"
	"atlas-character/configuration"
	"atlas-character/kafka/message"
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	pendingchange2 "atlas-character/kafka/message/pending_change"

	sagamsg "atlas-character/kafka/message/saga"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	database "github.com/Chronicle20/atlas/libs/atlas-database"
	outbox "github.com/Chronicle20/atlas/libs/atlas-outbox"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// DefaultExpiry is how long a pending request survives before the sweep
// expires and refunds it (design §5.5) when a tenant has no imprint-configs
// resource. NewProcessor now reads the tenant-configurable expiry from
// configuration.GetRegistry() (FR-2.6); this constant is what that registry's
// own DefaultConfig() carries, and processor_test.go still asserts against it
// directly since an unseeded test tenant resolves to the same value.
const DefaultExpiry = configuration.DefaultPendingExpiry

// ErrIneligible is the sentinel every request rejection wraps, so a caller that
// does not care which gate failed can still branch on the class. The concrete
// reason travels on IneligibleError.
var ErrIneligible = errors.New("request ineligible")

// IneligibleError carries the reason-taxonomy string (design §6) the REST layer
// and the client-facing notification both render.
type IneligibleError struct {
	Reason string
}

func (e IneligibleError) Error() string {
	return fmt.Sprintf("request ineligible: %s", e.Reason)
}

func (e IneligibleError) Unwrap() error {
	return ErrIneligible
}

// WorldTransferStarterFunc dispatches the world-transfer saga for an accepted
// request. It is injected rather than imported for the same reason
// character.NameReservedFunc is: the saga contract (libs/atlas-saga's
// WorldTransfer type and its five actions) is owned elsewhere, and the record
// deliberately stays PENDING until that saga's terminal event drives Resolve.
type WorldTransferStarterFunc func(l logrus.FieldLogger, ctx context.Context, mb *message.Buffer, m Model) error

type Processor interface {
	Create(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, changeType string, requestedName string, destinationWorldId world.Id, assetId *uint32) (Model, error)
	CreateAndEmit(transactionId uuid.UUID, characterId uint32, changeType string, requestedName string, destinationWorldId world.Id, assetId *uint32) (Model, error)
	Resolve(mb *message.Buffer) func(id uuid.UUID, status string, reason string) (Model, bool, error)
	ResolveAndEmit(id uuid.UUID, status string, reason string) (Model, bool, error)
	ApplyForCharacter(characterId uint32) error
	RenotifyForCharacter(characterId uint32) error
	Sweep(now time.Time) error
	GetByCharacterId(characterId uint32) ([]Model, error)
	GetById(id uuid.UUID) (Model, error)
	NameReserved(name string) (bool, error)
	WithTransaction(tx *gorm.DB) Processor
	WithWorldTransferStarter(f WorldTransferStarterFunc) Processor
}

type ProcessorImpl struct {
	l                    logrus.FieldLogger
	ctx                  context.Context
	db                   *gorm.DB
	t                    tenant.Model
	expiry               time.Duration
	worldTransferStarter WorldTransferStarterFunc
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) Processor {
	return &ProcessorImpl{
		l:      l,
		ctx:    ctx,
		db:     db,
		t:      tenant.MustFromContext(ctx),
		expiry: configuration.GetRegistry().Get(l, ctx).PendingExpiry(),
	}
}

var _ Processor = (*ProcessorImpl)(nil)

func (p *ProcessorImpl) WithTransaction(tx *gorm.DB) Processor {
	return &ProcessorImpl{
		l:                    p.l,
		ctx:                  p.ctx,
		db:                   tx,
		t:                    p.t,
		expiry:               p.expiry,
		worldTransferStarter: p.worldTransferStarter,
	}
}

func (p *ProcessorImpl) WithWorldTransferStarter(f WorldTransferStarterFunc) Processor {
	return &ProcessorImpl{
		l:                    p.l,
		ctx:                  p.ctx,
		db:                   p.db,
		t:                    p.t,
		expiry:               p.expiry,
		worldTransferStarter: f,
	}
}

// NameReservedFor adapts this package's reservation lookup to
// character.NameReservedFunc. The dependency runs one way only: pending_change
// imports character (for the apply path), character never imports
// pending_change — main.go closes the loop with WithNameReserved.
func NameReservedFor(db *gorm.DB) func(l logrus.FieldLogger, ctx context.Context, name string) (bool, error) {
	return func(l logrus.FieldLogger, ctx context.Context, name string) (bool, error) {
		return NewProcessor(l, ctx, db).NameReserved(name)
	}
}

// reasonForNameValidity maps a character.NameValidityResult reason onto the
// design §6 reason taxonomy.
func reasonForNameValidity(reason string) string {
	switch reason {
	case "length":
		return "name_invalid_length"
	case "regex":
		return "name_invalid_charset"
	case "duplicate":
		return "name_taken"
	case "reserved":
		return "name_reserved"
	default:
		return "name_invalid"
	}
}

func (p *ProcessorImpl) Create(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, changeType string, requestedName string, destinationWorldId world.Id, assetId *uint32) (Model, error) {
	return func(transactionId uuid.UUID, characterId uint32, changeType string, requestedName string, destinationWorldId world.Id, assetId *uint32) (Model, error) {
		c, err := character.NewProcessor(p.l, p.ctx, p.db).GetById()(characterId)
		if err != nil {
			return Model{}, err
		}

		switch changeType {
		case TypeNameChange:
			res, err := character.NewProcessor(p.l, p.ctx, p.db).
				WithNameReserved(NameReservedFor(p.db)).
				CheckNameValidity(requestedName, c.WorldId(), character.NameScopeTenant)
			if err != nil {
				return Model{}, err
			}
			if !res.Valid {
				return Model{}, IneligibleError{Reason: reasonForNameValidity(res.Reason)}
			}
		case TypeWorldTransfer:
			if reason, ok := p.evaluateTransferEligibility(c, destinationWorldId); !ok {
				return Model{}, IneligibleError{Reason: reason}
			}
		default:
			return Model{}, IneligibleError{Reason: "unknown_change_type"}
		}

		now := time.Now()
		b := NewBuilder().
			SetId(uuid.New()).
			SetCharacterId(characterId).
			SetType(changeType).
			SetStatus(StatusPending).
			SetRequestedName(requestedName).
			SetDestinationWorldId(destinationWorldId).
			SetSourceWorldId(c.WorldId()).
			SetTransactionId(transactionId).
			SetCreatedAt(now).
			SetExpiresAt(now.Add(p.expiry))
		if assetId != nil {
			b = b.SetAssetId(*assetId)
		}

		m, err := create(p.db, p.t.Id(), b.Build())
		if err != nil {
			return Model{}, err
		}

		if err := mb.Put(pendingchange2.EnvEventTopic, createdEventProvider(m)); err != nil {
			return Model{}, err
		}
		// Consumption is at request acceptance (FR-2.8). The purchase path has
		// no asset to destroy — atlas-cashshop consumes the entitlement off the
		// PENDING_CHANGE_CREATED event instead.
		if m.HasAsset() {
			if err := mb.Put(sagamsg.EnvCommandTopic, destroyAssetCommandProvider(m)); err != nil {
				return Model{}, err
			}
		}
		p.l.Infof("Created pending change [%s] type [%s] for character [%d] in world [%d].", m.Id(), changeType, characterId, c.WorldId())
		return m, nil
	}
}

func (p *ProcessorImpl) CreateAndEmit(transactionId uuid.UUID, characterId uint32, changeType string, requestedName string, destinationWorldId world.Id, assetId *uint32) (Model, error) {
	var out Model
	txErr := database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
		return message.Emit(outbox.EmitProvider(p.l, p.ctx, tx))(func(buf *message.Buffer) error {
			var err error
			out, err = p.WithTransaction(tx).Create(buf)(transactionId, characterId, changeType, requestedName, destinationWorldId, assetId)
			return err
		})
	})
	if txErr != nil {
		return Model{}, txErr
	}
	return out, nil
}

// Resolve moves a record to a terminal status and, ONLY when the row actually
// moved, emits the refund and the resolved notification. This is the whole of
// the idempotency contract: a redelivered command sees moved == false, emits
// nothing, and mints nothing (design §3.10). There is deliberately no other
// path in this package that puts awardAssetCommandProvider on a buffer.
func (p *ProcessorImpl) Resolve(mb *message.Buffer) func(id uuid.UUID, status string, reason string) (Model, bool, error) {
	return func(id uuid.UUID, status string, reason string) (Model, bool, error) {
		m, moved, err := transition(p.db, p.t.Id(), id, status, reason, time.Now())
		if err != nil {
			return Model{}, false, err
		}
		if !moved {
			p.l.Debugf("Pending change [%s] already terminal (%s); skipping refund and notification.", id, m.Status())
			return m, false, nil
		}

		if status != StatusApplied && m.HasAsset() {
			if err := mb.Put(sagamsg.EnvCommandTopic, awardAssetCommandProvider(m)); err != nil {
				return Model{}, false, err
			}
		}
		if err := mb.Put(pendingchange2.EnvEventTopic, resolvedEventProvider(m)); err != nil {
			return Model{}, false, err
		}
		p.l.Infof("Pending change [%s] for character [%d] transitioned PENDING -> %s, reason [%s].", m.Id(), m.CharacterId(), status, reason)
		return m, true, nil
	}
}

func (p *ProcessorImpl) ResolveAndEmit(id uuid.UUID, status string, reason string) (Model, bool, error) {
	var out Model
	var moved bool
	txErr := database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
		return message.Emit(outbox.EmitProvider(p.l, p.ctx, tx))(func(buf *message.Buffer) error {
			var err error
			out, moved, err = p.WithTransaction(tx).Resolve(buf)(id, status, reason)
			return err
		})
	})
	if txErr != nil {
		return Model{}, false, txErr
	}
	return out, moved, nil
}

// ApplyForCharacter runs every pending request the character holds. It is
// driven by the LOGOUT status event, which is the proof the character is not
// live in a channel (FR-2.4) — this is the only writer of the applied name.
func (p *ProcessorImpl) ApplyForCharacter(characterId uint32) error {
	ms, err := getPendingByCharacterId(p.db.WithContext(p.ctx), p.t.Id(), characterId)
	if err != nil {
		return err
	}
	for _, m := range ms {
		switch m.Type() {
		case TypeNameChange:
			if err := p.applyNameChange(m); err != nil {
				return err
			}
		case TypeWorldTransfer:
			if err := p.startWorldTransfer(m); err != nil {
				return err
			}
		default:
			return IneligibleError{Reason: "unknown_change_type"}
		}
	}
	return nil
}

func (p *ProcessorImpl) applyNameChange(m Model) error {
	return database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
		return message.Emit(outbox.EmitProvider(p.l, p.ctx, tx))(func(buf *message.Buffer) error {
			cp := character.NewProcessor(p.l, p.ctx, tx)
			c, err := cp.GetById()(m.CharacterId())
			if err != nil {
				return err
			}

			// Re-validated WITHOUT the reservation seam: this record's own
			// reservation is still live at this point and would reject the very
			// name it is holding. The tenant-wide duplicate check is the gate
			// that matters here — a creation could have taken the name in the
			// interim (design §5.2).
			res, err := cp.CheckNameValidity(m.RequestedName(), c.WorldId(), character.NameScopeTenant)
			if err != nil {
				return err
			}
			if !res.Valid {
				p.l.Infof("Pending name change [%s] lost its name [%s]; rejecting and refunding.", m.Id(), m.RequestedName())
				_, _, err = p.WithTransaction(tx).Resolve(buf)(m.Id(), StatusRejected, reasonForNameValidity(res.Reason))
				return err
			}

			// Gender is carried through explicitly: character.Update treats a
			// zero Gender as an explicit set (every other field's zero means
			// "absent"), so a bare RestModel would demote a female character.
			if err := cp.Update(buf)(m.TransactionId(), m.CharacterId(), character.RestModel{
				Name:   m.RequestedName(),
				Gender: c.Gender(),
			}); err != nil {
				return err
			}

			// APPLIED releases the reservation by leaving PENDING, which is what
			// drops the row out of idx_pc_name_reservation.
			_, _, err = p.WithTransaction(tx).Resolve(buf)(m.Id(), StatusApplied, "")
			return err
		})
	})
}

// startWorldTransfer dispatches the transfer saga and leaves the record PENDING
// on purpose: the saga's terminal event is what drives Resolve, so a failure
// mid-saga still lands on the refund path through the same transition guard.
func (p *ProcessorImpl) startWorldTransfer(m Model) error {
	if p.worldTransferStarter == nil {
		return fmt.Errorf("no world-transfer saga dispatcher wired; pending change [%s] for character [%d] cannot be applied", m.Id(), m.CharacterId())
	}
	return database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
		return message.Emit(outbox.EmitProvider(p.l, p.ctx, tx))(func(buf *message.Buffer) error {
			return p.worldTransferStarter(p.l, p.ctx, buf, m)
		})
	})
}

// RenotifyForCharacter drains the resolved-but-unnotified backlog at LOGIN
// (FR-2.9). Every re-emission is routed on the character's CURRENT world, read
// now — NOT on the record's SourceWorldId. For an APPLIED world transfer the
// character has already moved to the destination by the time they log back in,
// so routing on the source would announce the successful transfer to the world
// they just left, which is to say never announce it at all.
func (p *ProcessorImpl) RenotifyForCharacter(characterId uint32) error {
	ms, err := getResolvedUnnotified(p.db.WithContext(p.ctx), p.t.Id(), characterId)
	if err != nil {
		return err
	}
	if len(ms) == 0 {
		return nil
	}

	c, err := character.NewProcessor(p.l, p.ctx, p.db).GetById()(characterId)
	if err != nil {
		return err
	}
	currentWorldId := c.WorldId()

	now := time.Now()
	return database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
		return message.Emit(outbox.EmitProvider(p.l, p.ctx, tx))(func(buf *message.Buffer) error {
			for _, m := range ms {
				// The mark-then-emit order (reversed from a naive emit-then-mark)
				// is the fix: getResolvedUnnotified above is a plain SELECT outside
				// any transaction, so two concurrent LOGIN deliveries can both see
				// this row unnotified. Only the delivery whose UPDATE actually moves
				// the row (moved == true) may emit; the loser's UPDATE is a no-op and
				// it must not mint a second notification.
				moved, err := markNotified(tx, p.t.Id(), m.Id(), now)
				if err != nil {
					return err
				}
				if !moved {
					continue
				}
				if err := buf.Put(pendingchange2.EnvEventTopic, resolvedEventProviderForWorld(m, currentWorldId)); err != nil {
					return err
				}
				p.l.Infof("Re-emitted resolution of pending change [%s] to character [%d] on world [%d].", m.Id(), characterId, currentWorldId)
			}
			return nil
		})
	})
}

// Sweep expires every request whose deadline has passed. Each row goes through
// ResolveAndEmit, so the refund is minted by the same transition guard the
// operator cancel uses — a sweep that runs twice refunds once.
func (p *ProcessorImpl) Sweep(now time.Time) error {
	ms, err := getExpired(p.db.WithContext(p.ctx), p.t.Id(), now)
	if err != nil {
		return err
	}
	for _, m := range ms {
		if _, _, err := p.ResolveAndEmit(m.Id(), StatusExpired, "expired"); err != nil {
			return err
		}
	}
	return nil
}

func (p *ProcessorImpl) GetByCharacterId(characterId uint32) ([]Model, error) {
	return getByCharacterId(p.db.WithContext(p.ctx), p.t.Id(), characterId)
}

func (p *ProcessorImpl) GetById(id uuid.UUID) (Model, error) {
	return getById(p.db.WithContext(p.ctx), p.t.Id(), id)
}

// NameReserved reports whether a live pending name change holds this name
// (FR-3.3). The lookup is on the stored lower-cased column, so it agrees
// exactly with idx_pc_name_reservation.
func (p *ProcessorImpl) NameReserved(name string) (bool, error) {
	_, err := getPendingByNameLower(p.db.WithContext(p.ctx), p.t.Id(), strings.ToLower(name))
	if errors.Is(err, ErrNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// evaluateTransferEligibility applies the design §3.6 / §1.6 gate table in the
// documented order — cheapest and most local first, so an obviously-invalid
// request never fans out. The gates implemented here are the three that need no
// network call; the remaining eight (destination world capacity, character
// slots, bans, guild mastery, family, open trade, hired merchant, live MTS
// listings) each need a narrow REST client and are added in the same evaluation
// order behind gate 5.
func (p *ProcessorImpl) evaluateTransferEligibility(c character.Model, destinationWorldId world.Id) (string, bool) {
	// Gate 1: a transfer to the world you are already in is not a transfer.
	if destinationWorldId == c.WorldId() {
		return "world_same", false
	}
	// Gate 2: the v83 client's own CCashShop::CheckTransferWorldPossible
	// refuses to send the request for a GM, so a server that permits it
	// produces a state the client considers impossible.
	if c.GM() != 0 {
		return "is_gm", false
	}
	// Gate 5: the name must be free in the destination. Name uniqueness for a
	// pending change is already tenant-wide (FR-3.2), so a tenant-scoped check
	// subsumes the per-world one and cannot under-report.
	cs, err := character.NewProcessor(p.l, p.ctx, p.db).GetForName()(c.Name())
	if err != nil {
		p.l.WithError(err).Errorf("Unable to check name availability for character [%d] transferring to world [%d].", c.Id(), destinationWorldId)
		return "name_taken", false
	}
	for _, other := range cs {
		if other.Id() != c.Id() {
			return "name_taken", false
		}
	}

	return "", true
}
