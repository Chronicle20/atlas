package summon

import (
	"strconv"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

type RestModel struct {
	Id               string     `json:"-"`
	OwnerCharacterId uint32     `json:"ownerCharacterId"`
	SkillId          uint32     `json:"skillId"`
	SkillLevel       byte       `json:"skillLevel"`
	SummonType       string     `json:"summonType"`
	MovementType     byte       `json:"movementType"`
	X                int16      `json:"x"`
	Y                int16      `json:"y"`
	Hp               int32      `json:"hp"`
	MaxHp            int32      `json:"maxHp"`
	ExpiresAt        int64      `json:"expiresAt"`
	WorldId          world.Id   `json:"worldId"`
	ChannelId        channel.Id `json:"channelId"`
	MapId            _map.Id    `json:"mapId"`
	Instance         uuid.UUID  `json:"instance"`
}

func (m RestModel) GetID() string {
	return m.Id
}

func (m *RestModel) SetID(idStr string) error {
	m.Id = idStr
	return nil
}

func (m RestModel) GetName() string {
	return "summons"
}

func Transform(m Model) (RestModel, error) {
	f := m.Field()
	return RestModel{
		Id:               strconv.Itoa(int(m.Id())),
		OwnerCharacterId: m.OwnerCharacterId(),
		SkillId:          m.SkillId(),
		SkillLevel:       m.SkillLevel(),
		SummonType:       string(m.SummonType()),
		MovementType:     byte(m.MovementType()),
		X:                m.X(),
		Y:                m.Y(),
		Hp:               m.Hp(),
		MaxHp:            m.MaxHp(),
		ExpiresAt:        m.ExpiresAt().UnixMilli(),
		WorldId:          f.WorldId(),
		ChannelId:        f.ChannelId(),
		MapId:            f.MapId(),
		Instance:         f.Instance(),
	}, nil
}
