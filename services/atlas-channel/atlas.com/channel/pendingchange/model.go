package pendingchange

import (
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

// RestModel is the read-side JSON:API projection of a pending change record,
// as produced by atlas-character (services/atlas-character/atlas.com/character/pending_change/rest.go).
type RestModel struct {
	Id                 string     `json:"-"`
	CharacterId        uint32     `json:"characterId"`
	Type               string     `json:"type"`
	Status             string     `json:"status"`
	RequestedName      string     `json:"requestedName,omitempty"`
	DestinationWorldId world.Id   `json:"destinationWorldId"`
	SourceWorldId      world.Id   `json:"sourceWorldId"`
	Reason             string     `json:"reason,omitempty"`
	CreatedAt          time.Time  `json:"createdAt"`
	ExpiresAt          time.Time  `json:"expiresAt"`
	ResolvedAt         *time.Time `json:"resolvedAt,omitempty"`
}

func (r RestModel) GetName() string {
	return "pending-changes"
}

func (r RestModel) GetID() string {
	return r.Id
}

func (r *RestModel) SetID(id string) error {
	r.Id = id
	return nil
}

// CreateInputRestModel is the POST body atlas-character's
// POST /characters/{characterId}/pending-changes expects:
// {data:{type:"pending-changes",attributes:{type,requestedName,destinationWorldId,assetId}}}.
//
// AssetId is an item TEMPLATE id, not an instance id (task-6/task-7 seam;
// see the atlas-character doc comment on the same field).
type CreateInputRestModel struct {
	Id                 string   `json:"-"`
	Type               string   `json:"type"`
	RequestedName      string   `json:"requestedName,omitempty"`
	DestinationWorldId world.Id `json:"destinationWorldId,omitempty"`
	AssetId            *uint32  `json:"assetId,omitempty"`
}

func (r CreateInputRestModel) GetName() string {
	return "pending-changes"
}

func (r CreateInputRestModel) GetID() string {
	return r.Id
}

func (r *CreateInputRestModel) SetID(id string) error {
	r.Id = id
	return nil
}

// CancelInputRestModel is the POST body atlas-character's self-scoped
// POST /characters/{characterId}/pending-changes/cancel expects:
// {data:{type:"pending-changes",attributes:{type}}}. It carries no record
// id -- the wire packet that drives this call has none (task-227
// client-cancel addendum); atlas-character resolves the record by
// (characterId, type) instead.
type CancelInputRestModel struct {
	Id   string `json:"-"`
	Type string `json:"type"`
}

func (r CancelInputRestModel) GetName() string {
	return "pending-changes"
}

func (r CancelInputRestModel) GetID() string {
	return r.Id
}

func (r *CancelInputRestModel) SetID(id string) error {
	r.Id = id
	return nil
}
