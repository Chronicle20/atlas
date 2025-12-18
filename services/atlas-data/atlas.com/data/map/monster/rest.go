package monster

import "strconv"

type RestModel struct {
	Id       uint32 `json:"-"`
	Template uint32 `json:"template"`
	MobTime  uint32 `json:"mob_time"`
	Team     int32  `json:"team"`
	CY       int16  `json:"cy"`
	F        uint32 `json:"f"`
	FH       uint16 `json:"fh"`
	RX0      int16  `json:"rx0"`
	RX1      int16  `json:"rx1"`
	X        int16  `json:"x"`
	Y        int16  `json:"y"`
	Hide     bool   `json:"hide"`
}

func (r RestModel) GetName() string {
	return "monsters"
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
