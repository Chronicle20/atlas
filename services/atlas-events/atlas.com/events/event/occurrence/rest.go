package occurrence

import (
	"atlas-events/event/definition"
	"atlas-events/event/transition"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jtumidanski/api2go/jsonapi"
)

// Resource is the JSON:API resource type name for an event occurrence.
const Resource = "event-occurrences"

// RestModel is the wire representation of a live event occurrence (FR-API4).
// Context stays opaque JSON — this layer never interprets its shape.
type RestModel struct {
	Id               uuid.UUID       `json:"-"`
	Type             string          `json:"type"`
	State            string          `json:"state"`
	Stage            string          `json:"stage"`
	Context          json.RawMessage `json:"context"`
	StartedAt        time.Time       `json:"startedAt"`
	NextTransitionAt *time.Time      `json:"nextTransitionAt,omitempty"`
	CompletedAt      *time.Time      `json:"completedAt,omitempty"`
	CompletionReason string          `json:"completionReason,omitempty"`

	DefinitionId uuid.UUID `json:"-"`

	// Transitions is populated only by TransformWithTransitions, for
	// GET /events/occurrences/{id} (FR-API5). The collection listing never
	// loads per-row transition history.
	Transitions []transition.RestModel `json:"-"`
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
		return fmt.Errorf("invalid event occurrence id: %w", err)
	}
	r.Id = id
	return nil
}

func (r RestModel) GetReferences() []jsonapi.Reference {
	return []jsonapi.Reference{
		{Type: definition.Resource, Name: "definition", Relationship: jsonapi.ToOneRelationship},
		{Type: transition.Resource, Name: "transitions", Relationship: jsonapi.ToManyRelationship},
	}
}

func (r RestModel) GetReferencedIDs() []jsonapi.ReferenceID {
	result := []jsonapi.ReferenceID{
		{ID: r.DefinitionId.String(), Type: definition.Resource, Name: "definition"},
	}
	for _, t := range r.Transitions {
		result = append(result, jsonapi.ReferenceID{ID: t.GetID(), Type: transition.Resource, Name: "transitions"})
	}
	return result
}

func (r RestModel) GetReferencedStructs() []jsonapi.MarshalIdentifier {
	result := make([]jsonapi.MarshalIdentifier, 0, len(r.Transitions))
	for _, t := range r.Transitions {
		result = append(result, t)
	}
	return result
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

// Transform builds the wire model for m without transition history — used by
// the collection listing (FR-API6).
func Transform(m Model) (RestModel, error) {
	return RestModel{
		Id:               m.Id(),
		Type:             m.Type(),
		State:            m.State(),
		Stage:            m.Stage(),
		Context:          m.Context(),
		StartedAt:        m.StartedAt(),
		NextTransitionAt: m.NextTransitionAt(),
		CompletedAt:      m.CompletedAt(),
		CompletionReason: m.CompletionReason(),
		DefinitionId:     m.DefinitionId(),
	}, nil
}

// TransformWithTransitions builds the wire model for m plus its transition
// history as an included relationship (FR-API5) — used only by
// GET /events/occurrences/{id}.
func TransformWithTransitions(m Model, transitions []transition.RestModel) (RestModel, error) {
	rm, err := Transform(m)
	if err != nil {
		return RestModel{}, err
	}
	rm.Transitions = transitions
	return rm, nil
}
