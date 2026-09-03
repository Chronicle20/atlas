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

// RestModel is the medal-map record response. Count is the resulting
// distinct-map count; NewlyRecorded is Cosmic's `qs.addMedalMap(...)` dedup
// result -- callers use it to gate the quest-progress write that follows a
// record (task-290 C22b/G14: Cosmic's explorerQuest returns early on a
// duplicate map and never writes progress). No threshold/completed field
// exists here -- see Processor's doc comment for the infoEx(0) comparison,
// which callers derive themselves from atlas-data.
type RestModel struct {
	Id            string `json:"-"`
	Count         uint32 `json:"count"`
	NewlyRecorded bool   `json:"newlyRecorded"`
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
		Count:         result.Count,
		NewlyRecorded: result.NewlyRecorded,
	}, nil
}
