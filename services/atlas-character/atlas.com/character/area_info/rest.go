package area_info

type RestModel struct {
	Id          string `json:"-"`
	CharacterId uint32 `json:"characterId"`
	Area        uint16 `json:"area"`
	Info        string `json:"info"`
}

func (r RestModel) GetName() string {
	return "area-infos"
}

func (r RestModel) GetID() string {
	return r.Id
}

func (r *RestModel) SetID(id string) error {
	r.Id = id
	return nil
}

func Transform(m Model) (RestModel, error) {
	return RestModel{
		Id:          m.Id().String(),
		CharacterId: m.CharacterId(),
		Area:        m.Area(),
		Info:        m.Info(),
	}, nil
}

func Extract(rm RestModel) (Model, error) {
	return NewBuilder().
		SetCharacterId(rm.CharacterId).
		SetArea(rm.Area).
		SetInfo(rm.Info).
		Build(), nil
}

// TransformSlice maps a slice of domain Models to their REST projections.
// Returns the first transform error encountered, if any.
func TransformSlice(ms []Model) ([]RestModel, error) {
	rs := make([]RestModel, 0, len(ms))
	for _, m := range ms {
		r, err := Transform(m)
		if err != nil {
			return nil, err
		}
		rs = append(rs, r)
	}
	return rs, nil
}

// PutRestModel is the PUT route's request body: a full replace of the stored
// string for the area named in the path, matching Character.updateAreaInfo's
// replace semantics.
type PutRestModel struct {
	Id   string `json:"-"`
	Info string `json:"info"`
}

func (r PutRestModel) GetName() string {
	return "area-infos"
}

func (r PutRestModel) GetID() string {
	return r.Id
}

func (r *PutRestModel) SetID(id string) error {
	r.Id = id
	return nil
}
