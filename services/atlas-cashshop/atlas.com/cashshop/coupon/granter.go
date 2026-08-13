package coupon

import (
	"atlas-cashshop/cashshop/commodity"
	"atlas-cashshop/cashshop/inventory/asset"
	"atlas-cashshop/cashshop/inventory/compartment"
	"atlas-cashshop/coupon/reward"
	"atlas-cashshop/kafka/message"
	"atlas-cashshop/wallet"
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
)

// redemptionContext is what a granter needs about the redeeming player,
// resolved once by the processor before the reward loop.
type redemptionContext struct {
	accountId     uint32
	characterId   uint32
	compartmentId uuid.UUID
}

// grantedReward is one granter's contribution to the success event. The
// currency fields are DELTAS — the amounts this coupon awarded — because
// UseCouponDone renders maplePoint inside "You have received ... using the
// coupon" and skips it when zero (derivation.md, "Blocking answer 1").
type grantedReward struct {
	assetId     uint32
	maplePoints uint32
	credit      uint32
}

// rewardGranter applies one reward INSIDE the redemption transaction. Every
// implementation today writes only to atlas-cashshop's own tables, which is
// exactly why redemption is a single local transaction rather than a saga
// (design §2). When a reward type owned by ANOTHER service is added, that
// granter is the single place a saga gets introduced.
type rewardGranter interface {
	Grant(mb *message.Buffer) func(tx *gorm.DB, rc redemptionContext, r reward.Reward) (grantedReward, error)
}

func granterFor(l logrus.FieldLogger, ctx context.Context, r reward.Reward) (rewardGranter, error) {
	switch r.Type() {
	case reward.TypeCurrency:
		return currencyGranter{l: l, ctx: ctx}, nil
	case reward.TypeCashItem:
		return cashItemGranter{l: l, ctx: ctx, cp: commodity.NewProcessor(l, ctx)}, nil
	default:
		// Never silently skip: a coupon that claims to grant something and
		// grants nothing is worse than one that fails loudly.
		return nil, fmt.Errorf("%w: no granter for reward type %q", reward.ErrInvalid, r.Type())
	}
}

type currencyGranter struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func (g currencyGranter) Grant(mb *message.Buffer) func(tx *gorm.DB, rc redemptionContext, r reward.Reward) (grantedReward, error) {
	return func(tx *gorm.DB, rc redemptionContext, r reward.Reward) (grantedReward, error) {
		wp := wallet.NewProcessor(g.l, g.ctx, tx).WithTransaction(tx)
		w, err := wp.GetByAccountId(rc.accountId)
		if err != nil {
			return grantedReward{}, err
		}
		w = w.Award(r.Currency(), r.Amount())
		if _, err = wp.Update(mb)(rc.accountId)(w.Credit())(w.Points())(w.Prepaid()); err != nil {
			return grantedReward{}, err
		}
		out := grantedReward{}
		// Currency ids follow wallet.Model.Balance: 1 = credit (NX),
		// 2 = Maple Points, anything else = prepaid.
		switch r.Currency() {
		case 1:
			out.credit = r.Amount()
		case 2:
			out.maplePoints = r.Amount()
		}
		// Prepaid has no field in UseCouponDone; it is credited to the wallet
		// and shows up on the next CashQueryResult, which is what the client
		// refreshes balances from anyway.
		return out, nil
	}
}

type cashItemGranter struct {
	l   logrus.FieldLogger
	ctx context.Context
	cp  commodity.Processor
}

func (g cashItemGranter) Grant(mb *message.Buffer) func(tx *gorm.DB, rc redemptionContext, r reward.Reward) (grantedReward, error) {
	return func(tx *gorm.DB, rc redemptionContext, r reward.Reward) (grantedReward, error) {
		// Re-read capacity INSIDE the transaction, and before any remote
		// lookup, so a full locker is rejected on this handle's view of the
		// rows rather than on the processor's earlier pre-flight read.
		cp := compartment.NewProcessor(g.l, g.ctx, tx).WithTransaction(tx)
		ccm, err := cp.GetById(rc.compartmentId)
		if err != nil {
			return grantedReward{}, err
		}
		if ccm.Capacity() <= uint32(len(ccm.Assets())) {
			return grantedReward{}, NewRedemptionError(ErrorKeyInventoryFull, "cash locker filled up during redemption")
		}

		// A coupon names its reward by serialNumber — the COMMODITY id. The
		// locker row needs the item id too: asset.Create stores templateId
		// verbatim and consults commodityId only for the expiration period, so
		// a zero templateId would persist an asset the client cannot render.
		ci, err := g.cp.GetById(r.SerialNumber())
		if err != nil {
			g.l.WithError(err).Errorf("Unable to resolve commodity [%d] for a coupon reward.", r.SerialNumber())
			return grantedReward{}, err
		}

		// Pets need a pet row created against the SAME cash serial as the
		// locker asset (see cashshop.Processor.Purchase). This granter creates
		// only the asset, so granting a pet here would produce a locker item
		// with no pet behind it. Refuse loudly rather than write that row.
		if item.GetClassification(item.Id(ci.ItemId())) == item.ClassificationPet {
			return grantedReward{}, fmt.Errorf("%w: commodity %d is a pet; coupons cannot grant pets", reward.ErrInvalid, r.SerialNumber())
		}

		am, err := asset.NewProcessor(g.l, g.ctx, tx).
			Create(mb)(rc.compartmentId, ci.ItemId(), r.SerialNumber(), r.Quantity(), 0, rc.characterId)
		if err != nil {
			return grantedReward{}, err
		}
		return grantedReward{assetId: am.Id()}, nil
	}
}
