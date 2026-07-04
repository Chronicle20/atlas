package testsupport

// SeedEntry describes one batch of listings to fabricate. Zero-valued fields
// take defaults in handleSeedListings (count 1, quantity 1, sellerId
// 999000001, sellerName "TestSeller", listValue 1000, durationSeconds 300 for
// auctions).
type SeedEntry struct {
	SaleType        string  `json:"saleType"` // "fixed" | "auction"
	Count           int     `json:"count,omitempty"`
	TemplateId      uint32  `json:"templateId"`
	Quantity        uint32  `json:"quantity,omitempty"`
	ListValue       uint32  `json:"listValue,omitempty"`
	BuyNowPrice     *uint32 `json:"buyNowPrice,omitempty"`
	StartingBid     uint32  `json:"startingBid,omitempty"`
	DurationSeconds int     `json:"durationSeconds,omitempty"`
	SellerId        uint32  `json:"sellerId,omitempty"`
	SellerAccountId uint32  `json:"sellerAccountId,omitempty"`
	SellerName      string  `json:"sellerName,omitempty"`
}

// SeedRestModel is the input envelope for POST /test/listings/seed.
type SeedRestModel struct {
	Id      string      `json:"-"`
	WorldId byte        `json:"worldId"`
	Entries []SeedEntry `json:"entries"`
}

func (r SeedRestModel) GetName() string { return "test-seeds" }

func (r SeedRestModel) GetID() string { return r.Id }

func (r *SeedRestModel) SetID(idStr string) error {
	r.Id = idStr
	return nil
}

// SweepResultRestModel is the response envelope for POST /test/sweep.
type SweepResultRestModel struct {
	Id    string `json:"-"`
	Swept int    `json:"swept"`
}

func (r SweepResultRestModel) GetName() string { return "test-sweeps" }

func (r SweepResultRestModel) GetID() string { return r.Id }

func (r *SweepResultRestModel) SetID(idStr string) error {
	r.Id = idStr
	return nil
}
