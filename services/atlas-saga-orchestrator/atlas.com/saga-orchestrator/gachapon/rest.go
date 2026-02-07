package gachapon

type RewardRestModel struct {
	Id         string `json:"-"`
	ItemId     uint32 `json:"itemId"`
	Quantity   uint32 `json:"quantity"`
	Tier       string `json:"tier"`
	GachaponId string `json:"gachaponId"`
}

func (r RewardRestModel) GetID() string {
	return r.Id
}

func (r *RewardRestModel) SetID(idStr string) error {
	r.Id = idStr
	return nil
}

func (r RewardRestModel) GetName() string {
	return "gachapon-rewards"
}

type GachaponRestModel struct {
	Id             string   `json:"-"`
	Name           string   `json:"name"`
	NpcIds         []uint32 `json:"npcIds"`
	CommonWeight   uint32   `json:"commonWeight"`
	UncommonWeight uint32   `json:"uncommonWeight"`
	RareWeight     uint32   `json:"rareWeight"`
}

func (r GachaponRestModel) GetID() string {
	return r.Id
}

func (r *GachaponRestModel) SetID(idStr string) error {
	r.Id = idStr
	return nil
}

func (r GachaponRestModel) GetName() string {
	return "gachapons"
}
