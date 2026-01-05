package quest

import (
	"atlas-quest/quest/progress"
	"strconv"
	"time"

	"github.com/google/uuid"
)

type RestModel struct {
	Id             uint32               `json:"-"`
	TenantId       uuid.UUID            `json:"-"`
	CharacterId    uint32               `json:"characterId"`
	QuestId        uint32               `json:"questId"`
	State          State                `json:"state"`
	StartedAt      time.Time            `json:"startedAt"`
	CompletedAt    time.Time            `json:"completedAt,omitempty"`
	ExpirationTime time.Time            `json:"expirationTime,omitempty"`
	CompletedCount uint32               `json:"completedCount"`
	ForfeitCount   uint32               `json:"forfeitCount"`
	Progress       []progress.RestModel `json:"progress"`
}

// CompleteQuestResponseRestModel is returned when completing a quest that is part of a chain
type CompleteQuestResponseRestModel struct {
	NextQuestId uint32 `json:"nextQuestId"`
}

func (r CompleteQuestResponseRestModel) GetName() string {
	return "complete-quest-response"
}

func (r CompleteQuestResponseRestModel) GetID() string {
	return "0"
}

// ValidationFailedRestModel is returned when validation fails
type ValidationFailedRestModel struct {
	FailedConditions []string `json:"failedConditions"`
}

func (r ValidationFailedRestModel) GetName() string {
	return "validation-failed"
}

func (r ValidationFailedRestModel) GetID() string {
	return "0"
}

// StartQuestInputRestModel is the input for starting a quest
type StartQuestInputRestModel struct {
	Id             string `json:"-"`
	WorldId        byte   `json:"worldId"`
	ChannelId      byte   `json:"channelId"`
	MapId          uint32 `json:"mapId"`
	SkipValidation bool   `json:"skipValidation"`
}

func (r StartQuestInputRestModel) GetName() string {
	return "start-quest-input"
}

func (r StartQuestInputRestModel) GetID() string {
	return r.Id
}

func (r *StartQuestInputRestModel) SetID(id string) error {
	r.Id = id
	return nil
}

// CompleteQuestInputRestModel is the input for completing a quest
type CompleteQuestInputRestModel struct {
	Id             string `json:"-"`
	WorldId        byte   `json:"worldId"`
	ChannelId      byte   `json:"channelId"`
	MapId          uint32 `json:"mapId"`
	SkipValidation bool   `json:"skipValidation"`
}

func (r CompleteQuestInputRestModel) GetName() string {
	return "complete-quest-input"
}

func (r CompleteQuestInputRestModel) GetID() string {
	return r.Id
}

func (r *CompleteQuestInputRestModel) SetID(id string) error {
	r.Id = id
	return nil
}

func (r RestModel) GetName() string {
	return "quest-status"
}

func (r RestModel) GetID() string {
	return strconv.Itoa(int(r.Id))
}

func (r *RestModel) SetID(strId string) error {
	if strId == "" {
		r.Id = 0
		return nil
	}

	id, err := strconv.Atoi(strId)
	if err != nil {
		return err
	}
	r.Id = uint32(id)
	return nil
}

func Transform(m Model) (RestModel, error) {
	ps := make([]progress.RestModel, 0)
	for _, pm := range m.progress {
		rp, err := progress.Transform(pm)
		if err != nil {
			return RestModel{}, err
		}
		ps = append(ps, rp)
	}

	return RestModel{
		Id:             m.id,
		TenantId:       m.tenantId,
		CharacterId:    m.characterId,
		QuestId:        m.questId,
		State:          m.state,
		StartedAt:      m.startedAt,
		CompletedAt:    m.completedAt,
		ExpirationTime: m.expirationTime,
		CompletedCount: m.completedCount,
		ForfeitCount:   m.forfeitCount,
		Progress:       ps,
	}, nil
}

func Extract(rm RestModel) (Model, error) {
	return Model{
		tenantId:       rm.TenantId,
		id:             rm.Id,
		characterId:    rm.CharacterId,
		questId:        rm.QuestId,
		state:          rm.State,
		startedAt:      rm.StartedAt,
		completedAt:    rm.CompletedAt,
		expirationTime: rm.ExpirationTime,
		completedCount: rm.CompletedCount,
		forfeitCount:   rm.ForfeitCount,
		progress:       nil,
	}, nil
}
