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
	CreateListing(transactionId uuid.UUID, worldId world.Id, sellerId uint32, sellerAccountId uint32, sellerName string, saleType string, sourceInventoryType byte, assetId uint32, quantity uint32, listValue uint32, buyNowPrice *uint32, durationHours int, minIncrement uint32, category string, subCategory string, offerWishSerial uint32, offerWishOwnerId uint32) error
	Buy(transactionId uuid.UUID, worldId world.Id, serial uint32, buyerId uint32, buyerAccountId uint32, buyNow bool) error
	PlaceBid(transactionId uuid.UUID, worldId world.Id, serial uint32, bidderId uint32, bidderAccountId uint32, amount uint32) error
	CancelListing(transactionId uuid.UUID, worldId world.Id, serial uint32, sellerId uint32) error
	TakeHome(transactionId uuid.UUID, worldId world.Id, serial uint32, characterId uint32, inventoryType byte, slot int16) error
	RegisterWish(worldId world.Id, characterId uint32, itemId uint32, price uint32, origin string) error
	RemoveWish(worldId world.Id, wishId uuid.UUID, characterId uint32, origin string) error
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{l: l, ctx: ctx}
}

func (p *ProcessorImpl) CreateListing(transactionId uuid.UUID, worldId world.Id, sellerId uint32, sellerAccountId uint32, sellerName string, saleType string, sourceInventoryType byte, assetId uint32, quantity uint32, listValue uint32, buyNowPrice *uint32, durationHours int, minIncrement uint32, category string, subCategory string, offerWishSerial uint32, offerWishOwnerId uint32) error {
	p.l.Debugf("Character [%d] creating MTS listing (saleType [%s], asset [%d]).", sellerId, saleType, assetId)
	return producer.ProviderImpl(p.l)(p.ctx)(mtsmsg.EnvCommandTopic)(CreateListingCommandProvider(transactionId, worldId, sellerId, sellerAccountId, sellerName, saleType, sourceInventoryType, assetId, quantity, listValue, buyNowPrice, durationHours, minIncrement, category, subCategory, offerWishSerial, offerWishOwnerId))
}

func (p *ProcessorImpl) Buy(transactionId uuid.UUID, worldId world.Id, serial uint32, buyerId uint32, buyerAccountId uint32, buyNow bool) error {
	p.l.Debugf("Character [%d] buying MTS listing serial [%d] (buyNow [%t]).", buyerId, serial, buyNow)
	return producer.ProviderImpl(p.l)(p.ctx)(mtsmsg.EnvCommandTopic)(BuyCommandProvider(transactionId, worldId, serial, buyerId, buyerAccountId, buyNow))
}

func (p *ProcessorImpl) PlaceBid(transactionId uuid.UUID, worldId world.Id, serial uint32, bidderId uint32, bidderAccountId uint32, amount uint32) error {
	p.l.Debugf("Character [%d] placing bid [%d] on MTS listing serial [%d].", bidderId, amount, serial)
	return producer.ProviderImpl(p.l)(p.ctx)(mtsmsg.EnvCommandTopic)(PlaceBidCommandProvider(transactionId, worldId, serial, bidderId, bidderAccountId, amount))
}

func (p *ProcessorImpl) CancelListing(transactionId uuid.UUID, worldId world.Id, serial uint32, sellerId uint32) error {
	p.l.Debugf("Character [%d] cancelling MTS listing serial [%d].", sellerId, serial)
	return producer.ProviderImpl(p.l)(p.ctx)(mtsmsg.EnvCommandTopic)(CancelListingCommandProvider(transactionId, worldId, serial, sellerId))
}

func (p *ProcessorImpl) TakeHome(transactionId uuid.UUID, worldId world.Id, serial uint32, characterId uint32, inventoryType byte, slot int16) error {
	p.l.Debugf("Character [%d] taking home MTS holding serial [%d].", characterId, serial)
	return producer.ProviderImpl(p.l)(p.ctx)(mtsmsg.EnvCommandTopic)(TakeHomeCommandProvider(transactionId, worldId, serial, characterId, inventoryType, slot))
}

func (p *ProcessorImpl) RegisterWish(worldId world.Id, characterId uint32, itemId uint32, price uint32, origin string) error {
	p.l.Debugf("Character [%d] registering MTS wish for item [%d] (price [%d], origin [%s]).", characterId, itemId, price, origin)
	return producer.ProviderImpl(p.l)(p.ctx)(mtsmsg.EnvCommandTopic)(RegisterWishCommandProvider(worldId, characterId, itemId, price, origin))
}

func (p *ProcessorImpl) RemoveWish(worldId world.Id, wishId uuid.UUID, characterId uint32, origin string) error {
	p.l.Debugf("Character [%d] removing MTS wish [%s] (origin [%s]).", characterId, wishId.String(), origin)
	return producer.ProviderImpl(p.l)(p.ctx)(mtsmsg.EnvCommandTopic)(RemoveWishCommandProvider(worldId, wishId, characterId, origin))
}
