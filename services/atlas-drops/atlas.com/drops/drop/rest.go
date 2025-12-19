package drop

import (
	"github.com/Chronicle20/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
	"strconv"
	"time"
)

type RestModel struct {
	Id            uint32     `json:"-"`
	WorldId       world.Id   `json:"worldId"`
	ChannelId     channel.Id `json:"channelId"`
	MapId         _map.Id    `json:"mapId"`
	ItemId        uint32     `json:"itemId"`
	EquipmentId   uint32     `json:"equipmentId"`
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
		ItemId:        m.ItemId(),
		EquipmentId:   m.EquipmentId(),
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
	}, nil
}
