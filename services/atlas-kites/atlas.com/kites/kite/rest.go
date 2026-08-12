package kite

import (
	"strconv"
	"time"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

// RestModel is the JSON:API representation of a kite. Id holds the kite wire
// id -- NOT the character id. Chalkboards conflates the two (its RestModel.Id
// is the owning character's id, since a chalkboard message has no id of its
// own); a kite has a genuine wire id of its own (task-211 ADR-3), so its
// RestModel.Id must carry that, not CharacterId.
type RestModel struct {
	Id          uint32     `json:"-"`
	CharacterId uint32     `json:"characterId"`
	Name        string     `json:"name"`
	TemplateId  uint32     `json:"templateId"`
	Message     string     `json:"message"`
	X           int16      `json:"x"`
	Y           int16      `json:"y"`
	WorldId     world.Id   `json:"worldId"`
	ChannelId   channel.Id `json:"channelId"`
	MapId       _map.Id    `json:"mapId"`
	InstanceId  uuid.UUID  `json:"instanceId"`
	CreatedAt   time.Time  `json:"createdAt"`
}

func (r RestModel) GetName() string {
	return "kites"
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
		Id:          m.Id(),
		CharacterId: m.CharacterId(),
		Name:        m.Name(),
		TemplateId:  m.TemplateId(),
		Message:     m.Message(),
		X:           m.X(),
		Y:           m.Y(),
		WorldId:     m.Field().WorldId(),
		ChannelId:   m.Field().ChannelId(),
		MapId:       m.Field().MapId(),
		InstanceId:  m.Field().Instance(),
		CreatedAt:   m.CreatedAt(),
	}, nil
}
