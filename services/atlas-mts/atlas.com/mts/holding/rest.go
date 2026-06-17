package holding

import "time"

// RestModel is the JSON:API representation of a take-home holding: the item
// snapshot plus the origin that placed it in the owner's holding bucket.
type RestModel struct {
	Id      string `json:"-"`
	WorldId byte   `json:"worldId"`
	OwnerId uint32 `json:"ownerId"`
	Origin  string `json:"origin"`

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

	CreatedAt time.Time `json:"createdAt"`
}

func (r RestModel) GetName() string {
	return "holdings"
}

// TakeHomeRestModel is the JSON:API request/response for a take-home initiation.
// inventoryType is the destination inventory type; slot is the advisory target
// slot (the inventory grant auto-slots — WithdrawFromMtsPayload carries no slot,
// so this is not propagated to the saga). The response carries the allocated
// transaction id in Id.
type TakeHomeRestModel struct {
	Id            string `json:"-"`
	InventoryType byte   `json:"inventoryType"`
	Slot          int16  `json:"slot"`
}

func (r TakeHomeRestModel) GetName() string {
	return "holdings"
}

func (r TakeHomeRestModel) GetID() string {
	return r.Id
}

func (r *TakeHomeRestModel) SetID(idStr string) error {
	r.Id = idStr
	return nil
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
		Id:            m.Id().String(),
		WorldId:       byte(m.WorldId()),
		OwnerId:       m.OwnerId(),
		Origin:        string(m.Origin()),
		TemplateId:    m.TemplateId(),
		Quantity:      m.Quantity(),
		Strength:      m.Strength(),
		Dexterity:     m.Dexterity(),
		Intelligence:  m.Intelligence(),
		Luck:          m.Luck(),
		HP:            m.HP(),
		MP:            m.MP(),
		WeaponAttack:  m.WeaponAttack(),
		MagicAttack:   m.MagicAttack(),
		WeaponDefense: m.WeaponDefense(),
		MagicDefense:  m.MagicDefense(),
		Accuracy:      m.Accuracy(),
		Avoidability:  m.Avoidability(),
		Hands:         m.Hands(),
		Speed:         m.Speed(),
		Jump:          m.Jump(),
		Slots:         m.Slots(),
		Level:         m.Level(),
		ItemLevel:     m.ItemLevel(),
		ItemExp:       m.ItemExp(),
		RingId:        m.RingId(),
		ViciousCount:  m.ViciousCount(),
		Flags:         m.Flags(),
		CreatedAt:     m.CreatedAt(),
	}, nil
}
