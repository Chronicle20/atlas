package configuration

// RestModel is the player-npcs configuration resource served by
// atlas-tenants at /tenants/{tenantId}/configurations/player-npcs
// (design §9.4). The field names below are the FR-4.7 contract Task 20
// registers atlas-tenants against.
type RestModel struct {
	Id                string `json:"-"`
	InitialX          int16  `json:"initialX"`
	InitialY          int16  `json:"initialY"`
	AreaX             int16  `json:"areaX"`
	AreaY             int16  `json:"areaY"`
	AreaSteps         int    `json:"areaSteps"`
	OrganizeArea      bool   `json:"organizeArea"`
	AutoDeployEnabled bool   `json:"autoDeployEnabled"`
}

func (r RestModel) GetName() string {
	return "player-npcs"
}

func (r RestModel) GetID() string {
	return r.Id
}

func (r *RestModel) SetID(id string) error {
	r.Id = id
	return nil
}

// SetToOneReferenceID and SetToManyReferenceIDs are required even though
// this client no longer requests any relationship — see
// libs/atlas-rest/CLAUDE.md: api2go errors decoding any resource whose
// response carries a relationships block unless the target struct
// implements these, whether or not the caller cares about the data.
func (r *RestModel) SetToOneReferenceID(_, _ string) error {
	return nil
}

func (r *RestModel) SetToManyReferenceIDs(_ string, _ []string) error {
	return nil
}

// DefaultModel is the FR-4.7 fallback: applied when a tenant has no
// player-npcs configuration (404 — the expected unconfigured state) or any
// other read error.
func DefaultModel() Model {
	return Model{
		initialX:          262,
		initialY:          262,
		areaX:             320,
		areaY:             160,
		areaSteps:         4,
		organizeArea:      true,
		autoDeployEnabled: true,
	}
}

func Extract(r RestModel) (Model, error) {
	return Model{
		initialX:          r.InitialX,
		initialY:          r.InitialY,
		areaX:             r.AreaX,
		areaY:             r.AreaY,
		areaSteps:         r.AreaSteps,
		organizeArea:      r.OrganizeArea,
		autoDeployEnabled: r.AutoDeployEnabled,
	}, nil
}
