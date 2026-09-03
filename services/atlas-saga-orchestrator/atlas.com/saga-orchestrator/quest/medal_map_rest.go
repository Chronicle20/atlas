package quest

// medalMapPostRestModel is the request body for atlas-quest's
// POST /characters/{characterId}/quests/{questId}/medal-maps.
type medalMapPostRestModel struct {
	Id    string `json:"-"`
	MapId uint32 `json:"mapId"`
}

func (r medalMapPostRestModel) GetName() string {
	return "medal-maps"
}

func (r medalMapPostRestModel) GetID() string {
	return r.Id
}

func (r *medalMapPostRestModel) SetID(id string) error {
	r.Id = id
	return nil
}

// medalMapRestModel is atlas-quest's medal-map record response. It carries
// only the resulting distinct-map count -- see medal_map.Processor's doc
// comment in atlas-quest for why no threshold/completed field exists.
type medalMapRestModel struct {
	Id    string `json:"-"`
	Count uint32 `json:"count"`
}

func (r medalMapRestModel) GetName() string {
	return "medal-maps"
}

func (r medalMapRestModel) GetID() string {
	return r.Id
}

func (r *medalMapRestModel) SetID(id string) error {
	r.Id = id
	return nil
}
