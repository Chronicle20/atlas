package asset

import (
	"strconv"
	"time"
)

type RestModel struct {
	Id             uint32     `json:"-"`
	Slot           int16      `json:"slot"`
	TemplateId     uint32     `json:"templateId"`
	Expiration     time.Time  `json:"expiration"`
	CreatedAt      time.Time  `json:"createdAt"`
	Quantity       uint32     `json:"quantity"`
	OwnerId        uint32     `json:"ownerId"`
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
	Slots     uint16 `json:"slots"`
	LevelType byte   `json:"levelType"`
	Level          byte       `json:"level"`
	Experience     uint32     `json:"experience"`
	HammersApplied uint32     `json:"hammersApplied"`
	EquippedSince  *time.Time `json:"equippedSince"`
	CashId         int64      `json:"cashId,string"`
	CommodityId    uint32     `json:"commodityId"`
	PurchaseBy     uint32     `json:"purchaseBy"`
	PetId          uint32     `json:"petId"`
	PetName        string     `json:"petName"`
	PetLevel       byte       `json:"petLevel"`
	Closeness      uint16     `json:"closeness"`
	Fullness       byte       `json:"fullness"`
	PetSlot        int8       `json:"petSlot"`
}

// BaseRestModel is an alias for RestModel for backward compatibility
type BaseRestModel = RestModel

func (r RestModel) GetName() string {
	return "assets"
}

func (r RestModel) GetID() string {
	return strconv.Itoa(int(r.Id))
}

func (r *RestModel) SetID(strId string) error {
	id, err := strconv.Atoi(strId)
	if err != nil {
		return err
	}
	r.Id = uint32(id)
	return nil
}

func Transform(m Model) (RestModel, error) {
	return RestModel{
		Id:             m.id,
		Slot:           m.slot,
		TemplateId:     m.templateId,
		Expiration:     m.expiration,
		CreatedAt:      m.createdAt,
		Quantity:       m.Quantity(),
		OwnerId:        m.ownerId,
		Flag:           m.flag,
		Rechargeable:   m.rechargeable,
		Strength:       m.strength,
		Dexterity:      m.dexterity,
		Intelligence:   m.intelligence,
		Luck:           m.luck,
		Hp:             m.hp,
		Mp:             m.mp,
		WeaponAttack:   m.weaponAttack,
		MagicAttack:    m.magicAttack,
		WeaponDefense:  m.weaponDefense,
		MagicDefense:   m.magicDefense,
		Accuracy:       m.accuracy,
		Avoidability:   m.avoidability,
		Hands:          m.hands,
		Speed:          m.speed,
		Jump:           m.jump,
		Slots:          m.slots,
		LevelType:      m.levelType,
		Level:          m.level,
		Experience:     m.experience,
		HammersApplied: m.hammersApplied,
		EquippedSince:  m.equippedSince,
		CashId:         m.cashId,
		CommodityId:    m.commodityId,
		PurchaseBy:     m.purchaseBy,
		PetId:          m.petId,
		PetName:        m.petName,
		PetLevel:       m.petLevel,
		Closeness:      m.closeness,
		Fullness:       m.fullness,
		PetSlot:        m.petSlot,
	}, nil
}

func Extract(rm RestModel) (Model, error) {
	return Model{
		id:             rm.Id,
		slot:           rm.Slot,
		templateId:     rm.TemplateId,
		expiration:     rm.Expiration,
		createdAt:      rm.CreatedAt,
		quantity:       rm.Quantity,
		ownerId:        rm.OwnerId,
		flag:           rm.Flag,
		rechargeable:   rm.Rechargeable,
		strength:       rm.Strength,
		dexterity:      rm.Dexterity,
		intelligence:   rm.Intelligence,
		luck:           rm.Luck,
		hp:             rm.Hp,
		mp:             rm.Mp,
		weaponAttack:   rm.WeaponAttack,
		magicAttack:    rm.MagicAttack,
		weaponDefense:  rm.WeaponDefense,
		magicDefense:   rm.MagicDefense,
		accuracy:       rm.Accuracy,
		avoidability:   rm.Avoidability,
		hands:          rm.Hands,
		speed:          rm.Speed,
		jump:           rm.Jump,
		slots:     rm.Slots,
		levelType: rm.LevelType,
		level:          rm.Level,
		experience:     rm.Experience,
		hammersApplied: rm.HammersApplied,
		equippedSince:  rm.EquippedSince,
		cashId:         rm.CashId,
		commodityId:    rm.CommodityId,
		purchaseBy:     rm.PurchaseBy,
		petId:          rm.PetId,
		petName:        rm.PetName,
		petLevel:       rm.PetLevel,
		closeness:      rm.Closeness,
		fullness:       rm.Fullness,
		petSlot:        rm.PetSlot,
	}, nil
}
