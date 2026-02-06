package npc

import (
	"atlas-npc-conversations/conversation"
	"fmt"

	"github.com/google/uuid"
	"github.com/jtumidanski/api2go/jsonapi"
)

const (
	Resource = "conversations"
)

// RestModel represents the REST model for NPC conversations
type RestModel struct {
	Id         uuid.UUID                     `json:"-"`          // Conversation ID
	NpcId      uint32                        `json:"npcId"`      // NPC ID
	StartState string                        `json:"startState"` // Start state ID
	States     []conversation.RestStateModel `json:"states"`     // Conversation states
}

// GetName returns the resource name
func (r RestModel) GetName() string {
	return Resource
}

// GetID returns the resource ID
func (r RestModel) GetID() string {
	return r.Id.String()
}

// SetID sets the resource ID
func (r *RestModel) SetID(idStr string) error {
	id, err := uuid.Parse(idStr)
	if err != nil {
		return fmt.Errorf("invalid conversation ID: %w", err)
	}
	r.Id = id
	return nil
}

// GetReferences returns the resource references
func (r RestModel) GetReferences() []jsonapi.Reference {
	return []jsonapi.Reference{}
}

// GetReferencedIDs returns the referenced IDs
func (r RestModel) GetReferencedIDs() []jsonapi.ReferenceID {
	return []jsonapi.ReferenceID{}
}

// GetReferencedStructs returns the referenced structs
func (r RestModel) GetReferencedStructs() []jsonapi.MarshalIdentifier {
	return []jsonapi.MarshalIdentifier{}
}

// SetToOneReferenceID sets a to-one reference ID
func (r *RestModel) SetToOneReferenceID(_, _ string) error {
	return nil
}

// SetToManyReferenceIDs sets to-many reference IDs
func (r *RestModel) SetToManyReferenceIDs(_ string, _ []string) error {
	return nil
}

// SetReferencedStructs sets referenced structs
func (r *RestModel) SetReferencedStructs(_ map[string]map[string]jsonapi.Data) error {
	return nil
}

// Transform converts a domain model to a REST model
func Transform(m Model) (RestModel, error) {
	// Transform states
	restStates := make([]conversation.RestStateModel, 0, len(m.States()))
	for _, state := range m.States() {
		restState, err := conversation.TransformState(state)
		if err != nil {
			return RestModel{}, err
		}
		restStates = append(restStates, restState)
	}

	return RestModel{
		Id:         m.Id(),
		NpcId:      m.NpcId(),
		StartState: m.StartState(),
		States:     restStates,
	}, nil
}

// Extract converts a REST model to a domain model
func Extract(r RestModel) (Model, error) {
	// Validate required fields
	if r.NpcId == 0 {
		return Model{}, fmt.Errorf("npcId is required")
	}
	if r.StartState == "" {
		return Model{}, fmt.Errorf("startState is required")
	}
	if len(r.States) == 0 {
		return Model{}, fmt.Errorf("states are required")
	}

	// Create a new model using the builder
	builder := NewBuilder()

	// Set ID if provided, otherwise it will be auto-generated
	if r.Id != uuid.Nil {
		builder.SetId(r.Id)
	}

	builder.SetNpcId(r.NpcId).
		SetStartState(r.StartState)

	// Extract states
	for _, restState := range r.States {
		state, err := conversation.ExtractState(restState)
		if err != nil {
			return Model{}, err
		}
		builder.AddState(state)
	}

	return builder.Build()
}
