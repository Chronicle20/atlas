package listing

import (
	"atlas-merchant/kafka/message/asset"
)

type RestModel struct {
	Id               string          `json:"-"`
	ShopId           string          `json:"shopId"`
	ItemId           uint32          `json:"itemId"`
	ItemType         byte            `json:"itemType"`
	Quantity         uint16          `json:"quantity"`
	BundleSize       uint16          `json:"bundleSize"`
	BundlesRemaining uint16          `json:"bundlesRemaining"`
	PricePerBundle   uint32          `json:"pricePerBundle"`
	ItemSnapshot     asset.AssetData `json:"itemSnapshot"`
	DisplayOrder     uint16          `json:"displayOrder"`
}

func (r RestModel) GetID() string {
	return r.Id
}

func (r *RestModel) SetID(id string) error {
	r.Id = id
	return nil
}

func (r RestModel) GetName() string {
	return "listings"
}

func Transform(m Model) (RestModel, error) {
	return RestModel{
		Id:               m.Id().String(),
		ShopId:           m.ShopId().String(),
		ItemId:           m.ItemId(),
		ItemType:         m.ItemType(),
		Quantity:         m.Quantity(),
		BundleSize:       m.BundleSize(),
		BundlesRemaining: m.BundlesRemaining(),
		PricePerBundle:   m.PricePerBundle(),
		ItemSnapshot:     m.ItemSnapshot(),
		DisplayOrder:     m.DisplayOrder(),
	}, nil
}
