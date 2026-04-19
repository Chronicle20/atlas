package npc

import "strconv"

type NpcMapRestModel struct {
	NpcId      uint32 `json:"-"`
	MapId      uint32 `json:"mapId"`
	Name       string `json:"name"`
	StreetName string `json:"streetName"`
	SpawnCount uint32 `json:"spawnCount"`
}

func (r NpcMapRestModel) GetName() string { return "npc-maps" }
func (r NpcMapRestModel) GetID() string   { return strconv.Itoa(int(r.NpcId)) }

func (r *NpcMapRestModel) SetID(idStr string) error {
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return err
	}
	r.NpcId = uint32(id)
	return nil
}
