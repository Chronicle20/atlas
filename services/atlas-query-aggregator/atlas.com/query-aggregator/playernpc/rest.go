package playernpc

// EligibilityRestModel is the plain (non-JSON:API) body atlas-player-npcs'
// GET /player-npcs/eligibility returns
// (services/atlas-player-npcs/atlas.com/player-npcs/playernpc/rest.go's
// eligibilityResponse). It is deliberately not a jsonapi.MarshalIdentifier:
// the endpoint it decodes was designed as a plain body for exactly this
// kind of predicate lookup (design §9.1), so decoding it as JSON:API would
// simply fail.
type EligibilityRestModel struct {
	Eligible bool   `json:"eligible"`
	Reason   string `json:"reason"`
}
