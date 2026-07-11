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
	fixedSaleHours    int     // fixed-price sale term in hours (era-faithful: listings expire back to the seller)
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

func (m Model) FixedSaleDurationHours() int {
	return m.fixedSaleHours
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
		fixedSaleHours:    168,   // hours — era-faithful 7-day fixed-sale term (knob, no IDA reference)
		priceFloor:        110,   // NX, IDA-verified
		pageSize:          16,    //
		minBidIncrement:   1,     // chosen default (no IDA reference)
	}
}
