package job

import "strconv"

type RestModel struct {
	Id     uint32   `json:"-"`
	Skills []uint32 `json:"skills"`
}

func (r RestModel) GetName() string { return "jobs" }
func (r RestModel) GetID() string   { return strconv.Itoa(int(r.Id)) }

func (r *RestModel) SetID(idStr string) error {
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return err
	}
	r.Id = uint32(id)
	return nil
}
