package pet

import "strconv"

type RestModel struct {
	Id         uint32 `json:"-"`
	TemplateId uint32 `json:"templateId"`
	Name       string `json:"name"`
	OwnerId    uint32 `json:"ownerId"`
}

func (r RestModel) GetName() string {
	return "pets"
}

func (r RestModel) GetID() string {
	return strconv.Itoa(int(r.Id))
}

func (r *RestModel) SetID(strId string) error {
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
		TemplateId: m.TemplateId(),
		Name:       m.Name(),
		OwnerId:    m.OwnerId(),
	}, nil
}

func Extract(rm RestModel) (Model, error) {
	return Model{
		id:         rm.Id,
		templateId: rm.TemplateId,
		name:       rm.Name,
		ownerId:    rm.OwnerId,
	}, nil
}
