// Package surprise implements the Cash Shop Surprise open orchestration
// (task-207). Open resolves and validates the box, checks locker capacity,
// rolls a reward from the configured pool, resolves the reward's commodity,
// then commits exactly one database transaction that inserts the
// idempotency ledger row, consumes the box, and grants the reward asset.
//
// Ordering is the correctness argument for having no saga: the roll (step 3)
// mutates nothing, so ordering roll -> (consume + grant) makes partial
// application structurally impossible. Steps 1-4 (resolve, capacity check,
// roll, commodity resolve) touch no persisted state, so a rejection there
// can be reported immediately, on the direct producer path, with nothing to
// roll back. Only step 5 (the ledger insert, the box consume, and the
// reward grant) is transactional.
package surprise

import (
	"atlas-cashshop/cashshop/commodity"
	"atlas-cashshop/cashshop/inventory/asset"
	"atlas-cashshop/cashshop/inventory/compartment"
	"atlas-cashshop/character"
	"atlas-cashshop/configuration"
	"atlas-cashshop/kafka/message"
	"atlas-cashshop/kafka/message/cashshop"
	cashshop2 "atlas-cashshop/kafka/producer/cashshop"
	"atlas-cashshop/rewardpool"
	"atlas-cashshop/surprise/opening"
	"context"
	"errors"
	"slices"

	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	outbox "github.com/Chronicle20/atlas/libs/atlas-outbox"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

type Processor interface {
	OpenAndEmit(transactionId uuid.UUID, accountId uint32, characterId uint32, cashId int64) error
	Open(mb *message.Buffer) func(transactionId uuid.UUID, accountId uint32, characterId uint32, cashId int64) error
}

type ProcessorImpl struct {
	l    logrus.FieldLogger
	ctx  context.Context
	db   *gorm.DB
	t    tenant.Model
	chaP character.Processor
	cicP compartment.Processor
	comP commodity.Processor
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) Processor {
	return &ProcessorImpl{
		l:    l,
		ctx:  ctx,
		db:   db,
		t:    tenant.MustFromContext(ctx),
		chaP: character.NewProcessor(l, ctx),
		cicP: compartment.NewProcessor(l, ctx, db),
		comP: commodity.NewProcessor(l, ctx),
	}
}

var _ Processor = (*ProcessorImpl)(nil)

func (p *ProcessorImpl) OpenAndEmit(transactionId uuid.UUID, accountId uint32, characterId uint32, cashId int64) error {
	return database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
		return message.Emit(outbox.EmitProvider(p.l, p.ctx, tx))(func(buf *message.Buffer) error {
			return NewProcessor(p.l, p.ctx, tx).Open(buf)(transactionId, accountId, characterId, cashId)
		})
	})
}

