package merchant

import (
	merchant2 "atlas-channel/kafka/message/merchant"
	"atlas-channel/kafka/producer"
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

type Processor struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) *Processor {
	return &Processor{l: l, ctx: ctx}
}

func (p *Processor) InFieldModelProvider(f field.Model) model.Provider[[]Model] {
	return requests.SliceProvider[RestModel, Model](p.l, p.ctx)(requestInField(f), Extract, model.Filters[Model]())
}

func (p *Processor) ForEachInField(f field.Model, o model.Operator[Model]) error {
	return model.ForEachSlice(p.InFieldModelProvider(f), o, model.ParallelExecute())
}

func (p *Processor) GetVisitingShop(characterId uint32) (Model, error) {
	return requests.Provider[RestModel, Model](p.l, p.ctx)(requestVisiting(characterId), Extract)()
}

func (p *Processor) GetShop(shopId string) (Model, error) {
	return requests.Provider[RestModel, Model](p.l, p.ctx)(requestShop(shopId), Extract)()
}

func (p *Processor) GetByCharacterId(characterId uint32) ([]Model, error) {
	return requests.SliceProvider[RestModel, Model](p.l, p.ctx)(requestByCharacterId(characterId), Extract, model.Filters[Model]())()
}

func (p *Processor) PlaceShop(f field.Model, characterId uint32, shopType byte, title string, permitItemId uint32, x int16, y int16) error {
	return producer.ProviderImpl(p.l)(p.ctx)(merchant2.EnvCommandTopic)(PlaceShopCommandProvider(f, characterId, shopType, title, permitItemId, x, y))
}

func (p *Processor) OpenShop(characterId uint32, shopId uuid.UUID) error {
	return producer.ProviderImpl(p.l)(p.ctx)(merchant2.EnvCommandTopic)(OpenShopCommandProvider(characterId, shopId))
}

func (p *Processor) CloseShop(characterId uint32, shopId uuid.UUID) error {
	return producer.ProviderImpl(p.l)(p.ctx)(merchant2.EnvCommandTopic)(CloseShopCommandProvider(characterId, shopId))
}

func (p *Processor) EnterShop(characterId uint32, shopId uuid.UUID) error {
	return producer.ProviderImpl(p.l)(p.ctx)(merchant2.EnvCommandTopic)(EnterShopCommandProvider(characterId, shopId))
}

func (p *Processor) ExitShop(characterId uint32, shopId uuid.UUID) error {
	return producer.ProviderImpl(p.l)(p.ctx)(merchant2.EnvCommandTopic)(ExitShopCommandProvider(characterId, shopId))
}

func (p *Processor) SendMessage(characterId uint32, shopId uuid.UUID, content string) error {
	return producer.ProviderImpl(p.l)(p.ctx)(merchant2.EnvCommandTopic)(SendMessageCommandProvider(characterId, shopId, content))
}

func (p *Processor) EnterMaintenance(characterId uint32, shopId uuid.UUID) error {
	return producer.ProviderImpl(p.l)(p.ctx)(merchant2.EnvCommandTopic)(EnterMaintenanceCommandProvider(characterId, shopId))
}

func (p *Processor) ExitMaintenance(characterId uint32, shopId uuid.UUID) error {
	return producer.ProviderImpl(p.l)(p.ctx)(merchant2.EnvCommandTopic)(ExitMaintenanceCommandProvider(characterId, shopId))
}

func (p *Processor) AddListing(characterId uint32, shopId uuid.UUID, inventoryType byte, slot int16, quantity uint16, bundleSize uint16, pricePerBundle uint32) error {
	return producer.ProviderImpl(p.l)(p.ctx)(merchant2.EnvCommandTopic)(AddListingCommandProvider(characterId, shopId, inventoryType, slot, quantity, bundleSize, pricePerBundle))
}

func (p *Processor) RemoveListing(characterId uint32, shopId uuid.UUID, listingIndex uint16) error {
	return producer.ProviderImpl(p.l)(p.ctx)(merchant2.EnvCommandTopic)(RemoveListingCommandProvider(characterId, shopId, listingIndex))
}

func (p *Processor) PurchaseBundle(characterId uint32, shopId uuid.UUID, listingIndex uint16, bundleCount uint16) error {
	return producer.ProviderImpl(p.l)(p.ctx)(merchant2.EnvCommandTopic)(PurchaseBundleCommandProvider(characterId, shopId, listingIndex, bundleCount))
}
