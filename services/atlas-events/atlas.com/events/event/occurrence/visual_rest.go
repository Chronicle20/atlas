package occurrence

import "encoding/json"

// VisualRestModel is the map-entry projection. It is intentionally narrow: the
// channel needs "what should I draw", not the occurrence's whole shape. The
// visual/state/subState/bgm values are read out of the occurrence context,
// which the EVENT package populated — the generic layer copies them, it does
// not interpret them.
type VisualRestModel struct {
	Id           string `json:"-"`
	OccurrenceId string `json:"occurrenceId"`
	Visual       string `json:"visual"`
	State        byte   `json:"state"`
	SubState     byte   `json:"subState"`
	Bgm          string `json:"bgm"`
}

func (m VisualRestModel) GetName() string { return "event-visuals" }

func (m VisualRestModel) GetID() string { return m.Id }

// visualContext is the subset of an occurrence's context this projection
// reads out. Any other keys the EVENT package wrote into the context are
// irrelevant here and are ignored by json.Unmarshal.
type visualContext struct {
	Visual   string `json:"visual"`
	State    byte   `json:"state"`
	SubState byte   `json:"subState"`
	Bgm      string `json:"bgm"`
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
		Visual:       vc.Visual,
		State:        vc.State,
		SubState:     vc.SubState,
		Bgm:          vc.Bgm,
	}, nil
}
