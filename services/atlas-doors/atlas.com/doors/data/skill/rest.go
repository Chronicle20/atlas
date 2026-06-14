package skill

import (
	"atlas-doors/data/skill/effect"
	"strconv"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

// RestModel mirrors the subset of atlas-data's skill wire format that
// atlas-doors needs: the id and the per-level effects array.
type RestModel struct {
	Id      uint32             `json:"-"`
	Effects []effect.RestModel `json:"effects"`
}

func (r RestModel) GetName() string {
	return "skills"
}

func (r RestModel) GetID() string {
	return strconv.Itoa(int(r.Id))
}

func (r *RestModel) SetID(idStr string) error {
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return err
	}
	r.Id = uint32(id)
	return nil
}

// Extract converts a RestModel into an immutable Model by extracting each
// per-level effect.
// Exported so tests can call it directly without network I/O.
func Extract(rm RestModel) (Model, error) {
	es, err := model.SliceMap(effect.Extract)(model.FixedProvider(rm.Effects))()()
	if err != nil {
		return Model{}, err
	}
	return Model{
		id:      rm.Id,
		effects: es,
	}, nil
}
