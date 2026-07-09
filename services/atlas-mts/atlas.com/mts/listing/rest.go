package listing

import "time"

// RestModel is the JSON:API representation of a marketplace listing. It covers
// both the browse (list) and detail (single) attribute surface: the full item
// snapshot plus the sale/auction/state fields. ItcSn is the listing's persistent
// per-(tenant, world) ITC serial (the client's nITCSN) — the channel emits it as
// MtsItem.itcSn so the client can address this listing in subsequent
// buy/cancel/bid ITC_OPERATION arms.
type RestModel struct {
	Id         string `json:"-"`
	WorldId    byte   `json:"worldId"`
	ItcSn      uint32 `json:"itcSn"`
	SellerId   uint32 `json:"sellerId"`
	SellerName string `json:"sellerName"`

	SaleType string `json:"saleType"`
	State    string `json:"state"`

	TemplateId uint32 `json:"templateId"`
	Quantity   uint32 `json:"quantity"`

	Strength      uint16 `json:"strength"`
	Dexterity     uint16 `json:"dexterity"`
	Intelligence  uint16 `json:"intelligence"`
	Luck          uint16 `json:"luck"`
	HP            uint16 `json:"hp"`
	MP            uint16 `json:"mp"`
	WeaponAttack  uint16 `json:"weaponAttack"`
	MagicAttack   uint16 `json:"magicAttack"`
	WeaponDefense uint16 `json:"weaponDefense"`
	MagicDefense  uint16 `json:"magicDefense"`
	Accuracy      uint16 `json:"accuracy"`
	Avoidability  uint16 `json:"avoidability"`
	Hands         uint16 `json:"hands"`
	Speed         uint16 `json:"speed"`
	Jump          uint16 `json:"jump"`
	Slots         uint16 `json:"slots"`
	Level         byte   `json:"level"`
	ItemLevel     byte   `json:"itemLevel"`
	ItemExp       uint32 `json:"itemExp"`
	RingId        uint32 `json:"ringId"`
	ViciousCount  uint32 `json:"viciousCount"`
	Flags         uint16 `json:"flags"`

	ListValue      uint32  `json:"listValue"`
	BuyNowPrice    *uint32 `json:"buyNowPrice,omitempty"`
	CommissionRate float64 `json:"commissionRate"`
	// ContractFee is the buyer-visible fee on top of the current price
	// (markedUp(base)-base, base = currentBid for auctions else listValue). The
	// channel adds it into the client's price column. Populated by the browse/get
	// handlers (they hold the tenant commissionBase); Transform alone leaves it 0.
	ContractFee uint32 `json:"contractFee"`
	Category    string `json:"category"`
	SubCategory string `json:"subCategory"`

	EndsAt       *time.Time `json:"endsAt,omitempty"`
	CurrentBid   uint32     `json:"currentBid"`
	HighBidderId uint32     `json:"highBidderId"`
	MinIncrement uint32     `json:"minIncrement"`
	BidCount     uint32     `json:"bidCount"`

	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func (r RestModel) GetName() string {
	return "listings"
}

func (r RestModel) GetID() string {
	return r.Id
}

func (r *RestModel) SetID(idStr string) error {
	r.Id = idStr
	return nil
}

// CreateListingRestModel is the JSON:API request body for POST
// /worlds/{worldId}/listings. It carries the seller's list parameters: the item
// reference being listed (the inventory asset id + its inventory type), the sale
// type and price, and (for auctions) the duration. The seller identity and the
// item snapshot are NOT trusted from the body — sellerId/sellerName come from the
// authenticated request and the snapshot is looked up during saga expansion.
//
// GetName returns "listings" so the envelope {data:{type:"listings",attributes:
// {...}}} unmarshals; bare bodies 400.
type CreateListingRestModel struct {
	Id                  string  `json:"-"`
	SellerId            uint32  `json:"sellerId"`
	SellerAccountId     uint32  `json:"sellerAccountId"`
	SellerName          string  `json:"sellerName"`
	SaleType            string  `json:"saleType"`
	SourceInventoryType byte    `json:"sourceInventoryType"`
	AssetId             uint32  `json:"assetId"`
	Quantity            uint32  `json:"quantity"`
	ListValue           uint32  `json:"listValue"`
	BuyNowPrice         *uint32 `json:"buyNowPrice,omitempty"`
	DurationHours       int     `json:"durationHours,omitempty"`
	Category            string  `json:"category"`
	SubCategory         string  `json:"subCategory"`
}

func (r CreateListingRestModel) GetName() string {
	return "listings"
}

func (r CreateListingRestModel) GetID() string {
	return r.Id
}

func (r *CreateListingRestModel) SetID(idStr string) error {
	r.Id = idStr
	return nil
}

func Transform(m Model) (RestModel, error) {
	return RestModel{
		Id:             m.Id().String(),
		WorldId:        byte(m.WorldId()),
		ItcSn:          m.Serial(),
		SellerId:       m.SellerId(),
		SellerName:     m.SellerName(),
		SaleType:       string(m.SaleType()),
		State:          string(m.State()),
		TemplateId:     m.TemplateId(),
		Quantity:       m.Quantity(),
		Strength:       m.Strength(),
		Dexterity:      m.Dexterity(),
		Intelligence:   m.Intelligence(),
		Luck:           m.Luck(),
		HP:             m.HP(),
		MP:             m.MP(),
		WeaponAttack:   m.WeaponAttack(),
		MagicAttack:    m.MagicAttack(),
		WeaponDefense:  m.WeaponDefense(),
		MagicDefense:   m.MagicDefense(),
		Accuracy:       m.Accuracy(),
		Avoidability:   m.Avoidability(),
		Hands:          m.Hands(),
		Speed:          m.Speed(),
		Jump:           m.Jump(),
		Slots:          m.Slots(),
		Level:          m.Level(),
		ItemLevel:      m.ItemLevel(),
		ItemExp:        m.ItemExp(),
		RingId:         m.RingId(),
		ViciousCount:   m.ViciousCount(),
		Flags:          m.Flags(),
		ListValue:      m.ListValue(),
		BuyNowPrice:    m.BuyNowPrice(),
		CommissionRate: m.CommissionRate(),
		Category:       m.Category(),
		SubCategory:    m.SubCategory(),
		EndsAt:         m.EndsAt(),
		CurrentBid:     m.CurrentBid(),
		HighBidderId:   m.HighBidderId(),
		MinIncrement:   m.MinIncrement(),
		BidCount:       m.BidCount(),
		CreatedAt:      m.CreatedAt(),
		UpdatedAt:      m.UpdatedAt(),
	}, nil
}

// withContractFee stamps the buyer-visible fee (markedUp(listValue)-listValue)
// onto a FIXED-sale REST model, so the client's price column shows nPrice +
// nContractFee = the true final cost. commissionBase is the tenant flat NX fee
// (the caller reads it once from tenant config).
//
// Auctions carry NO contract fee (0). The v83 client already computes the
// buyer's commission itself for the bid dialog's "Your Bid" line
// (CITCBidAuctionDlg::GetPriceWithCommision = commissionBase +
// (commissionRate+100)*bid/100, IDA-verified v95 0x58b5e0), so adding
// nContractFee would double-count it onto the "Highest Bid" column — and on a
// no-bid auction (nBidPrice = the opening price, the bid floor) it inflated the
// displayed opening price (e.g. 1000 -> 1570). The fee is realised at settle via
// markedUp on the escrow, not shown on the bid line.
func withContractFee(rm RestModel, commissionBase uint32) RestModel {
	if rm.SaleType == string(SaleTypeAuction) {
		rm.ContractFee = 0
		return rm
	}
	rm.ContractFee = markedUp(rm.ListValue, rm.CommissionRate, commissionBase) - rm.ListValue
	return rm
}
