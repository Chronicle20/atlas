package merchant

import "encoding/json"

type ListingRestModel struct {
	Id               string          `json:"-"`
	ShopId           string          `json:"shopId"`
	ItemId           uint32          `json:"itemId"`
	ItemType         byte            `json:"itemType"`
	Quantity         uint16          `json:"quantity"`
	BundleSize       uint16          `json:"bundleSize"`
	BundlesRemaining uint16          `json:"bundlesRemaining"`
	PricePerBundle   uint32          `json:"pricePerBundle"`
	ItemSnapshot     json.RawMessage `json:"itemSnapshot"`
	DisplayOrder     uint16          `json:"displayOrder"`
}

func (r ListingRestModel) GetID() string {
	return r.Id
}

func (r *ListingRestModel) SetID(id string) error {
	r.Id = id
	return nil
}

func (r ListingRestModel) GetName() string {
	return "listings"
}

type ListingModel struct {
	id               string
	shopId           string
	itemId           uint32
	itemType         byte
	quantity         uint16
	bundleSize       uint16
	bundlesRemaining uint16
	pricePerBundle   uint32
	itemSnapshot     json.RawMessage
	displayOrder     uint16
}

func (m ListingModel) Id() string                  { return m.id }
func (m ListingModel) ShopId() string              { return m.shopId }
func (m ListingModel) ItemId() uint32              { return m.itemId }
func (m ListingModel) ItemType() byte              { return m.itemType }
func (m ListingModel) Quantity() uint16            { return m.quantity }
func (m ListingModel) BundleSize() uint16          { return m.bundleSize }
func (m ListingModel) BundlesRemaining() uint16    { return m.bundlesRemaining }
func (m ListingModel) PricePerBundle() uint32      { return m.pricePerBundle }
func (m ListingModel) ItemSnapshot() json.RawMessage { return m.itemSnapshot }
func (m ListingModel) DisplayOrder() uint16        { return m.displayOrder }

func ExtractListing(rm ListingRestModel) (ListingModel, error) {
	return ListingModel{
		id:               rm.Id,
		shopId:           rm.ShopId,
		itemId:           rm.ItemId,
		itemType:         rm.ItemType,
		quantity:         rm.Quantity,
		bundleSize:       rm.BundleSize,
		bundlesRemaining: rm.BundlesRemaining,
		pricePerBundle:   rm.PricePerBundle,
		itemSnapshot:     rm.ItemSnapshot,
		displayOrder:     rm.DisplayOrder,
	}, nil
}
