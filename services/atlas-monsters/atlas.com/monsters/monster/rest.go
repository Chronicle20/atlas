package monster

import (
	"strconv"

	"github.com/Chronicle20/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
)

type RestModel struct {
	Id                 string              `json:"-"`
	WorldId            world.Id            `json:"worldId"`
	ChannelId          channel.Id          `json:"channelId"`
	MapId              _map.Id             `json:"mapId"`
	Instance           uuid.UUID           `json:"instance"`
	MonsterId          uint32              `json:"monsterId"`
	ControlCharacterId uint32              `json:"controlCharacterId"`
	X                  int16               `json:"x"`
	Y                  int16               `json:"y"`
	Fh                 int16               `json:"fh"`
	Stance             byte                `json:"stance"`
	Team               int8                `json:"team"`
	MaxHp              uint32              `json:"maxHp"`
	Hp                 uint32              `json:"hp"`
	MaxMp              uint32              `json:"maxMp"`
	Mp                 uint32              `json:"mp"`
	DamageEntries      []DamageEntry       `json:"damageEntries"`
	StatusEffects      []StatusEffectEntry `json:"statusEffects"`
}

type StatusEffectEntry struct {
	SourceSkillId    uint32           `json:"sourceSkillId"`
	SourceSkillLevel uint32           `json:"sourceSkillLevel"`
	Statuses         map[string]int32 `json:"statuses"`
	ExpiresAt        int64            `json:"expiresAt"`
}

type DamageEntry struct {
	CharacterId uint32 `json:"characterId"`
	Damage      uint32 `json:"damage"`
}

func (m RestModel) GetID() string {
	return m.Id
}

func (m *RestModel) SetID(idStr string) error {
	m.Id = idStr
	return nil
}

func (m RestModel) GetName() string {
	return "monsters"
}

func Transform(m Model) (RestModel, error) {
	des, err := model.SliceMap(TransformDamageEntry)(model.FixedProvider(m.damageEntries))(model.ParallelMap())()
	if err != nil {
		return RestModel{}, err
	}

	ses := make([]StatusEffectEntry, 0, len(m.statusEffects))
	for _, se := range m.statusEffects {
		ses = append(ses, StatusEffectEntry{
			SourceSkillId:    se.sourceSkillId,
			SourceSkillLevel: se.sourceSkillLevel,
			Statuses:         se.statuses,
			ExpiresAt:        se.expiresAt.UnixMilli(),
		})
	}

	return RestModel{
		Id:                 strconv.Itoa(int(m.UniqueId())),
		WorldId:            m.worldId,
		ChannelId:          m.channelId,
		MapId:              m.mapId,
		Instance:           m.instance,
		MonsterId:          m.monsterId,
		ControlCharacterId: m.controlCharacterId,
		X:                  m.x,
		Y:                  m.y,
		Fh:                 m.fh,
		Stance:             m.stance,
		Team:               m.team,
		MaxHp:              m.maxHp,
		Hp:                 m.hp,
		MaxMp:              m.maxMp,
		Mp:                 m.mp,
		DamageEntries:      des,
		StatusEffects:      ses,
	}, nil
}

func TransformDamageEntry(m entry) (DamageEntry, error) {
	return DamageEntry{
		CharacterId: m.CharacterId,
		Damage:      m.Damage,
	}, nil
}
