package configuration

// RestModel is the JSON:API representation of the MTS configuration fetched
// from atlas-tenants. Fields default to the zero value when atlas-tenants has
// not yet provisioned the resource (Phase 8); Extract folds any zero knob back
// to its default so a partial config never yields a nonsensical zero.
type RestModel struct {
	Id                string  `json:"-"`
	ListingFee        uint32  `json:"listingFee"`
	CommissionRate    float64 `json:"commissionRate"`
	CommissionBase    uint32  `json:"commissionBase"`
	MaxActiveListings int     `json:"maxActiveListings"`
	MinLevel          int     `json:"minLevel"`
	AuctionMinHours   int     `json:"auctionMinHours"`
	AuctionMaxHours   int     `json:"auctionMaxHours"`
	FixedSaleHours    int     `json:"fixedSaleHours"`
	PriceFloor        uint32  `json:"priceFloor"`
	PageSize          int     `json:"pageSize"`
	MinBidIncrement   uint32  `json:"minBidIncrement"`
}

func (r RestModel) GetName() string {
	return "mts-configs"
}

func (r RestModel) GetID() string {
	return r.Id
}

func (r *RestModel) SetID(id string) error {
	r.Id = id
	return nil
}

// Extract converts the fetched RestModel into the immutable domain Model,
// substituting the default for any knob left at its zero value.
func Extract(r RestModel) Model {
	d := DefaultConfig()
	m := Model{
		listingFee:        r.ListingFee,
		commissionRate:    r.CommissionRate,
		commissionBase:    r.CommissionBase,
		maxActiveListings: r.MaxActiveListings,
		minLevel:          r.MinLevel,
		auctionMinHours:   r.AuctionMinHours,
		auctionMaxHours:   r.AuctionMaxHours,
		fixedSaleHours:    r.FixedSaleHours,
		priceFloor:        r.PriceFloor,
		pageSize:          r.PageSize,
		minBidIncrement:   r.MinBidIncrement,
	}
	if m.listingFee == 0 {
		m.listingFee = d.listingFee
	}
	if m.commissionRate == 0 {
		m.commissionRate = d.commissionRate
	}
	if m.commissionBase == 0 {
		m.commissionBase = d.commissionBase
	}
	if m.maxActiveListings == 0 {
		m.maxActiveListings = d.maxActiveListings
	}
	if m.minLevel == 0 {
		m.minLevel = d.minLevel
	}
	if m.auctionMinHours == 0 {
		m.auctionMinHours = d.auctionMinHours
	}
	if m.auctionMaxHours == 0 {
		m.auctionMaxHours = d.auctionMaxHours
	}
	if m.fixedSaleHours == 0 {
		m.fixedSaleHours = d.fixedSaleHours
	}
	if m.priceFloor == 0 {
		m.priceFloor = d.priceFloor
	}
	if m.pageSize == 0 {
		m.pageSize = d.pageSize
	}
	if m.minBidIncrement == 0 {
		m.minBidIncrement = d.minBidIncrement
	}
	return m
}
