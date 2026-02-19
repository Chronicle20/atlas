package drop

import (
	"encoding/json"
	"time"

	"github.com/Chronicle20/atlas-constants/field"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
)

type modelJSON struct {
	Tenant        *tenant.Model `json:"tenant"`
	Id            uint32        `json:"id"`
	TransactionId uuid.UUID     `json:"transactionId"`
	Field         field.Model   `json:"field"`
	ItemId        uint32       `json:"itemId"`
	Quantity      uint32       `json:"quantity"`
	Meso          uint32       `json:"meso"`
	DropType      byte         `json:"dropType"`
	X             int16        `json:"x"`
	Y             int16        `json:"y"`
	OwnerId       uint32       `json:"ownerId"`
	OwnerPartyId  uint32       `json:"ownerPartyId"`
	DropTime      time.Time    `json:"dropTime"`
	DropperId     uint32       `json:"dropperId"`
	DropperX      int16        `json:"dropperX"`
	DropperY      int16        `json:"dropperY"`
	PlayerDrop    bool         `json:"playerDrop"`
	Status        string       `json:"status"`
	PetSlot       int8         `json:"petSlot"`
	Strength      uint16       `json:"strength"`
	Dexterity     uint16       `json:"dexterity"`
	Intelligence  uint16       `json:"intelligence"`
	Luck          uint16       `json:"luck"`
	Hp            uint16       `json:"hp"`
	Mp            uint16       `json:"mp"`
	WeaponAttack  uint16       `json:"weaponAttack"`
	MagicAttack   uint16       `json:"magicAttack"`
	WeaponDefense uint16       `json:"weaponDefense"`
	MagicDefense  uint16       `json:"magicDefense"`
	Accuracy      uint16       `json:"accuracy"`
	Avoidability  uint16       `json:"avoidability"`
	Hands         uint16       `json:"hands"`
	Speed         uint16       `json:"speed"`
	Jump          uint16       `json:"jump"`
	Slots         uint16       `json:"slots"`
}

func (m Model) MarshalJSON() ([]byte, error) {
	t := m.tenant
	return json.Marshal(modelJSON{
		Tenant:        &t,
		Id:            m.id,
		TransactionId: m.transactionId,
		Field:         m.field,
		ItemId:        m.itemId,
		Quantity:      m.quantity,
		Meso:          m.meso,
		DropType:      m.dropType,
		X:             m.x,
		Y:             m.y,
		OwnerId:       m.ownerId,
		OwnerPartyId:  m.ownerPartyId,
		DropTime:      m.dropTime,
		DropperId:     m.dropperId,
		DropperX:      m.dropperX,
		DropperY:      m.dropperY,
		PlayerDrop:    m.playerDrop,
		Status:        m.status,
		PetSlot:       m.petSlot,
		Strength:      m.strength,
		Dexterity:     m.dexterity,
		Intelligence:  m.intelligence,
		Luck:          m.luck,
		Hp:            m.hp,
		Mp:            m.mp,
		WeaponAttack:  m.weaponAttack,
		MagicAttack:   m.magicAttack,
		WeaponDefense: m.weaponDefense,
		MagicDefense:  m.magicDefense,
		Accuracy:      m.accuracy,
		Avoidability:  m.avoidability,
		Hands:         m.hands,
		Speed:         m.speed,
		Jump:          m.jump,
		Slots:         m.slots,
	})
}

func (m *Model) UnmarshalJSON(data []byte) error {
	var j modelJSON
	if err := json.Unmarshal(data, &j); err != nil {
		return err
	}
	if j.Tenant != nil {
		m.tenant = *j.Tenant
	}
	m.id = j.Id
	m.transactionId = j.TransactionId
	m.field = j.Field
	m.itemId = j.ItemId
	m.quantity = j.Quantity
	m.meso = j.Meso
	m.dropType = j.DropType
	m.x = j.X
	m.y = j.Y
	m.ownerId = j.OwnerId
	m.ownerPartyId = j.OwnerPartyId
	m.dropTime = j.DropTime
	m.dropperId = j.DropperId
	m.dropperX = j.DropperX
	m.dropperY = j.DropperY
	m.playerDrop = j.PlayerDrop
	m.status = j.Status
	m.petSlot = j.PetSlot
	m.strength = j.Strength
	m.dexterity = j.Dexterity
	m.intelligence = j.Intelligence
	m.luck = j.Luck
	m.hp = j.Hp
	m.mp = j.Mp
	m.weaponAttack = j.WeaponAttack
	m.magicAttack = j.MagicAttack
	m.weaponDefense = j.WeaponDefense
	m.magicDefense = j.MagicDefense
	m.accuracy = j.Accuracy
	m.avoidability = j.Avoidability
	m.hands = j.Hands
	m.speed = j.Speed
	m.jump = j.Jump
	m.slots = j.Slots
	return nil
}
