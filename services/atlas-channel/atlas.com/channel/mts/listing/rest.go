package listing

import (
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

// RestModel mirrors atlas-mts's listing.RestModel (the JSON:API "listings"
// resource). It carries the full item snapshot plus the sale/auction/state fields
// and the persistent ItcSn serial. The To-One/To-Many relationship stubs are
// required boilerplate for the api2go unmarshal even though listings have no
// relationships block (see libs/atlas-rest/CLAUDE.md).
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

	ListValue   uint32 `json:"listValue"`
	BuyNowPrice uint32 `json:"buyNowPrice"`
	Category    string `json:"category"`
	SubCategory string `json:"subCategory"`

	ContractFee  uint32 `json:"contractFee"`
	CurrentBid   uint32 `json:"currentBid"`
	HighBidderId uint32 `json:"highBidderId"`
	MinIncrement uint32 `json:"minIncrement"`
	BidCount     uint32 `json:"bidCount"`

	EndsAt *time.Time `json:"endsAt,omitempty"`
}

func (r RestModel) GetName() string { return "listings" }
func (r RestModel) GetID() string   { return r.Id }

func (r *RestModel) SetID(idStr string) error {
	r.Id = idStr
	return nil
}

// Required api2go relationship stubs (listings carry no relationships, but the
// unmarshal path walks the interfaces — see libs/atlas-rest/CLAUDE.md).
func (r *RestModel) SetToOneReferenceID(_, _ string) error            { return nil }
func (r *RestModel) SetToManyReferenceIDs(_ string, _ []string) error { return nil }

func Extract(r RestModel) (Model, error) {
	return Model{
		id:            r.Id,
		worldId:       world.Id(r.WorldId),
		itcSn:         r.ItcSn,
		sellerId:      r.SellerId,
		sellerName:    r.SellerName,
		saleType:      r.SaleType,
		state:         r.State,
		templateId:    r.TemplateId,
		quantity:      r.Quantity,
		strength:      r.Strength,
		dexterity:     r.Dexterity,
		intelligence:  r.Intelligence,
		luck:          r.Luck,
		hp:            r.HP,
		mp:            r.MP,
		weaponAttack:  r.WeaponAttack,
		magicAttack:   r.MagicAttack,
		weaponDefense: r.WeaponDefense,
		magicDefense:  r.MagicDefense,
		accuracy:      r.Accuracy,
		avoidability:  r.Avoidability,
		hands:         r.Hands,
		speed:         r.Speed,
		jump:          r.Jump,
		slots:         r.Slots,
		level:         r.Level,
		itemLevel:     r.ItemLevel,
		itemExp:       r.ItemExp,
		ringId:        r.RingId,
		viciousCount:  r.ViciousCount,
		flags:         r.Flags,
		listValue:     r.ListValue,
		buyNowPrice:   r.BuyNowPrice,
		contractFee:   r.ContractFee,
		currentBid:    r.CurrentBid,
		highBidderId:  r.HighBidderId,
		minIncrement:  r.MinIncrement,
		bidCount:      r.BidCount,
		category:      r.Category,
		subCategory:   r.SubCategory,
		endsAt:        r.EndsAt,
	}, nil
}
