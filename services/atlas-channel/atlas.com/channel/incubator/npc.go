package incubator

import "strconv"

// SuccessNpcId is the NPC the client renders in the incubator result dialog.
//
// It is HARD-CODED in the client's CWvsContext::OnIncubatorResult — verified in
// the GMS v83 (@0xa28298) and JMS v185 (@0xb0f30b) IDBs, where the success
// branch (itemId > 0) passes the immediate 0x8A1798 (= 9050008) to
// CUtilDlgEx::SetUtilDlgEx as the dialog NPC; v84/v87 share the same flat
// handler. On the wire the v83-family INCUBATOR_RESULT carries no NPC — the
// client always uses this fixed id.
//
// The GMS client family never shipped Npc/9050008.img, so on those clients
// OnIncubatorResult(itemId > 0) faults inside CUtilDlgEx::SetNPC with
// STG_E_FILENOTFOUND (it formats "Npc/%07d.img/info/default" and loads it). The
// channel therefore gates incubation on this NPC being present in the tenant's
// game data (atlas-data) — see Processor.SuccessNpcAvailable — so a player can't
// win the lottery and then crash on a reward the client can't present.
const SuccessNpcId uint32 = 9050008

// npcRestModel is a minimal projection of the atlas-data npc resource
// (GET /data/npcs/{id}) — enough to detect existence (200 vs 404).
type npcRestModel struct {
	Id   uint32 `json:"-"`
	Name string `json:"name"`
}

func (r npcRestModel) GetName() string { return "npcs" }
func (r npcRestModel) GetID() string   { return strconv.Itoa(int(r.Id)) }
func (r *npcRestModel) SetID(id string) error {
	v, err := strconv.Atoi(id)
	if err != nil {
		return err
	}
	r.Id = uint32(v)
	return nil
}

// SetToOneReferenceID satisfies the api2go UnmarshalToOneRelations interface.
func (r *npcRestModel) SetToOneReferenceID(_ string, _ string) error { return nil }

// SetToManyReferenceIDs satisfies the api2go UnmarshalToManyRelations interface.
func (r *npcRestModel) SetToManyReferenceIDs(_ string, _ []string) error { return nil }
