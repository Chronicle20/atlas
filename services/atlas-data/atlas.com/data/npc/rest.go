package npc

import "strconv"

type RestModel struct {
	Id        uint32 `json:"-"`
	Name      string `json:"name"`
	TrunkPut  int32  `json:"trunk_put"`
	TrunkGet  int32  `json:"trunk_get"`
	Storebank bool   `json:"storebank"`
	HideName  bool   `json:"hide_name"`
	DcLeft    int32  `json:"dc_left"`
	DcRight   int32  `json:"dc_right"`
	DcTop     int32  `json:"dc_top"`
	DcBottom  int32  `json:"dc_bottom"`
}

func (r RestModel) GetName() string {
	return "npcs"
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
