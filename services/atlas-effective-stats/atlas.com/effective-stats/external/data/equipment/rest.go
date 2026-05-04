package equipment

import "strconv"

// RestModel mirrors the requirement subset of atlas-data's equipment endpoint.
// atlas-data exposes many more fields (per-stat bonuses, classification, etc.)
// but atlas-effective-stats only needs the requirement gate inputs — per-asset
// stats already arrive via atlas-inventory.
type RestModel struct {
	Id       uint32 `json:"-"`
	ReqLevel byte   `json:"reqLevel"`
	ReqJob   uint16 `json:"reqJob"`
	ReqStr   uint16 `json:"reqStr"`
	ReqDex   uint16 `json:"reqDex"`
	ReqInt   uint16 `json:"reqInt"`
	ReqLuk   uint16 `json:"reqLuk"`
}

func (r RestModel) GetName() string { return "equipment" }

func (r RestModel) GetID() string { return strconv.Itoa(int(r.Id)) }

func (r *RestModel) SetID(idStr string) error {
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return err
	}
	r.Id = uint32(id)
	return nil
}
