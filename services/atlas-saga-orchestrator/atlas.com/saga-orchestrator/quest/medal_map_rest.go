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

// medalMapRestModel is atlas-quest's medal-map record response. Count is the
// resulting distinct-map count; NewlyRecorded is Cosmic's
// `qs.addMedalMap(...)` dedup result (MapScriptMethods.java:104-139:
// `if (!qs.addMedalMap(...)) return;`) -- RequestExplorerQuest uses it to
// gate the progress write that follows a record. No threshold/completed
// field exists here -- see atlas-quest's medal_map.Processor doc comment for
// the infoEx(0) comparison, which is derived from atlas-data separately.
type medalMapRestModel struct {
	Id            string `json:"-"`
	Count         uint32 `json:"count"`
	NewlyRecorded bool   `json:"newlyRecorded"`
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
