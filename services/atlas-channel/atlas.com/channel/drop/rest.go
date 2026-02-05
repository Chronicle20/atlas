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

func Extract(rm RestModel) (Model, error) {
	return Model{
		id:           rm.Id,
		itemId:       rm.ItemId,
		equipmentId:  rm.EquipmentId,
		quantity:     rm.Quantity,
		meso:         rm.Meso,
		dropType:     rm.Type,
		x:            rm.X,
		y:            rm.Y,
		ownerId:      rm.OwnerId,
		ownerPartyId: rm.OwnerPartyId,
		dropTime:     rm.DropTime,
		dropperId:    rm.DropperId,
		dropperX:     rm.DropperX,
		dropperY:     rm.DropperY,
		playerDrop:   rm.CharacterDrop,
	}, nil
}
