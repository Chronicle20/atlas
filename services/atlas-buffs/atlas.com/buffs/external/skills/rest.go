package skills

import "strconv"

// RestModel is the trimmed atlas-skills projection: Berserk level at login.
type RestModel struct {
	Id    uint32 `json:"-"`
	Level byte   `json:"level"`
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
