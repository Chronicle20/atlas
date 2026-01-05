package quest

import (
	"atlas-quest/quest/progress"
	"strconv"
	"time"

	"github.com/google/uuid"
)

type RestModel struct {
	Id          uint32               `json:"-"`
	TenantId    uuid.UUID            `json:"-"`
	CharacterId uint32               `json:"characterId"`
	QuestId     uint32               `json:"questId"`
	State       State                `json:"state"`
	StartedAt   time.Time            `json:"startedAt"`
	CompletedAt time.Time            `json:"completedAt,omitempty"`
	Progress    []progress.RestModel `json:"progress"`
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
		Id:          m.id,
		TenantId:    m.tenantId,
		CharacterId: m.characterId,
		QuestId:     m.questId,
		State:       m.state,
		StartedAt:   m.startedAt,
		CompletedAt: m.completedAt,
		Progress:    ps,
	}, nil
}

func Extract(rm RestModel) (Model, error) {
	return Model{
		tenantId:    rm.TenantId,
		id:          rm.Id,
		characterId: rm.CharacterId,
		questId:     rm.QuestId,
		state:       rm.State,
		startedAt:   rm.StartedAt,
		completedAt: rm.CompletedAt,
		progress:    nil,
	}, nil
}
