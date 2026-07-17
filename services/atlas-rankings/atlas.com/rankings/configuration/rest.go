package configuration

import (
	"github.com/jtumidanski/api2go/jsonapi"
)

// RestModel is the rankings configuration resource served by atlas-tenants
// at /tenants/{tenantId}/configurations/rankings.
type RestModel struct {
	Id                       string `json:"-"`
	RecomputeIntervalMinutes uint32 `json:"recomputeIntervalMinutes"`
}

func (r RestModel) GetName() string {
	return "rankings"
}

func (r RestModel) GetID() string {
	return r.Id
}

func (r *RestModel) SetID(id string) error {
	r.Id = id
	return nil
}

// Relationship stubs — required by libs/atlas-rest/CLAUDE.md so every
// JSON:API target struct implements the full trio, matching character.RestModel
// (character/rest.go). Not currently exercised (the rankings resource carries
// no relationships block), but keeps this client consistent if it ever gains
// an `included` block.
func (r *RestModel) SetToOneReferenceID(_, _ string) error {
	return nil
}

func (r *RestModel) SetToManyReferenceIDs(_ string, _ []string) error {
	return nil
}

func (r *RestModel) SetReferencedStructs(_ map[string]map[string]jsonapi.Data) error {
	return nil
}
