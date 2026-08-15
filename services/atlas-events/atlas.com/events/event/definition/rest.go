package definition

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jtumidanski/api2go/jsonapi"
)

// Resource is the JSON:API resource type name for an event definition.
const Resource = "event-definitions"

// RestModel is the wire representation of an event definition (FR-API3).
// Configuration stays opaque JSON — this layer never interprets its shape.
type RestModel struct {
	Id      uuid.UUID       `json:"-"`
	Type    string          `json:"type"`
	Name    string          `json:"name"`
	Enabled bool            `json:"enabled"`
	Config  json.RawMessage `json:"configuration"`

	// SingleOccurrence tells the UI how to render this row (FR-UI4) without
	// switching on type: where the handler's concurrency key is a constant, at
	// most one occurrence can exist and the row may show live occurrence state;
	// where it varies, the row must link to the filtered occurrence list
	// instead of implying a single state.
	SingleOccurrence bool `json:"singleOccurrence"`

	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
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
		return fmt.Errorf("invalid event definition id: %w", err)
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

// Transform builds the wire model for m, deriving SingleOccurrence via the
// processor-owned handler probe (FR-UI4) rather than switching on m.Type()
// here in the REST layer.
func Transform(ctx context.Context, m Model) (RestModel, error) {
	return RestModel{
		Id:               m.Id(),
		Type:             m.Type(),
		Name:             m.Name(),
		Enabled:          m.Enabled(),
		Config:           m.Configuration(),
		SingleOccurrence: singleOccurrence(ctx, m.Type()),
		CreatedAt:        m.CreatedAt(),
		UpdatedAt:        m.UpdatedAt(),
	}, nil
}

// Extract builds a domain Model from a REST create body.
func Extract(r RestModel) (Model, error) {
	if r.Type == "" {
		return Model{}, fmt.Errorf("type is required")
	}
	if r.Name == "" {
		return Model{}, fmt.Errorf("name is required")
	}

	builder := NewBuilder(r.Type, r.Name).
		SetEnabled(r.Enabled).
		SetConfiguration(r.Config)
	if r.Id != uuid.Nil {
		builder.SetId(r.Id)
	}

	return builder.Build()
}
