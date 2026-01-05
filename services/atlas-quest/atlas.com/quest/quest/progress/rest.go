package progress

import "strconv"

type RestModel struct {
	Id         uint32 `json:"-"`
	InfoNumber uint32 `json:"infoNumber"`
	Progress   string `json:"progress"`
}

func (r RestModel) GetName() string {
	return "progress"
}

func (r RestModel) GetID() string {
	return strconv.Itoa(int(r.Id))
}

func (r *RestModel) SetID(strId string) error {
	if strId == "" {
		r.Id = 0
		return nil
	}

	id, err := strconv.Atoi(strId)
	if err != nil {
		return err
	}
	r.Id = uint32(id)
	return nil
}

func Transform(m Model) (RestModel, error) {
	return RestModel{
		Id:         m.id,
		InfoNumber: m.infoNumber,
		Progress:   m.progress,
	}, nil
}

func Extract(rm RestModel) (Model, error) {
	return Model{
		id:         rm.Id,
		infoNumber: rm.InfoNumber,
		progress:   rm.Progress,
	}, nil
}
