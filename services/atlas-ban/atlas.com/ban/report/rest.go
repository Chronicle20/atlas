package report

import (
	"time"

	"github.com/google/uuid"
)

type RestModel struct {
	Id               uuid.UUID        `json:"-"`
	Kind             string           `json:"kind"`
	ReporterId       uint32           `json:"reporterId"`
	ReporterName     string           `json:"reporterName"`
	AccusedId        uint32           `json:"accusedId"`
	AccusedName      string           `json:"accusedName"`
	ReasonType       byte             `json:"reasonType"`
	Description      string           `json:"description"`
	ChatLog          *string          `json:"chatLog"`
	ServerTranscript []TranscriptLine `json:"serverTranscript"`
	Status           string           `json:"status"`
	CreatedAt        time.Time        `json:"createdAt"`
	UpdatedAt        time.Time        `json:"updatedAt"`
}

func (r RestModel) GetName() string {
	return "reports"
}

func (r RestModel) GetID() string {
	return r.Id.String()
}

func (r *RestModel) SetID(idStr string) error {
	id, err := uuid.Parse(idStr)
	if err != nil {
		return err
	}
	r.Id = id
	return nil
}

func Transform(m Model) (RestModel, error) {
	return RestModel{
		Id:               m.Id(),
		Kind:             string(m.Kind()),
		ReporterId:       m.ReporterId(),
		ReporterName:     m.ReporterName(),
		AccusedId:        m.AccusedId(),
		AccusedName:      m.AccusedName(),
		ReasonType:       m.ReasonType(),
		Description:      m.Description(),
		ChatLog:          m.ChatLog(),
		ServerTranscript: m.ServerTranscript(),
		Status:           string(m.Status()),
		CreatedAt:        m.CreatedAt(),
		UpdatedAt:        m.UpdatedAt(),
	}, nil
}

// TransformSlice maps a slice of domain Models to their REST projections.
// Returns the first transform error encountered, if any.
func TransformSlice(ms []Model) ([]RestModel, error) {
	out := make([]RestModel, 0, len(ms))
	for _, m := range ms {
		rm, err := Transform(m)
		if err != nil {
			return nil, err
		}
		out = append(out, rm)
	}
	return out, nil
}
