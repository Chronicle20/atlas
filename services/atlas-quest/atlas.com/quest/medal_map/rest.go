package medal_map

// PostRestModel is the medal-map record request body: the map the character
// is currently standing in.
type PostRestModel struct {
	Id    string `json:"-"`
	MapId uint32 `json:"mapId"`
}

func (r PostRestModel) GetName() string {
	return "medal-maps"
}

func (r PostRestModel) GetID() string {
	return r.Id
}

func (r *PostRestModel) SetID(id string) error {
	r.Id = id
	return nil
}

// RestModel is the medal-map record response. It reports only the resulting
// distinct-map count -- no threshold or completed flag, because atlas-data
// does not serve the quest's infoEx(0) threshold Cosmic compares against
// (see Processor's doc comment). A caller wanting completion semantics must
// derive them from a data source this task did not find.
type RestModel struct {
	Id    string `json:"-"`
	Count uint32 `json:"count"`
}

func (r RestModel) GetName() string {
	return "medal-maps"
}

func (r RestModel) GetID() string {
	return r.Id
}

func (r *RestModel) SetID(id string) error {
	r.Id = id
	return nil
}

func Transform(result RecordResult) (RestModel, error) {
	return RestModel{
		Count: result.Count,
	}, nil
}
