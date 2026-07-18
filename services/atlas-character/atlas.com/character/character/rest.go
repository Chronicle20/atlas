package character

import (
	"context"
	"strconv"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

type RestModel struct {
	Id                 uint32   `json:"-"`
	AccountId          uint32   `json:"accountId"`
	WorldId            world.Id `json:"worldId"`
	Name               string   `json:"name"`
	Level              byte     `json:"level"`
	Experience         uint32   `json:"experience"`
	GachaponExperience uint32   `json:"gachaponExperience"`
	Strength           uint16   `json:"strength"`
	Dexterity          uint16   `json:"dexterity"`
	Intelligence       uint16   `json:"intelligence"`
	Luck               uint16   `json:"luck"`
	Hp                 uint16   `json:"hp"`
	MaxHp              uint16   `json:"maxHp"`
	Mp                 uint16   `json:"mp"`
	MaxMp              uint16   `json:"maxMp"`
	Meso               uint32   `json:"meso"`
	HpMpUsed           int      `json:"hpMpUsed"`
	JobId              job.Id   `json:"jobId"`
	SkinColor          byte     `json:"skinColor"`
	Gender             byte     `json:"gender"`
	Fame               int16    `json:"fame"`
	Hair               uint32   `json:"hair"`
	Face               uint32   `json:"face"`
	Ap                 uint16   `json:"ap"`
	Sp                 string   `json:"sp"`
	// MapId is create-time INPUT only (consumed by Extract / POST CreateAndEmit).
	// It is absent from GET responses — atlas-maps owns character location (task-087).
	MapId      _map.Id `json:"mapId"`
	SpawnPoint uint32  `json:"spawnPoint"`
	// Gm is a pointer so PATCH can distinguish an explicit gm:0 (demote) from
	// an absent field (no change). GET responses always set it.
	Gm     *int  `json:"gm"`
	X      int16 `json:"x"`
	Y      int16 `json:"y"`
	Fh     int16 `json:"fh"`
	Stance byte  `json:"stance"`
}

func (r RestModel) GetName() string {
	return "characters"
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

func (r *RestModel) SetToManyReferenceIDs(_ string, _ []string) error {
	return nil
}

// Transform produces the JSON:API projection for a character. MapId/Instance
// are NOT part of the GET projection — atlas-maps owns location (task-087).
// MapId remains on RestModel as a create-time input only (see Extract / POST).
func Transform(l logrus.FieldLogger, ctx context.Context) func(m Model) (RestModel, error) {
	t := tenant.MustFromContext(ctx)
	return func(m Model) (RestModel, error) {
		td := GetTemporalRegistry().GetById(ctx, t, m.Id())
		return transformWithTemporal(m, td), nil
	}
}

func transformWithTemporal(m Model, td temporalData) RestModel {
	gm := m.GM()
	rm := RestModel{
		Id:                 m.Id(),
		AccountId:          m.AccountId(),
		WorldId:            m.WorldId(),
		Name:               m.Name(),
		Level:              m.Level(),
		Experience:         m.Experience(),
		GachaponExperience: m.GachaponExperience(),
		Strength:           m.Strength(),
		Dexterity:          m.Dexterity(),
		Intelligence:       m.Intelligence(),
		Luck:               m.Luck(),
		Hp:                 m.Hp(),
		MaxHp:              m.MaxHp(),
		Mp:                 m.Mp(),
		MaxMp:              m.MaxMp(),
		Meso:               m.Meso(),
		HpMpUsed:           m.HpMpUsed(),
		JobId:              m.JobId(),
		SkinColor:          m.SkinColor(),
		Gender:             m.Gender(),
		Fame:               m.Fame(),
		Hair:               m.Hair(),
		Face:               m.Face(),
		Ap:                 m.AP(),
		Sp:                 m.SPString(),
		SpawnPoint:         m.SpawnPoint(),
		Gm:                 &gm,
		X:                  td.X(),
		Y:                  td.Y(),
		Fh:                 td.Fh(),
		Stance:             td.Stance(),
	}
	return rm
}

// Extract converts an inbound RestModel to the domain Model. MapId is NOT
// mapped onto the Model here — the create path reads input.MapId separately and
// passes it to CreateAndEmit (atlas-maps owns location state, task-087).
func Extract(m RestModel) (Model, error) {
	return NewModelBuilder().
		SetId(m.Id).
		SetAccountId(m.AccountId).
		SetWorldId(m.WorldId).
		SetName(m.Name).
		SetLevel(m.Level).
		SetExperience(m.Experience).
		SetGachaponExperience(m.GachaponExperience).
		SetStrength(m.Strength).
		SetDexterity(m.Dexterity).
		SetIntelligence(m.Intelligence).
		SetLuck(m.Luck).
		SetHp(m.Hp).
		SetMp(m.Mp).
		SetMaxHp(m.MaxHp).
		SetMaxMp(m.MaxMp).
		SetMeso(m.Meso).
		SetHpMpUsed(m.HpMpUsed).
		SetJobId(m.JobId).
		SetSkinColor(m.SkinColor).
		SetGender(m.Gender).
		SetFame(m.Fame).
		SetHair(m.Hair).
		SetFace(m.Face).
		SetAp(m.Ap).
		SetSp(m.Sp).
		SetSpawnPoint(m.SpawnPoint).
		SetGm(derefOrZero(m.Gm)).
		Build(), nil
}

func derefOrZero(v *int) int {
	if v == nil {
		return 0
	}
	return *v
}
