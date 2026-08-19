package occurrence

import "encoding/json"

// VisualRestModel is the map-entry projection. It is intentionally narrow:
// the channel needs "what should I draw", not the occurrence's whole shape.
// The visual/bgm values are read out of the occurrence context, which the
// EVENT package populated — the generic layer copies them, it does not
// interpret them.
//
// No State/SubState here (nor on visualContext below): since B6
// (commit d19237c1e), ContiMove's wire state/subState bytes are resolved
// per-tenant from atlas-channel's writer-options table, keyed off the
// visual's Type (SHOW/HIDE) — there is no longer any source of truth for
// them on the events side to project.
type VisualRestModel struct {
	Id           string `json:"-"`
	OccurrenceId string `json:"occurrenceId"`
	Visual       string `json:"visual"`
	Bgm          string `json:"bgm"`
}

func (m VisualRestModel) GetName() string { return "event-visuals" }

func (m VisualRestModel) GetID() string { return m.Id }

// visualContext is the subset of an occurrence's context this projection
// reads out. Any other keys the EVENT package wrote into the context are
// irrelevant here and are ignored by json.Unmarshal.
//
// Shape matches what CRIMSON_BALROG — the only event type that currently
// populates visuals — actually writes (events/crimsonbalrog/config.go
// OccurrenceContext): `visual` is an object ({"name": ...}, VisualConfig)
// and the music key is `backgroundMusic`.
type visualContext struct {
	Visual          visualConfigContext `json:"visual"`
	BackgroundMusic string              `json:"backgroundMusic"`
}

// visualConfigContext mirrors crimsonbalrog.VisualConfig's wire shape.
type visualConfigContext struct {
	Name string `json:"name"`
}

// TransformVisual projects m into the narrow visual wire model (FR-API8,
// design §9.7).
func TransformVisual(m Model) (VisualRestModel, error) {
	var vc visualContext
	if len(m.Context()) > 0 {
		if err := json.Unmarshal(m.Context(), &vc); err != nil {
			return VisualRestModel{}, err
		}
	}
	return VisualRestModel{
		Id:           m.Id().String(),
		OccurrenceId: m.Id().String(),
		Visual:       vc.Visual.Name,
		Bgm:          vc.BackgroundMusic,
	}, nil
}
