package quest

import (
	"strconv"
	"time"
)

type ProgressRestModel struct {
	Id         uint32 `json:"-"`
	InfoNumber uint32 `json:"infoNumber"`
	Progress   string `json:"progress"`
}

type RestModel struct {
	Id             uint32              `json:"-"`
	CharacterId    uint32              `json:"characterId"`
	QuestId        uint32              `json:"questId"`
	State          State               `json:"state"`
	StartedAt      time.Time           `json:"startedAt"`
	CompletedAt    time.Time           `json:"completedAt,omitempty"`
	ExpirationTime time.Time           `json:"expirationTime,omitempty"`
	CompletedCount uint32              `json:"completedCount"`
	ForfeitCount   uint32              `json:"forfeitCount"`
	Progress       []ProgressRestModel `json:"progress"`
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

// Transform converts the domain Model into the wire RestModel.
func Transform(m Model) (RestModel, error) {
	ps := make([]ProgressRestModel, 0, len(m.progress))
	for _, p := range m.progress {
		ps = append(ps, ProgressRestModel{
			InfoNumber: p.InfoNumber(),
			Progress:   p.Progress(),
		})
	}

	return RestModel{
		Id:             m.id,
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
	ps := make([]Progress, 0, len(rm.Progress))
	for _, p := range rm.Progress {
		ps = append(ps, Progress{
			infoNumber: p.InfoNumber,
			progress:   p.Progress,
		})
	}

	return Model{
		id:             rm.Id,
		characterId:    rm.CharacterId,
		questId:        rm.QuestId,
		state:          rm.State,
		startedAt:      rm.StartedAt,
		completedAt:    rm.CompletedAt,
		expirationTime: rm.ExpirationTime,
		completedCount: rm.CompletedCount,
		forfeitCount:   rm.ForfeitCount,
		progress:       ps,
	}, nil
}
