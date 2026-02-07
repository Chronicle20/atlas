package gachapon

type RestModel struct {
	Id             string   `json:"-"`
	Name           string   `json:"name"`
	NpcIds         []uint32 `json:"npcIds"`
	CommonWeight   uint32   `json:"commonWeight"`
	UncommonWeight uint32   `json:"uncommonWeight"`
	RareWeight     uint32   `json:"rareWeight"`
}

func (r RestModel) GetName() string {
	return "gachapons"
}

func (r RestModel) GetID() string {
	return r.Id
}

func (r *RestModel) SetID(idStr string) error {
	r.Id = idStr
	return nil
}

func Transform(m Model) (RestModel, error) {
	return RestModel{
		Id:             m.Id(),
		Name:           m.Name(),
		NpcIds:         m.NpcIds(),
		CommonWeight:   m.CommonWeight(),
		UncommonWeight: m.UncommonWeight(),
		RareWeight:     m.RareWeight(),
	}, nil
}

type JSONModel struct {
	Id             string   `json:"id"`
	Name           string   `json:"name"`
	NpcIds         []uint32 `json:"npcIds"`
	CommonWeight   uint32   `json:"commonWeight"`
	UncommonWeight uint32   `json:"uncommonWeight"`
	RareWeight     uint32   `json:"rareWeight"`
}
