package instance

import (
	"fmt"
	"time"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/google/uuid"
	"github.com/jtumidanski/api2go/jsonapi"
)

const Resource = "instances"

type CharacterEntryRestModel struct {
	CharacterId uint32     `json:"characterId"`
	WorldId     world.Id   `json:"worldId"`
	ChannelId   channel.Id `json:"channelId"`
}

type StageStateRestModel struct {
	ItemCounts   map[uint32]uint32 `json:"itemCounts,omitempty"`
	MonsterKills map[uint32]uint32 `json:"monsterKills,omitempty"`
	Combination  []uint32          `json:"combination,omitempty"`
	Attempts     uint32            `json:"attempts"`
	CustomData   map[string]any    `json:"customData,omitempty"`
}

type RestModel struct {
	Id                uuid.UUID                 `json:"-"`
	DefinitionId      uuid.UUID                 `json:"definitionId"`
	QuestId           string                    `json:"questId"`
	State             string                    `json:"state"`
	WorldId           world.Id                  `json:"worldId"`
	ChannelId         channel.Id                `json:"channelId"`
	PartyId           uint32                    `json:"partyId"`
	Characters        []CharacterEntryRestModel `json:"characters"`
	CurrentStageIndex uint32                    `json:"currentStageIndex"`
	StartedAt         time.Time                 `json:"startedAt"`
	StageStartedAt    time.Time                 `json:"stageStartedAt"`
	RegisteredAt      time.Time                 `json:"registeredAt"`
	FieldInstances    []uuid.UUID               `json:"fieldInstances"`
	StageState        StageStateRestModel       `json:"stageState"`
}

func (r RestModel) GetName() string {
	return Resource
}

func (r RestModel) GetID() string {
	return r.Id.String()
}

func (r *RestModel) SetID(idStr string) error {
	id, err := uuid.Parse(idStr)
	if err != nil {
		return fmt.Errorf("invalid instance ID: %w", err)
	}
	r.Id = id
	return nil
}

func (r RestModel) GetReferences() []jsonapi.Reference {
	return []jsonapi.Reference{}
}

func (r RestModel) GetReferencedIDs() []jsonapi.ReferenceID {
	return []jsonapi.ReferenceID{}
}

func (r RestModel) GetReferencedStructs() []jsonapi.MarshalIdentifier {
	return []jsonapi.MarshalIdentifier{}
}

func (r *RestModel) SetToOneReferenceID(_, _ string) error {
	return nil
}

func (r *RestModel) SetToManyReferenceIDs(_ string, _ []string) error {
	return nil
}

func (r *RestModel) SetReferencedStructs(_ map[string]map[string]jsonapi.Data) error {
	return nil
}

func Transform(m Model) (RestModel, error) {
	chars := make([]CharacterEntryRestModel, 0, len(m.Characters()))
	for _, c := range m.Characters() {
		chars = append(chars, CharacterEntryRestModel{
			CharacterId: c.CharacterId(),
			WorldId:     c.WorldId(),
			ChannelId:   c.ChannelId(),
		})
	}

	ss := m.StageState()
	return RestModel{
		Id:                m.Id(),
		DefinitionId:      m.DefinitionId(),
		QuestId:           m.QuestId(),
		State:             string(m.State()),
		WorldId:           m.WorldId(),
		ChannelId:         m.ChannelId(),
		PartyId:           m.PartyId(),
		Characters:        chars,
		CurrentStageIndex: m.CurrentStageIndex(),
		StartedAt:         m.StartedAt(),
		StageStartedAt:    m.StageStartedAt(),
		RegisteredAt:      m.RegisteredAt(),
		FieldInstances:    m.FieldInstances(),
		StageState: StageStateRestModel{
			ItemCounts:   ss.ItemCounts(),
			MonsterKills: ss.MonsterKills(),
			Combination:  ss.Combination(),
			Attempts:     ss.Attempts(),
			CustomData:   ss.CustomData(),
		},
	}, nil
}
