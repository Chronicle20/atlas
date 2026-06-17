package listing

import "time"

// RestModel is the JSON:API representation of a marketplace listing. It covers
// both the browse (list) and detail (single) attribute surface: the full item
// snapshot plus the sale/auction/state fields.
type RestModel struct {
	Id         string `json:"-"`
	WorldId    byte   `json:"worldId"`
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
	Category       string  `json:"category"`
	SubCategory    string  `json:"subCategory"`

	EndsAt       *time.Time `json:"endsAt,omitempty"`
	CurrentBid   uint32     `json:"currentBid"`
	HighBidderId uint32     `json:"highBidderId"`
	MinIncrement uint32     `json:"minIncrement"`

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

func Transform(m Model) (RestModel, error) {
	return RestModel{
		Id:             m.Id().String(),
		WorldId:        byte(m.WorldId()),
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
		CreatedAt:      m.CreatedAt(),
		UpdatedAt:      m.UpdatedAt(),
	}, nil
}
