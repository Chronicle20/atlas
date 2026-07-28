package job

import (
	"atlas-data/skill"
	"strconv"

	"github.com/jtumidanski/api2go/jsonapi"
)

// ListRestModel is the GET /data/jobs projection of RestModel. It is NEVER
// persisted: document.DbStorage.Add marshals the model type it is handed
// (document/db_storage.go:123), and api2go writes `relationships` — and
// `included`, once any referenced struct is attached — for any model
// implementing the relationship interfaces (jsonapi/marshal.go:186-208).
// Keeping the persisted type (RestModel) relationship-free is what stops that
// leaking into the stored `content` column (FR-4.4, design D2), and what keeps
// GET /data/jobs/{jobId}/skills at the shape PRD §5 pins as unchanged.
//
// It deliberately implements only the marshal side. The Unmarshal* /
// SetToManyReferenceIDs / SetReferencedStructs counterparts that
// shops.RestModel carries are omitted: nothing ever unmarshals this type, and
// adding them would imply it is persistable or inbound, which it is not.
type ListRestModel struct {
	Id     uint32   `json:"-"`
	Skills []uint32 `json:"skills"`
	// resolved holds the full skill resources emitted into `included`. It is
	// unexported, so encoding/json never sees it; it is populated only when the
	// request asked for include=skills (design D3).
	resolved []skill.RestModel
}

func (r ListRestModel) GetID() string   { return strconv.Itoa(int(r.Id)) }
func (r ListRestModel) GetName() string { return "jobs" }

// GetReferences to satisfy jsonapi.MarshalReferences.
func (r ListRestModel) GetReferences() []jsonapi.Reference {
	return []jsonapi.Reference{
		{
			Type: "skills",
			Name: "skills",
		},
	}
}

// GetReferencedIDs to satisfy jsonapi.MarshalLinkedRelations. Derived from the
// stored id list, so linkage is ALWAYS present (FR-4.3) — a deliberate
// divergence from the `shops` reference implementation, where linkage itself is
// a function of the include decorator (shops/resource.go:75-84).
func (r ListRestModel) GetReferencedIDs() []jsonapi.ReferenceID {
	result := make([]jsonapi.ReferenceID, 0, len(r.Skills))
	for _, id := range r.Skills {
		result = append(result, jsonapi.ReferenceID{
			ID:   strconv.Itoa(int(id)),
			Type: "skills",
			Name: "skills",
		})
	}
	return result
}

// GetReferencedStructs to satisfy jsonapi.MarshalIncludedRelations. Empty
// unless the handler resolved skills, and api2go omits the `included` key
// entirely when nothing is returned (jsonapi/marshal.go:203).
func (r ListRestModel) GetReferencedStructs() []jsonapi.MarshalIdentifier {
	result := make([]jsonapi.MarshalIdentifier, 0, len(r.resolved))
	for _, s := range r.resolved {
		result = append(result, s)
	}
	return result
}

// ListFrom is the only bridge from the persisted type to the list projection.
func ListFrom(m RestModel) ListRestModel {
	return ListRestModel{Id: m.Id, Skills: m.Skills}
}

func ListFromAll(ms []RestModel) []ListRestModel {
	out := make([]ListRestModel, 0, len(ms))
	for _, m := range ms {
		out = append(out, ListFrom(m))
	}
	return out
}

// WithResolvedSkills attaches the full skill resources for each job's id list.
// Ids with no matching skill document are skipped — the linkage still names
// them, which is the JSON:API-correct way to express "referenced but not
// included".
func WithResolvedSkills(items []ListRestModel, byId map[uint32]skill.RestModel) []ListRestModel {
	out := make([]ListRestModel, 0, len(items))
	for _, it := range items {
		resolved := make([]skill.RestModel, 0, len(it.Skills))
		for _, id := range it.Skills {
			if s, ok := byId[id]; ok {
				resolved = append(resolved, s)
			}
		}
		it.resolved = resolved
		out = append(out, it)
	}
	return out
}
