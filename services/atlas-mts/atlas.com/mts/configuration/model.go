package configuration

// Model is the immutable per-tenant MTS configuration. It holds the economic
// knobs that govern listing fees, commissions, auction durations, and paging.
// Fields are private with getters; construct via the builder or DefaultConfig.
type Model struct {
	listingFee        uint32  // flat meso fee charged to the seller to create a listing
	commissionRate    float64 // buyer-markup rate on the sale price (client m_nCommissionRate%, e.g. 0.07 = 7%)
	commissionBase    uint32  // flat NX added to the buyer's payment (client m_nCommissionBase, e.g. 500)
	maxActiveListings int     // per-character cap on concurrently active listings
	minLevel          int     // minimum character level required to use the MTS
	auctionMinHours   int     // minimum auction duration in hours
	auctionMaxHours   int     // maximum auction duration in hours
	priceFloor        uint32  // minimum NX price (IDA-verified floor)
	pageSize          int     // results returned per browse page
	minBidIncrement   uint32  // minimum increment over the current bid
}

func (m Model) ListingFee() uint32 {
	return m.listingFee
}

func (m Model) CommissionRate() float64 {
	return m.commissionRate
}

func (m Model) CommissionBase() uint32 {
	return m.commissionBase
}

func (m Model) MaxActiveListings() int {
	return m.maxActiveListings
}

func (m Model) MinLevel() int {
	return m.minLevel
}

func (m Model) AuctionMinHours() int {
	return m.auctionMinHours
}

func (m Model) AuctionMaxHours() int {
	return m.auctionMaxHours
}

func (m Model) PriceFloor() uint32 {
	return m.priceFloor
}

func (m Model) PageSize() int {
	return m.pageSize
}

func (m Model) MinBidIncrement() uint32 {
	return m.minBidIncrement
}

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

// DefaultConfig returns the Model populated with the economic-knob defaults.
// The registry falls back to these whenever the tenant has not configured the
// MTS (a fetch miss or error), so the service never hard-fails on a missing
// configuration resource.
func DefaultConfig() Model {
	return Model{
		listingFee:        5000,  // flat meso seller fee to list
		commissionRate:    0.07,  // buyer-markup rate (client m_nCommissionRate, IDA-verified)
		commissionBase:    500,   // flat NX added to buyer payment (client m_nCommissionBase, IDA-verified)
		maxActiveListings: 10,    //
		minLevel:          10,    //
		auctionMinHours:   24,    // hours
		auctionMaxHours:   168,   // hours (1-week cap, 1-hour step)
		priceFloor:        110,   // NX, IDA-verified
		pageSize:          16,    //
		minBidIncrement:   1,     // chosen default (no IDA reference)
	}
}
