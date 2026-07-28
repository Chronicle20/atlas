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

// PurchaseRestModel is the input for POST /test/purchases — simulate another
// character buying a listing. BuyNow=true is the auction immediate-buyout arm.
type PurchaseRestModel struct {
	Id             string `json:"-"`
	ListingId      string `json:"listingId"`
	BuyerId        uint32 `json:"buyerId"`
	BuyerAccountId uint32 `json:"buyerAccountId"`
	BuyNow         bool   `json:"buyNow,omitempty"`
}

func (r PurchaseRestModel) GetName() string { return "test-purchases" }

func (r PurchaseRestModel) GetID() string { return r.Id }

func (r *PurchaseRestModel) SetID(idStr string) error {
	r.Id = idStr
	return nil
}

// BidRestModel is the input for POST /test/bids — simulate a competing bidder.
type BidRestModel struct {
	Id              string `json:"-"`
	ListingId       string `json:"listingId"`
	BidderId        uint32 `json:"bidderId"`
	BidderAccountId uint32 `json:"bidderAccountId"`
	Amount          uint32 `json:"amount"`
}

func (r BidRestModel) GetName() string { return "test-bids" }

func (r BidRestModel) GetID() string { return r.Id }

func (r *BidRestModel) SetID(idStr string) error {
	r.Id = idStr
	return nil
}
