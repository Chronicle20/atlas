package parcel

import "time"

// AssetData is the orchestrator's view of atlas-parcel's item snapshot,
// captured at send time and returned verbatim by GET /parcels/{parcelId}.
// Mirrors services/atlas-parcel/atlas.com/parcel/parcel/asset_data.go.
type AssetData struct {
	Expiration     time.Time  `json:"expiration"`
	CreatedAt      time.Time  `json:"createdAt"`
	Quantity       uint32     `json:"quantity"`
	OwnerId        uint32     `json:"ownerId"`
	Owner          string     `json:"owner"`
	Flag           uint16     `json:"flag"`
	Rechargeable   uint64     `json:"rechargeable"`
	Strength       uint16     `json:"strength"`
	Dexterity      uint16     `json:"dexterity"`
	Intelligence   uint16     `json:"intelligence"`
	Luck           uint16     `json:"luck"`
	Hp             uint16     `json:"hp"`
	Mp             uint16     `json:"mp"`
	WeaponAttack   uint16     `json:"weaponAttack"`
	MagicAttack    uint16     `json:"magicAttack"`
	WeaponDefense  uint16     `json:"weaponDefense"`
	MagicDefense   uint16     `json:"magicDefense"`
	Accuracy       uint16     `json:"accuracy"`
	Avoidability   uint16     `json:"avoidability"`
	Hands          uint16     `json:"hands"`
	Speed          uint16     `json:"speed"`
	Jump           uint16     `json:"jump"`
	Slots          uint16     `json:"slots"`
	LevelType      byte       `json:"levelType"`
	Level          byte       `json:"level"`
	Experience     uint32     `json:"experience"`
	HammersApplied uint32     `json:"hammersApplied"`
	EquippedSince  *time.Time `json:"equippedSince"`
	CashId         int64      `json:"cashId,string"`
	CommodityId    uint32     `json:"commodityId"`
	PurchaseBy     uint32     `json:"purchaseBy"`
	PetId          uint32     `json:"petId"`
}

// RestModel is the orchestrator's view of an atlas-parcel row, carrying the
// full item snapshot needed to re-grant the item to a character's inventory
// on WithdrawFromParcel. Mirrors
// services/atlas-parcel/atlas.com/parcel/parcel/rest.go RestModel.
//
// ItemId is nil for a meso-only parcel (design §12 RISK-2) — the same
// escape hatch AcceptToParcelPayload.HasItem pins on the send side.
type RestModel struct {
	Id string `json:"-"`

	WorldId byte `json:"worldId"`

	SenderId        uint32 `json:"senderId"`
	SenderAccountId uint32 `json:"senderAccountId"`
	SenderName      string `json:"senderName"`

	RecipientId        uint32 `json:"recipientId"`
	RecipientAccountId uint32 `json:"recipientAccountId"`

	Message    string `json:"message"`
	MesoAmount uint32 `json:"mesoAmount"`
	FeePaid    uint32 `json:"feePaid"`

	ItemId       *uint32   `json:"itemId,omitempty"`
	ItemType     byte      `json:"itemType"`
	Quantity     uint16    `json:"quantity"`
	ItemSnapshot AssetData `json:"itemSnapshot"`

	Status   string `json:"status"`
	Quick    bool   `json:"quick"`
	Returned bool   `json:"returned"`
}

func (r RestModel) GetName() string {
	return "parcels"
}

func (r RestModel) GetID() string {
	return r.Id
}

func (r *RestModel) SetID(id string) error {
	r.Id = id
	return nil
}

// Required JSON:API relationship stubs (libs/atlas-rest gotcha): api2go errors
// out decoding any response unless the target implements these, even with no
// relationships present.
func (r *RestModel) SetToOneReferenceID(_, _ string) error { return nil }

func (r *RestModel) SetToManyReferenceIDs(_ string, _ []string) error { return nil }
