package _map

import "strconv"

type RestModel struct {
	Id string `json:"-"`
}

func (m RestModel) GetID() string {
	return m.Id
}

func (m RestModel) GetName() string {
	return "characters"
}

func (m *RestModel) SetID(idStr string) error {
	m.Id = idStr
	return nil
}

func Transform(id uint32) (RestModel, error) {
	return RestModel{Id: strconv.Itoa(int(id))}, nil
}

// ResetFieldInputRestModel is the body of POST .../reset -- Cosmic's
// resetPQ(difficulty). Difficulty is accepted for wire parity but is
// currently unused by ResetField; see the doc comment on
// map/monster.ProcessorImpl.ResetField for why.
type ResetFieldInputRestModel struct {
	Id         string `json:"-"`
	Difficulty int    `json:"difficulty"`
}

func (r ResetFieldInputRestModel) GetName() string {
	return "maps"
}

func (r ResetFieldInputRestModel) GetID() string {
	return r.Id
}

func (r *ResetFieldInputRestModel) SetID(strId string) error {
	r.Id = strId
	return nil
}
