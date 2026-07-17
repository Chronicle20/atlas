package merchant

import (
	"atlas-channel/character"
	merchant2 "atlas-channel/kafka/message/merchant"
	"context"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	inventory2 "github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

type Processor interface {
	InFieldModelProvider(f field.Model) model.Provider[[]Model]
	ForEachInField(f field.Model, o model.Operator[Model]) error
	GetVisitingShop(characterId uint32) (Model, error)
	GetShop(shopId string) (Model, error)
	GetByCharacterId(characterId uint32) ([]Model, error)
	HasFrederickPending(characterId uint32) (bool, error)
	PlaceShop(f field.Model, characterId uint32, shopType byte, title string, permitItemId uint32, x int16, y int16) error
	OpenShop(characterId uint32, shopId uuid.UUID) error
	CloseShop(characterId uint32, shopId uuid.UUID) error
	EnterShop(characterId uint32, shopId uuid.UUID, visitorName string) error
	AddBlacklist(characterId uint32, shopId uuid.UUID, name string, bannedCharacterId uint32) error
	RemoveBlacklist(characterId uint32, shopId uuid.UUID, name string) error
	GetBlacklist(shopId string) ([]string, error)
	GetVisits(shopId string) ([]VisitEntry, error)
	ExitShop(characterId uint32, shopId uuid.UUID) error
	SendMessage(characterId uint32, shopId uuid.UUID, content string) error
	EnterMaintenance(characterId uint32, shopId uuid.UUID) error
	ExitMaintenance(characterId uint32, shopId uuid.UUID) error
	WithdrawMeso(characterId uint32, shopId uuid.UUID) error
	OrganizeListings(characterId uint32, shopId uuid.UUID) error
	AddListing(characterId uint32, shopId uuid.UUID, inventoryType byte, slot int16, quantity uint16, bundleSize uint16, pricePerBundle uint32) error
	RemoveListing(characterId uint32, shopId uuid.UUID, listingIndex uint16) error
	PurchaseBundle(characterId uint32, shopId uuid.UUID, listingIndex uint16, bundleCount uint16) error
	SearchListings(worldId world.Id, itemId uint32, descending bool) ([]SearchListing, error)
	GetTopSearches(worldId world.Id) ([]TopSearch, error)
	RecordItemSearch(f field.Model, characterId uint32, itemId uint32) error
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{l: l, ctx: ctx}
}

var _ Processor = (*ProcessorImpl)(nil)

// InFieldModelProvider fetches the complete set of shops on a field.
// atlas-merchant's field-scoped list is now paginated (task-117); callers
// here (session spawn on map load) need the whole set, so this drains every
// page rather than fetching one.
func (p *ProcessorImpl) InFieldModelProvider(f field.Model) model.Provider[[]Model] {
	return requests.DrainProvider[RestModel, Model](p.l, p.ctx)(inFieldUrl(f), 250, Extract, model.Filters[Model]())
}

func (p *ProcessorImpl) ForEachInField(f field.Model, o model.Operator[Model]) error {
	return model.ForEachSlice(p.InFieldModelProvider(f), o, model.ParallelExecute())
}

func (p *ProcessorImpl) GetVisitingShop(characterId uint32) (Model, error) {
	return requests.Provider[RestModel, Model](p.l, p.ctx)(requestVisiting(characterId), Extract)()
}

func (p *ProcessorImpl) GetShop(shopId string) (Model, error) {
	return requests.Provider[RestModel, Model](p.l, p.ctx)(requestShop(shopId), Extract)()
}

// GetByCharacterId fetches the complete set of shops for a character.
// atlas-merchant's per-character list is now paginated (task-117); callers
// here (visit lookup, hired-merchant permit check) need the whole set (a
// character has at most one active shop per ShopType, so it is naturally
// small), so this drains every page rather than fetching one.
func (p *ProcessorImpl) GetByCharacterId(characterId uint32) ([]Model, error) {
	return requests.DrainProvider[RestModel, Model](p.l, p.ctx)(byCharacterIdUrl(characterId), 250, Extract, model.Filters[Model]())()
}

func (p *ProcessorImpl) HasFrederickPending(characterId uint32) (bool, error) {
	rm, err := requestFrederickStatus(characterId)(p.l, p.ctx)
	if err != nil {
		return false, err
	}
	return rm.HasPending, nil
}

func (p *ProcessorImpl) PlaceShop(f field.Model, characterId uint32, shopType byte, title string, permitItemId uint32, x int16, y int16) error {
	return producer.ProviderImpl(p.l)(p.ctx)(merchant2.EnvCommandTopic)(PlaceShopCommandProvider(f, characterId, shopType, title, permitItemId, x, y))
}

func (p *ProcessorImpl) OpenShop(characterId uint32, shopId uuid.UUID) error {
	return producer.ProviderImpl(p.l)(p.ctx)(merchant2.EnvCommandTopic)(OpenShopCommandProvider(characterId, shopId))
}

func (p *ProcessorImpl) CloseShop(characterId uint32, shopId uuid.UUID) error {
	return producer.ProviderImpl(p.l)(p.ctx)(merchant2.EnvCommandTopic)(CloseShopCommandProvider(characterId, shopId))
}

func (p *ProcessorImpl) EnterShop(characterId uint32, shopId uuid.UUID, visitorName string) error {
	return producer.ProviderImpl(p.l)(p.ctx)(merchant2.EnvCommandTopic)(EnterShopCommandProvider(characterId, shopId, visitorName))
}

func (p *ProcessorImpl) AddBlacklist(characterId uint32, shopId uuid.UUID, name string, bannedCharacterId uint32) error {
	return producer.ProviderImpl(p.l)(p.ctx)(merchant2.EnvCommandTopic)(AddBlacklistCommandProvider(characterId, shopId, name, bannedCharacterId))
}

func (p *ProcessorImpl) RemoveBlacklist(characterId uint32, shopId uuid.UUID, name string) error {
	return producer.ProviderImpl(p.l)(p.ctx)(merchant2.EnvCommandTopic)(RemoveBlacklistCommandProvider(characterId, shopId, name))
}

// GetBlacklist fetches the complete blacklist for a shop. atlas-merchant's
// per-shop blacklist list is now paginated (task-117); the mini-room dialog
// here shows the whole set, so this drains every page rather than fetching
// one.
func (p *ProcessorImpl) GetBlacklist(shopId string) ([]string, error) {
	return requests.DrainProvider[BlacklistRestModel, string](p.l, p.ctx)(blacklistUrl(shopId), 250, ExtractBlacklistName, model.Filters[string]())()
}

// GetVisits fetches the complete visit log for a shop — paginated
// server-side (task-117) and drained here for the same whole-set dialog
// semantics as GetBlacklist.
func (p *ProcessorImpl) GetVisits(shopId string) ([]VisitEntry, error) {
	return requests.DrainProvider[VisitRestModel, VisitEntry](p.l, p.ctx)(visitsUrl(shopId), 250, ExtractVisitEntry, model.Filters[VisitEntry]())()
}

func (p *ProcessorImpl) ExitShop(characterId uint32, shopId uuid.UUID) error {
	return producer.ProviderImpl(p.l)(p.ctx)(merchant2.EnvCommandTopic)(ExitShopCommandProvider(characterId, shopId))
}

func (p *ProcessorImpl) SendMessage(characterId uint32, shopId uuid.UUID, content string) error {
	return producer.ProviderImpl(p.l)(p.ctx)(merchant2.EnvCommandTopic)(SendMessageCommandProvider(characterId, shopId, content))
}

func (p *ProcessorImpl) EnterMaintenance(characterId uint32, shopId uuid.UUID) error {
	return producer.ProviderImpl(p.l)(p.ctx)(merchant2.EnvCommandTopic)(EnterMaintenanceCommandProvider(characterId, shopId))
}

func (p *ProcessorImpl) ExitMaintenance(characterId uint32, shopId uuid.UUID) error {
	return producer.ProviderImpl(p.l)(p.ctx)(merchant2.EnvCommandTopic)(ExitMaintenanceCommandProvider(characterId, shopId))
}

func (p *ProcessorImpl) WithdrawMeso(characterId uint32, shopId uuid.UUID) error {
	return producer.ProviderImpl(p.l)(p.ctx)(merchant2.EnvCommandTopic)(WithdrawMesoCommandProvider(characterId, shopId))
}

func (p *ProcessorImpl) OrganizeListings(characterId uint32, shopId uuid.UUID) error {
	return producer.ProviderImpl(p.l)(p.ctx)(merchant2.EnvCommandTopic)(OrganizeListingsCommandProvider(characterId, shopId))
}

func (p *ProcessorImpl) AddListing(characterId uint32, shopId uuid.UUID, inventoryType byte, slot int16, quantity uint16, bundleSize uint16, pricePerBundle uint32) error {
	a, err := character.NewProcessor(p.l, p.ctx).GetItemInSlot(characterId, inventory2.Type(inventoryType), slot)()
	if err != nil {
		p.l.WithError(err).Errorf("Character [%d] attempting to list item from inventory [%d] slot [%d], but the item could not be resolved.", characterId, inventoryType, slot)
		return err
	}
	return producer.ProviderImpl(p.l)(p.ctx)(merchant2.EnvCommandTopic)(AddListingCommandProvider(characterId, shopId, inventoryType, slot, quantity, bundleSize, pricePerBundle, a))
}

func (p *ProcessorImpl) RemoveListing(characterId uint32, shopId uuid.UUID, listingIndex uint16) error {
	return producer.ProviderImpl(p.l)(p.ctx)(merchant2.EnvCommandTopic)(RemoveListingCommandProvider(characterId, shopId, listingIndex))
}

func (p *ProcessorImpl) PurchaseBundle(characterId uint32, shopId uuid.UUID, listingIndex uint16, bundleCount uint16) error {
	return producer.ProviderImpl(p.l)(p.ctx)(merchant2.EnvCommandTopic)(PurchaseBundleCommandProvider(characterId, shopId, listingIndex, bundleCount))
}

func (p *ProcessorImpl) SearchListings(worldId world.Id, itemId uint32, descending bool) ([]SearchListing, error) {
	return requests.SliceProvider[ListingSearchRestModel, SearchListing](p.l, p.ctx)(requestSearchListings(itemId, worldId, descending), ExtractSearchListing, model.Filters[SearchListing]())()
}

// GetTopSearches deliberately fetches a single page: the route is a bounded
// top-N ranking (LIMIT 10 server-side, task-117 envelope), so page 1 at the
// route's default size is always the complete collection — no drain needed.
func (p *ProcessorImpl) GetTopSearches(worldId world.Id) ([]TopSearch, error) {
	return requests.SliceProvider[TopSearchRestModel, TopSearch](p.l, p.ctx)(requestTopSearches(worldId), ExtractTopSearch, model.Filters[TopSearch]())()
}

func (p *ProcessorImpl) RecordItemSearch(f field.Model, characterId uint32, itemId uint32) error {
	return producer.ProviderImpl(p.l)(p.ctx)(merchant2.EnvCommandTopic)(RecordItemSearchCommandProvider(f, characterId, itemId))
}
