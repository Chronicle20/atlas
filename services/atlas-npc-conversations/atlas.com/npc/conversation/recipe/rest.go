package recipe

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/jtumidanski/api2go/jsonapi"
)

const (
	Resource        = "recipes"
	ReindexResource = "recipeReindexResults"
)

// RestMaterial is the JSON:API representation of one material entry.
type RestMaterial struct {
	ItemId   uint32 `json:"itemId"`
	Quantity uint32 `json:"quantity"`
}

// RestModel is the JSON:API representation of one recipe row.
type RestModel struct {
	Id                   uuid.UUID      `json:"-"`
	NpcId                uint32         `json:"npcId"`
	ConversationId       uuid.UUID      `json:"conversationId"`
	StateId              string         `json:"stateId"`
	ItemId               uint32         `json:"itemId"`
	Materials            []RestMaterial `json:"materials"`
	MesoCost             uint32         `json:"mesoCost"`
	StimulatorId         uint32         `json:"stimulatorId"`
	StimulatorFailChance float64        `json:"stimulatorFailChance"`
}

func (r RestModel) GetName() string  { return Resource }
func (r RestModel) GetID() string    { return r.Id.String() }
func (r *RestModel) SetID(idStr string) error {
	id, err := uuid.Parse(idStr)
	if err != nil {
		return fmt.Errorf("invalid recipe ID: %w", err)
	}
	r.Id = id
	return nil
}
func (r RestModel) GetReferences() []jsonapi.Reference                { return []jsonapi.Reference{} }
func (r RestModel) GetReferencedIDs() []jsonapi.ReferenceID           { return []jsonapi.ReferenceID{} }
func (r RestModel) GetReferencedStructs() []jsonapi.MarshalIdentifier { return []jsonapi.MarshalIdentifier{} }
func (r *RestModel) SetToOneReferenceID(_, _ string) error            { return nil }
func (r *RestModel) SetToManyReferenceIDs(_ string, _ []string) error { return nil }
func (r *RestModel) SetReferencedStructs(_ map[string]map[string]jsonapi.Data) error {
	return nil
}

// Transform converts a Model to the wire-format RestModel.
func Transform(m Model) RestModel {
	mats := make([]RestMaterial, 0, len(m.Materials()))
	for _, mat := range m.Materials() {
		mats = append(mats, RestMaterial{ItemId: mat.ItemId, Quantity: mat.Quantity})
	}
	return RestModel{
		Id:                   m.Id(),
		NpcId:                m.NpcId(),
		ConversationId:       m.ConversationId(),
		StateId:              m.StateId(),
		ItemId:               m.ItemId(),
		Materials:            mats,
		MesoCost:             m.MesoCost(),
		StimulatorId:         m.StimulatorId(),
		StimulatorFailChance: m.StimulatorFailChance(),
	}
}

// RestReindexResult is the JSON:API representation of a reindex run.
type RestReindexResult struct {
	Id                   string          `json:"-"`
	DeletedCount         int64           `json:"deletedCount"`
	InsertedCount        int             `json:"insertedCount"`
	SkippedCount         int             `json:"skippedCount"`
	SkippedDetails       []SkippedRecipe `json:"skippedDetails,omitempty"`
	ConversationsScanned int             `json:"conversationsScanned"`
}

func (r RestReindexResult) GetName() string  { return ReindexResource }
func (r RestReindexResult) GetID() string    { return r.Id }
func (r *RestReindexResult) SetID(id string) error { r.Id = id; return nil }
func (r RestReindexResult) GetReferences() []jsonapi.Reference                { return []jsonapi.Reference{} }
func (r RestReindexResult) GetReferencedIDs() []jsonapi.ReferenceID           { return []jsonapi.ReferenceID{} }
func (r RestReindexResult) GetReferencedStructs() []jsonapi.MarshalIdentifier { return []jsonapi.MarshalIdentifier{} }
func (r *RestReindexResult) SetToOneReferenceID(_, _ string) error            { return nil }
func (r *RestReindexResult) SetToManyReferenceIDs(_ string, _ []string) error { return nil }
func (r *RestReindexResult) SetReferencedStructs(_ map[string]map[string]jsonapi.Data) error {
	return nil
}

// MakeRestReindexResult builds the rest payload from an internal ReindexResult,
// using the tenant id as the synthetic resource id.
func MakeRestReindexResult(tenantId uuid.UUID, r ReindexResult) RestReindexResult {
	return RestReindexResult{
		Id:                   tenantId.String(),
		DeletedCount:         r.DeletedCount,
		InsertedCount:        r.InsertedCount,
		SkippedCount:         r.SkippedCount,
		SkippedDetails:       r.SkippedDetails,
		ConversationsScanned: r.ConversationsScanned,
	}
}
