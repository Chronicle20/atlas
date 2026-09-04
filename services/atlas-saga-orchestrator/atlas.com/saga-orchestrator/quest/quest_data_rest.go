package quest

// questRequirementsRestModel mirrors the fields of atlas-data's
// quest.RequirementsRestModel that explorerQuest needs: infoNumber and
// infoEx (services/atlas-data/atlas.com/data/quest/rest.go).
type questRequirementsRestModel struct {
	InfoNumber uint32   `json:"infoNumber,omitempty"`
	InfoEx     []string `json:"infoEx,omitempty"`
}

// questDataRestModel is the subset of atlas-data's
// GET /data/quests/{questId} response explorerQuest needs to resolve
// Cosmic's status->section infoNumber/infoEx mapping
// (Quest.java:462-485: `boolean checkEnd = qs.equals(Status.STARTED)`).
// The quest is force-started (STARTED) at the point explorerQuest reads it,
// so both lookups read EndRequirements.
type questDataRestModel struct {
	Id              string                     `json:"-"`
	EndRequirements questRequirementsRestModel `json:"endRequirements"`
}

func (r questDataRestModel) GetName() string {
	return "quests"
}

func (r questDataRestModel) GetID() string {
	return r.Id
}

func (r *questDataRestModel) SetID(id string) error {
	r.Id = id
	return nil
}

// Required JSON:API relationship stubs (libs/atlas-rest gotcha): api2go errors
// out decoding any response unless the target implements these, even with no
// relationships present.
func (r *questDataRestModel) SetToOneReferenceID(_, _ string) error { return nil }

func (r *questDataRestModel) SetToManyReferenceIDs(_ string, _ []string) error { return nil }