// Open resolves and validates a Cash Shop Surprise open, then commits the
// single transaction described at package level. Rejections found during
// resolve/capacity/roll/commodity-resolve (nothing mutated yet) fire
// SURPRISE_FAILED on the DIRECT producer path and are swallowed (nil
// return) -- retrying the identical command would fail identically, and the
// client has already been told. A genuine failure inside the transaction
// (e.g. the reward asset write itself failing) is propagated as a real
// error with no event fired, mirroring cashshop.Purchase's precedent: an
// unexpected infrastructure fault is not a business rejection.
func (p *ProcessorImpl) Open(mb *message.Buffer) func(transactionId uuid.UUID, accountId uint32, characterId uint32, cashId int64) error {
	return func(transactionId uuid.UUID, accountId uint32, characterId uint32, cashId int64) error {
		log := p.l.WithFields(logrus.Fields{
			"tenant":         p.t.Id().String(),
			"account_id":     accountId,
			"character_id":   characterId,
			"transaction_id": transactionId.String(),
			"box_cash_id":    cashId,
		})

		// reject fires SURPRISE_FAILED directly (steps 1-4 mutate nothing, so
		// there is no committed state to reconcile) and swallows the error --
		// the client has already been told via the event.
		reject := func(reason string, cause error, fields logrus.Fields) error {
			entry := log.WithField("outcome", reason)
			if fields != nil {
				entry = entry.WithFields(fields)
			}
			if cause != nil {
				entry = entry.WithError(cause)
			}
			entry.Warnf("Surprise open rejected: %s.", reason)
			if err := producer.ProviderImpl(p.l)(p.ctx)(cashshop.EnvEventTopicStatus)(cashshop2.SurpriseFailedStatusEventProvider(characterId, reason)); err != nil {
				log.WithError(err).Errorf("Unable to emit SURPRISE_FAILED [%s] on the direct producer path.", reason)
			}
			return nil
		}

		// Step 1: resolve the character's job-typed compartment, then the box
		// asset within it. Ownership (FR-2.1) is enforced structurally: the
		// compartment is looked up by accountId, so an asset belonging to
		// another account is simply not in the scanned set -- BOX_NOT_FOUND
		// and NOT_OWNED collapse to the same observable outcome.
		c, err := p.chaP.GetById()(characterId)
		if err != nil {
			return reject("INTERNAL", err, logrus.Fields{"stage": "resolve_character"})
		}

		var compartmentType compartment.CompartmentType
		switch {
		case job.GetType(c.JobId()) == job.TypeExplorer:
			compartmentType = compartment.TypeExplorer
		case job.GetType(c.JobId()) == job.TypeCygnus:
			compartmentType = compartment.TypeCygnus
		default:
			compartmentType = compartment.TypeLegend
		}

		ccm, err := p.cicP.GetByAccountIdAndType(accountId, compartmentType)
		if err != nil {
			return reject("INTERNAL", err, logrus.Fields{"stage": "resolve_compartment"})
		}

		var box *asset.Model
		for i := range ccm.Assets() {
			if ccm.Assets()[i].CashId() == cashId {
				box = &ccm.Assets()[i]
				break
			}
		}
		if box == nil {
			return reject("BOX_NOT_FOUND", nil, nil)
		}
		log = log.WithFields(logrus.Fields{"box_asset_id": box.Id(), "box_template_id": box.TemplateId(), "pool_id": box.TemplateId()})

		surpriseBoxTemplateIds := configuration.GetSurpriseBoxTemplateIds(p.l, p.ctx, p.t.Id())
		if !slices.Contains(surpriseBoxTemplateIds, box.TemplateId()) {
			return reject("NOT_A_SURPRISE_BOX", nil, nil)
		}

		// Step 2: capacity.
		if !HasRoomForSwap(uint32(len(ccm.Assets())), ccm.Capacity(), box.Quantity()) {
			return reject("LOCKER_FULL", nil, nil)
		}

		// Step 3: roll. Mutates nothing -- this is what makes the ordering
		// (roll, THEN consume+grant) the correctness argument for skipping a
		// saga: a failed roll can never leave a half-consumed box.
		reward, err := rewardpool.NewProcessor(p.l, p.ctx).SelectReward(box.TemplateId())
		if err != nil {
			switch {
			case errors.Is(err, rewardpool.ErrPoolMissing):
				return reject("POOL_MISSING", err, nil)
			case errors.Is(err, rewardpool.ErrPoolEmpty):
				return reject("POOL_EMPTY", err, nil)
			default:
				return reject("INTERNAL", err, logrus.Fields{"stage": "select_reward"})
			}
		}

		// FR-4.5: a pool configured to award a Surprise box creates an
		// infinite box. Honoured by configuration, not blocked in code --
		// just surfaced loudly so an operator notices.
		if slices.Contains(surpriseBoxTemplateIds, reward.ItemId()) {
			log.WithField("reward_template_id", reward.ItemId()).Warnf("Reward pool for box template [%d] rolled another configured Surprise box -- this recurses.", box.TemplateId())
		}

		// Step 4: resolve the commodity. A zero commodityId means the reward
		// pool entry carries no price/period basis to derive an expiration
		// from -- granting it anyway risks an asset with a wrong or absent
		// expiration, so it is treated the same as a commodity lookup miss
		// (COMMODITY_MISSING) rather than silently flowing into asset
		// creation with commodityId 0 (which asset.Create would otherwise
		// accept, defaulting to a 30-day expiration that no operator asked
		// for).
		if reward.CommodityId() == 0 {
			return reject("COMMODITY_MISSING", nil, logrus.Fields{"reward_template_id": reward.ItemId()})
		}
		ci, err := commodity.NewProcessor(p.l, p.ctx).GetById(reward.CommodityId())
		if err != nil {
			return reject("COMMODITY_MISSING", err, logrus.Fields{"commodity_id": reward.CommodityId()})
		}
		log = log.WithFields(logrus.Fields{"commodity_id": ci.Id(), "reward_template_id": ci.ItemId()})

		// Step 5: the one transaction. opening.Insert MUST be first so a
		// duplicate transactionId aborts before anything is consumed or
		// granted (FR-4.4).
		var remaining uint32
		txErr := database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
			insertErr := opening.Insert(tx, p.t.Id(), transactionId, accountId, box.Id())
			if errors.Is(insertErr, opening.ErrAlreadyOpened) {
				// Success-without-effect: a Kafka redelivery of an
				// already-committed open, not a new click. No state
				// changes, no event -- the original open already told the
				// client.
				log.Infof("Surprise open replayed for an already-committed transaction; granting nothing further.")
				return nil
			}
			if insertErr != nil {
				return insertErr
			}

			// astP.UpdateQuantity and astP.Release both run on whatever db
			// the processor was built with -- rebuild against tx so these
			// writes land INSIDE this transaction rather than escaping to
			// p.db (task-207 FR-4.1; see cashshop.Purchase for the same
			// precedent).
			astP := asset.NewProcessor(p.l, p.ctx, tx)

			remaining = box.Quantity() - 1
			if remaining == 0 {
				if err := astP.Release(mb)(box.Id()); err != nil {
					return err
				}
			} else {
				if err := astP.UpdateQuantity(box.Id(), remaining); err != nil {
					return err
				}
			}

			rewardAsset, err := astP.Create(mb)(ccm.Id(), ci.ItemId(), reward.CommodityId(), ci.Count(), 0, characterId)
			if err != nil {
				return err
			}

			log.WithFields(logrus.Fields{"outcome": "OPENED", "box_remaining": remaining, "reward_asset_id": rewardAsset.Id(), "reward_count": ci.Count()}).Infof("Surprise box opened.")
			return mb.Put(cashshop.EnvEventTopicStatus, cashshop2.SurpriseOpenedStatusEventProvider(characterId, ccm.Id(), cashId, remaining, rewardAsset.Id(), ci.ItemId(), ci.Count()))
		})
		if txErr != nil {
			log.WithError(txErr).WithField("outcome", "INTERNAL").Errorf("Unable to open surprise box.")
			return txErr
		}
		return nil
	}
}
