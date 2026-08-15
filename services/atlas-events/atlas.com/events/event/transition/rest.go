package transition

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jtumidanski/api2go/jsonapi"
)

// Resource is the JSON:API resource type name for an occurrence transition.
const Resource = "event-occurrence-transitions"

// RestModel is the wire representation of an occurrence stage transition.
// It is read-only — transitions are only ever written inside the occurrence
// administrator's paired transaction (Task 15).
type RestModel struct {
	Id               uuid.UUID `json:"-"`
	OccurrenceId     uuid.UUID `json:"occurrenceId"`
	FromStage        string    `json:"fromStage"`
	ToStage          string    `json:"toStage"`
	OccurredAt       time.Time `json:"occurredAt"`
	TriggerType      string    `json:"triggerType"`
	TriggerReference string    `json:"triggerReference"`
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
		return fmt.Errorf("invalid event occurrence transition id: %w", err)
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

// Transform builds the wire model for m.
func Transform(m Model) (RestModel, error) {
	return RestModel{
		Id:               m.Id(),
		OccurrenceId:     m.OccurrenceId(),
		FromStage:        m.FromStage(),
		ToStage:          m.ToStage(),
		OccurredAt:       m.OccurredAt(),
		TriggerType:      m.TriggerType(),
		TriggerReference: m.TriggerReference(),
	}, nil
}
