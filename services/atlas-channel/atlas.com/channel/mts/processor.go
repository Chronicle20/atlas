package mts

import (
	mtsmsg "atlas-channel/kafka/message/mts"
	"atlas-channel/kafka/producer"
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// Processor emits COMMAND_TOPIC_MTS commands to atlas-mts. The per-arm
// ITC_OPERATION handlers (sibling tasks) call these to drive create/buy/bid/
// cancel/take-home/wish operations; the channel never talks to the atlas-mts
// REST surface for writes — it commands via Kafka, mirroring the messenger and
// cashshop command processors.
type Processor interface {
	CreateListing(transactionId uuid.UUID, worldId world.Id, sellerId uint32, sellerAccountId uint32, itemId uint32, quantity uint32, price uint32, isAuction bool, buyNowPrice uint32, durationHours uint32) error
	Buy(transactionId uuid.UUID, worldId world.Id, listingId uuid.UUID, buyerId uint32, buyerAccountId uint32, sellerAccountId uint32) error
	PlaceBid(transactionId uuid.UUID, worldId world.Id, listingId uuid.UUID, bidderId uint32, bidderAccountId uint32, amount uint32) error
	CancelListing(transactionId uuid.UUID, worldId world.Id, listingId uuid.UUID, sellerId uint32) error
	TakeHome(transactionId uuid.UUID, worldId world.Id, holdingId uuid.UUID, characterId uint32) error
	RegisterWish(worldId world.Id, characterId uint32, itemId uint32) error
	RemoveWish(worldId world.Id, wishId uuid.UUID, characterId uint32) error
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{l: l, ctx: ctx}
}

func (p *ProcessorImpl) CreateListing(transactionId uuid.UUID, worldId world.Id, sellerId uint32, sellerAccountId uint32, itemId uint32, quantity uint32, price uint32, isAuction bool, buyNowPrice uint32, durationHours uint32) error {
	p.l.Debugf("Character [%d] creating MTS listing for item [%d] (auction [%t]).", sellerId, itemId, isAuction)
	return producer.ProviderImpl(p.l)(p.ctx)(mtsmsg.EnvCommandTopic)(CreateListingCommandProvider(transactionId, worldId, sellerId, sellerAccountId, itemId, quantity, price, isAuction, buyNowPrice, durationHours))
}

func (p *ProcessorImpl) Buy(transactionId uuid.UUID, worldId world.Id, listingId uuid.UUID, buyerId uint32, buyerAccountId uint32, sellerAccountId uint32) error {
	p.l.Debugf("Character [%d] buying MTS listing [%s].", buyerId, listingId.String())
	return producer.ProviderImpl(p.l)(p.ctx)(mtsmsg.EnvCommandTopic)(BuyCommandProvider(transactionId, worldId, listingId, buyerId, buyerAccountId, sellerAccountId))
}

func (p *ProcessorImpl) PlaceBid(transactionId uuid.UUID, worldId world.Id, listingId uuid.UUID, bidderId uint32, bidderAccountId uint32, amount uint32) error {
	p.l.Debugf("Character [%d] placing bid [%d] on MTS listing [%s].", bidderId, amount, listingId.String())
	return producer.ProviderImpl(p.l)(p.ctx)(mtsmsg.EnvCommandTopic)(PlaceBidCommandProvider(transactionId, worldId, listingId, bidderId, bidderAccountId, amount))
}

func (p *ProcessorImpl) CancelListing(transactionId uuid.UUID, worldId world.Id, listingId uuid.UUID, sellerId uint32) error {
	p.l.Debugf("Character [%d] cancelling MTS listing [%s].", sellerId, listingId.String())
	return producer.ProviderImpl(p.l)(p.ctx)(mtsmsg.EnvCommandTopic)(CancelListingCommandProvider(transactionId, worldId, listingId, sellerId))
}

func (p *ProcessorImpl) TakeHome(transactionId uuid.UUID, worldId world.Id, holdingId uuid.UUID, characterId uint32) error {
	p.l.Debugf("Character [%d] taking home MTS holding [%s].", characterId, holdingId.String())
	return producer.ProviderImpl(p.l)(p.ctx)(mtsmsg.EnvCommandTopic)(TakeHomeCommandProvider(transactionId, worldId, holdingId, characterId))
}

func (p *ProcessorImpl) RegisterWish(worldId world.Id, characterId uint32, itemId uint32) error {
	p.l.Debugf("Character [%d] registering MTS wish for item [%d].", characterId, itemId)
	return producer.ProviderImpl(p.l)(p.ctx)(mtsmsg.EnvCommandTopic)(RegisterWishCommandProvider(worldId, characterId, itemId))
}

func (p *ProcessorImpl) RemoveWish(worldId world.Id, wishId uuid.UUID, characterId uint32) error {
	p.l.Debugf("Character [%d] removing MTS wish [%s].", characterId, wishId.String())
	return producer.ProviderImpl(p.l)(p.ctx)(mtsmsg.EnvCommandTopic)(RemoveWishCommandProvider(worldId, wishId, characterId))
}
