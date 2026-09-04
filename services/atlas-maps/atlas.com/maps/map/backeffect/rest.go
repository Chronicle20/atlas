package backeffect

import (
	"strconv"

	beconst "github.com/Chronicle20/atlas/libs/atlas-constants/backeffect"
)

type RestModel struct {
	Id       string         `json:"-"`
	Effect   beconst.Effect `json:"effect"`
	FieldId  uint32         `json:"fieldId"`
	PageId   uint8          `json:"pageId"`
	Duration uint32         `json:"duration"`
}

func (m RestModel) GetID() string {
	return m.Id
}

func (m RestModel) GetName() string {
	return "backEffect"
}

func (m *RestModel) SetID(idStr string) error {
	m.Id = idStr
	return nil
}

func Transform(e BackEffectEntry) (RestModel, error) {
	return RestModel{
		Id:       strconv.Itoa(int(e.PageId)),
		Effect:   e.Effect,
		FieldId:  e.FieldId,
		PageId:   uint8(e.PageId),
		Duration: e.Duration,
	}, nil
}

// TransformSlice maps a slice of domain BackEffectEntry values to their REST
// projections. Returns the first transform error encountered, if any.
func TransformSlice(es []BackEffectEntry) ([]RestModel, error) {
	out := make([]RestModel, 0, len(es))
	for _, e := range es {
		rm, err := Transform(e)
		if err != nil {
			return nil, err
		}
		out = append(out, rm)
	}
	return out, nil
}
