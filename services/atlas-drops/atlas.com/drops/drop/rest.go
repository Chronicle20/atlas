package drop

import (
	"strconv"
	"time"

	"github.com/Chronicle20/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/google/uuid"
)

type RestModel struct {
	Id            uint32     `json:"-"`
	WorldId       world.Id   `json:"worldId"`
	ChannelId     channel.Id `json:"channelId"`
	MapId         _map.Id    `json:"mapId"`
	Instance      uuid.UUID  `json:"instance"`
	ItemId        uint32     `json:"itemId"`
	Quantity      uint32     `json:"quantity"`
	Meso          uint32     `json:"meso"`
	Type          byte       `json:"type"`
	X             int16      `json:"x"`
	Y             int16      `json:"y"`
	OwnerId       uint32     `json:"ownerId"`
	OwnerPartyId  uint32     `json:"ownerPartyId"`
	DropTime      time.Time  `json:"dropTime"`
	DropperId     uint32     `json:"dropperId"`
	DropperX      int16      `json:"dropperX"`
	DropperY      int16      `json:"dropperY"`
	CharacterDrop bool       `json:"characterDrop"`
	Mod           byte       `json:"mod"`
	Strength      uint16     `json:"strength"`
	Dexterity     uint16     `json:"dexterity"`
	Intelligence  uint16     `json:"intelligence"`
	Luck          uint16     `json:"luck"`
	Hp            uint16     `json:"hp"`
	Mp            uint16     `json:"mp"`
	WeaponAttack  uint16     `json:"weaponAttack"`
	MagicAttack   uint16     `json:"magicAttack"`
	WeaponDefense uint16     `json:"weaponDefense"`
	MagicDefense  uint16     `json:"magicDefense"`
	Accuracy      uint16     `json:"accuracy"`
	Avoidability  uint16     `json:"avoidability"`
	Hands         uint16     `json:"hands"`
	Speed         uint16     `json:"speed"`
	Jump          uint16     `json:"jump"`
	Slots         uint16     `json:"slots"`
}

func (r RestModel) GetName() string {
	return "drops"
}

func (r RestModel) GetID() string {
	return strconv.Itoa(int(r.Id))
}

func (r *RestModel) SetID(id string) error {
	strId, err := strconv.ParseUint(id, 10, 32)
	if err != nil {
		return err
	}
	r.Id = uint32(strId)
	return nil
}

func Transform(m Model) (RestModel, error) {
	return RestModel{
		Id:            m.Id(),
		WorldId:       m.WorldId(),
		ChannelId:     m.ChannelId(),
		MapId:         m.MapId(),
		Instance:      m.Instance(),
		ItemId:        m.ItemId(),
		Quantity:      m.Quantity(),
		Meso:          m.Meso(),
		Type:          m.Type(),
		X:             m.X(),
		Y:             m.Y(),
		OwnerId:       m.OwnerId(),
		OwnerPartyId:  m.OwnerPartyId(),
		DropTime:      m.DropTime(),
		DropperId:     m.DropperId(),
		DropperX:      m.DropperX(),
		DropperY:      m.DropperY(),
		CharacterDrop: m.CharacterDrop(),
		Strength:      m.Strength(),
		Dexterity:     m.Dexterity(),
		Intelligence:  m.Intelligence(),
		Luck:          m.Luck(),
		Hp:            m.Hp(),
		Mp:            m.Mp(),
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
	}, nil
}
