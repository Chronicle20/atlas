package factory

import (
	"atlas-character-factory/character"
	cmock "atlas-character-factory/character/mock"
	"atlas-character-factory/configuration"
	confmock "atlas-character-factory/configuration/mock"
	"atlas-character-factory/configuration/tenant"
	"atlas-character-factory/configuration/tenant/maplelife"
	"atlas-character-factory/data"
	dmock "atlas-character-factory/data/mock"
	"atlas-character-factory/saga"
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	tenantlib "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// mapleLifeWarriorEffectX / mapleLifeMagicianEffectX are test-data effect
// tables (index 0 == level 1), shaped after maple-life-content.md §5.3(b)'s
// x = 4L / x = 2L formulas but with arbitrary values since the brief allows
// fixture numbers that need not match seed data.
var (
	mapleLifeWarriorEffectX  = []int16{4, 8, 12, 16, 20, 24, 28, 32, 36, 40}
	mapleLifeMagicianEffectX = []int16{2, 4, 6, 8, 10, 12, 14, 16, 18, 20}
)

// mapleLifeTenantConfig is the fixture from the task-22 brief: one tenant,
// one gender's look options, and three classes (Warrior/Magician/Bowman).
// Characters.Templates is left at its zero value (empty) deliberately --
// TestCreateMapleLifeNeverConsultsCreationTemplates depends on that.
func mapleLifeTenantConfig(tenantId uuid.UUID) tenant.RestModel {
	return tenant.RestModel{
		Id: tenantId.String(),
		MapleLife: maplelife.RestModel{
			Looks: []maplelife.LookOptions{
				{
					Gender:     0,
					Faces:      []uint32{20000, 20001},
					Hairs:      []uint32{30030, 30020},
					HairColors: []uint32{0, 2, 3, 7},
					SkinColors: []uint32{0, 1, 2, 3},
				},
			},
			Classes: []maplelife.ClassEntry{
				{
					Ordinal: 0, Gender: 0, JobId: 100, Level: 30, MapId: 104000000,
					Stats:     maplelife.StatBlock{Str: 35, Dex: 4, Int: 4, Luk: 4, Hp: 600, Mp: 100},
					AP:        110,
					SP:        "9,0,0,0,0,0,0,0,0,0",
					SpSkillId: 1000001,
					Meso:      100000,
					Equipment: []maplelife.EquipmentEntry{{TemplateId: 1040002}},
					Inventory: []maplelife.InventoryEntry{{TemplateId: 2000002, Quantity: 100}},
				},
				{
					Ordinal: 1, Gender: 0, JobId: 200, Level: 30, MapId: 101000000,
					Stats:     maplelife.StatBlock{Str: 4, Dex: 4, Int: 35, Luk: 4, Hp: 200, Mp: 300},
					AP:        110,
					SP:        "7,0,0,0,0,0,0,0,0,0",
					SpSkillId: 2000001,
					Meso:      100000,
					Equipment: []maplelife.EquipmentEntry{{TemplateId: 1042002}},
					Inventory: []maplelife.InventoryEntry{{TemplateId: 2000002, Quantity: 100}},
				},
				{
					Ordinal: 2, Gender: 0, JobId: 300, Level: 30, MapId: 102000000,
					Stats:     maplelife.StatBlock{Str: 4, Dex: 35, Int: 4, Luk: 4, Hp: 250, Mp: 50},
					AP:        110,
					SP:        "0,0,0,0,0,0,0,0,0,0",
					Meso:      100000,
					Equipment: []maplelife.EquipmentEntry{{TemplateId: 1050002}},
					Inventory: []maplelife.InventoryEntry{{TemplateId: 2000002, Quantity: 100}},
				},
			},
		},
	}
}

func mapleLifeItems() map[uint32]data.ItemInfo {
	return map[uint32]data.ItemInfo{
		1040002: {Id: 1040002, Equipable: true},
		1042002: {Id: 1042002, Equipable: true},
		1050002: {Id: 1050002, Equipable: true},
		2000002: {Id: 2000002, Equipable: false},
	}
}

func mapleLifeSkills() map[uint32]data.SkillInfo {
	return map[uint32]data.SkillInfo{
		1000001: {Id: 1000001, MaxLevel: 10, EffectX: mapleLifeWarriorEffectX},
		2000001: {Id: 2000001, MaxLevel: 10, EffectX: mapleLifeMagicianEffectX},
	}
}

// mapleLifeCtx creates a fresh tenant, publishes cfg (or the standard
// fixture when cfg is nil) as its configuration, and returns a context
// carrying that tenant.
func mapleLifeCtx(t *testing.T, cfg *tenant.RestModel) context.Context {
	t.Helper()
	tn, err := tenantlib.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatal(err)
	}
	c := mapleLifeTenantConfig(tn.Id())
	if cfg != nil {
		c = *cfg
		c.Id = tn.Id().String()
	}
	configuration.PublishSnapshot(map[uuid.UUID]tenant.RestModel{tn.Id(): c})
	return tenantlib.WithContext(context.Background(), tn)
}

func mapleLifeBaseRequest() MapleLifeCreateRestModel {
	return MapleLifeCreateRestModel{
		AccountId:    1,
		WorldId:      0,
		Name:         "Hero",
		ClassOrdinal: 0,
		Gender:       0,
		Face:         20000,
		Hair:         30030,
		HairColor:    2,
		SkinColor:    1,
		SP:           5,
	}
}

func newMapleLifeProcessor(nc character.NameValidityClient, dc data.Processor) Processor {
	return NewProcessorWithClients(logrus.StandardLogger(), &confmock.FakePresetClient{}, nc, dc)
}

func validNameClient() character.NameValidityClient {
	return &cmock.FakeNameValidityClient{Result: character.NameValidityResult{Valid: true}}
}

func TestCreateMapleLife(t *testing.T) {
	tests := []struct {
		name    string
		request func() MapleLifeCreateRestModel
		nc      character.NameValidityClient
		dc      data.Processor
		cfg     *tenant.RestModel
		wantErr error
		// wantNameErrReason, when set, asserts a *NameInvalidError with this Reason.
		wantNameErrReason string
	}{
		{
			name:    "happy path, class 0",
			request: mapleLifeBaseRequest,
			nc:      validNameClient(),
			dc:      &dmock.ProcessorMock{Items: mapleLifeItems(), Skills: mapleLifeSkills()},
		},
		{
			name: "sp = 0, class 0",
			request: func() MapleLifeCreateRestModel {
				r := mapleLifeBaseRequest()
				r.SP = 0
				return r
			},
			nc: validNameClient(),
			dc: &dmock.ProcessorMock{Items: mapleLifeItems(), Skills: mapleLifeSkills()},
		},
		{
			name: "unknown ordinal",
			request: func() MapleLifeCreateRestModel {
				r := mapleLifeBaseRequest()
				r.ClassOrdinal = 3
				return r
			},
			nc:      validNameClient(),
			dc:      &dmock.ProcessorMock{Items: mapleLifeItems(), Skills: mapleLifeSkills()},
			wantErr: ErrClassOrdinalUnknown,
		},
		{
			name: "ordinal present, wrong gender",
			request: func() MapleLifeCreateRestModel {
				r := mapleLifeBaseRequest()
				r.Gender = 1
				return r
			},
			nc:      validNameClient(),
			dc:      &dmock.ProcessorMock{Items: mapleLifeItems(), Skills: mapleLifeSkills()},
			wantErr: ErrClassOrdinalUnknown,
		},
		{
			name: "sp on a class with no skill",
			request: func() MapleLifeCreateRestModel {
				r := mapleLifeBaseRequest()
				r.ClassOrdinal = 2
				r.SP = 3
				return r
			},
			nc:      validNameClient(),
			dc:      &dmock.ProcessorMock{Items: mapleLifeItems(), Skills: mapleLifeSkills()},
			wantErr: ErrSPInvalid,
		},
		{
			name: "sp above the pool",
			request: func() MapleLifeCreateRestModel {
				r := mapleLifeBaseRequest()
				r.SP = 10
				return r
			},
			nc:      validNameClient(),
			dc:      &dmock.ProcessorMock{Items: mapleLifeItems(), Skills: mapleLifeSkills()},
			wantErr: ErrSPInvalid,
		},
		{
			name: "face not offered",
			request: func() MapleLifeCreateRestModel {
				r := mapleLifeBaseRequest()
				r.Face = 29999
				return r
			},
			nc:      validNameClient(),
			dc:      &dmock.ProcessorMock{Items: mapleLifeItems(), Skills: mapleLifeSkills()},
			wantErr: ErrLookInvalid,
		},
		{
			name: "hair not offered",
			request: func() MapleLifeCreateRestModel {
				r := mapleLifeBaseRequest()
				r.Hair = 39999
				return r
			},
			nc:      validNameClient(),
			dc:      &dmock.ProcessorMock{Items: mapleLifeItems(), Skills: mapleLifeSkills()},
			wantErr: ErrLookInvalid,
		},
		{
			name: "hairColor not offered",
			request: func() MapleLifeCreateRestModel {
				r := mapleLifeBaseRequest()
				r.HairColor = 5
				return r
			},
			nc:      validNameClient(),
			dc:      &dmock.ProcessorMock{Items: mapleLifeItems(), Skills: mapleLifeSkills()},
			wantErr: ErrLookInvalid,
		},
		{
			name: "skin not offered",
			request: func() MapleLifeCreateRestModel {
				r := mapleLifeBaseRequest()
				r.SkinColor = 9
				return r
			},
			nc:      validNameClient(),
			dc:      &dmock.ProcessorMock{Items: mapleLifeItems(), Skills: mapleLifeSkills()},
			wantErr: ErrLookInvalid,
		},
		{
			name: "gender neither 0 nor 1",
			request: func() MapleLifeCreateRestModel {
				r := mapleLifeBaseRequest()
				r.Gender = 2
				return r
			},
			nc:      validNameClient(),
			dc:      &dmock.ProcessorMock{Items: mapleLifeItems(), Skills: mapleLifeSkills()},
			wantErr: ErrLookInvalid,
		},
		{
			name:    "duplicate name",
			request: mapleLifeBaseRequest,
			nc:      &cmock.FakeNameValidityClient{Result: character.NameValidityResult{Valid: false, Reason: "duplicate"}},
			dc:      &dmock.ProcessorMock{Items: mapleLifeItems(), Skills: mapleLifeSkills()},
			wantErr: ErrNameDuplicate,
		},
		{
			name:              "other name rejection",
			request:           mapleLifeBaseRequest,
			nc:                &cmock.FakeNameValidityClient{Result: character.NameValidityResult{Valid: false, Reason: "blocked"}},
			dc:                &dmock.ProcessorMock{Items: mapleLifeItems(), Skills: mapleLifeSkills()},
			wantNameErrReason: "blocked",
		},
		{
			name:    "no block configured",
			request: mapleLifeBaseRequest,
			nc:      validNameClient(),
			dc:      &dmock.ProcessorMock{Items: mapleLifeItems(), Skills: mapleLifeSkills()},
			cfg:     &tenant.RestModel{},
			wantErr: ErrMapleLifeNotConfigured,
		},
		{
			name:    "atlas-data down",
			request: mapleLifeBaseRequest,
			nc:      validNameClient(),
			dc:      &dmock.ProcessorMock{Items: mapleLifeItems(), SkillsErr: errors.New("atlas-data connection refused")},
			wantErr: ErrAtlasDataUnreachable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := mapleLifeCtx(t, tt.cfg)
			p := newMapleLifeProcessor(tt.nc, tt.dc)
			_, err := p.CreateMapleLife(ctx, tt.request())

			if tt.wantNameErrReason != "" {
				var nameErr *NameInvalidError
				if !errors.As(err, &nameErr) {
					t.Fatalf("expected *NameInvalidError, got %T: %v", err, err)
				}
				if nameErr.Reason != tt.wantNameErrReason {
					t.Fatalf("expected Reason %q, got %q", tt.wantNameErrReason, nameErr.Reason)
				}
				return
			}

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("expected %v, got %v", tt.wantErr, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
		})
	}
}

// TestCreateMapleLifeSagaPayload exercises resolveMapleLifePreset +
// buildPresetCharacterCreationSaga -- the same two calls CreateMapleLife
// itself makes -- so the assertions can inspect the built
// CharacterCreatePayload directly rather than round-tripping through Kafka.
func TestCreateMapleLifeSagaPayload(t *testing.T) {
	dc := &dmock.ProcessorMock{Items: mapleLifeItems(), Skills: mapleLifeSkills()}
	nc := validNameClient()

	ctx := mapleLifeCtx(t, nil)
	pi := &ProcessorImpl{l: logrus.StandardLogger(), presetClient: &confmock.FakePresetClient{}, nameClient: nc, dataClient: dc}

	req := mapleLifeBaseRequest()
	pr, skillsById, err := pi.resolveMapleLifePreset(ctx, req)
	if err != nil {
		t.Fatalf("resolveMapleLifePreset: %v", err)
	}

	sg := buildPresetCharacterCreationSaga(uuid.New(), PresetCreateRestModel{AccountId: req.AccountId, WorldId: req.WorldId, Name: req.Name}, pr, skillsById)

	var payload saga.CharacterCreatePayload
	found := false
	var createSkillCount int
	var createSkillId uint32
	var createSkillLevel byte
	for _, st := range sg.Steps {
		if st.Action == saga.CreateCharacter {
			payload = st.Payload.(saga.CharacterCreatePayload)
			found = true
		}
		if st.Action == saga.CreateSkill {
			createSkillCount++
			p := st.Payload.(saga.CreateSkillPayload)
			createSkillId = p.SkillId
			createSkillLevel = p.Level
		}
	}
	if !found {
		t.Fatalf("expected a CreateCharacter step")
	}

	if payload.JobId != 100 {
		t.Errorf("JobId: expected 100, got %d", payload.JobId)
	}
	if payload.Level != 30 {
		t.Errorf("Level: expected 30, got %d", payload.Level)
	}
	if uint32(payload.MapId) != 104000000 {
		t.Errorf("MapId: expected 104000000, got %d", payload.MapId)
	}
	if payload.Face != 20000 {
		t.Errorf("Face: expected 20000, got %d", payload.Face)
	}
	if payload.Hair != 30032 {
		t.Errorf("Hair: expected 30032 (style + colour), got %d", payload.Hair)
	}
	if payload.Skin != 1 {
		t.Errorf("Skin: expected 1, got %d", payload.Skin)
	}
	if payload.Meso != 100000 {
		t.Errorf("Meso: expected 100000, got %d", payload.Meso)
	}
	if payload.AP != 110 {
		t.Errorf("AP: expected 110, got %d", payload.AP)
	}
	if payload.SP != "4,0,0,0,0,0,0,0,0,0" {
		t.Errorf("SP: expected pool minus 5 spent, got %q", payload.SP)
	}

	// Warrior HP: seeded 600 + 29*4*5 = 1180; MP unchanged at 100.
	if payload.Hp != 600+29*4*5 {
		t.Errorf("Hp: expected %d, got %d", 600+29*4*5, payload.Hp)
	}
	if payload.Mp != 100 {
		t.Errorf("Mp: expected unchanged 100, got %d", payload.Mp)
	}

	if createSkillCount != 1 {
		t.Fatalf("expected exactly one CreateSkill step, got %d", createSkillCount)
	}
	if createSkillId != 1000001 {
		t.Errorf("CreateSkill SkillId: expected 1000001, got %d", createSkillId)
	}
	if createSkillLevel != 5 {
		t.Errorf("CreateSkill Level: expected 5, got %d", createSkillLevel)
	}
}

func TestCreateMapleLifeSagaPayload_SPZero(t *testing.T) {
	dc := &dmock.ProcessorMock{Items: mapleLifeItems(), Skills: mapleLifeSkills()}
	ctx := mapleLifeCtx(t, nil)
	pi := &ProcessorImpl{l: logrus.StandardLogger(), presetClient: &confmock.FakePresetClient{}, nameClient: validNameClient(), dataClient: dc}

	req := mapleLifeBaseRequest()
	req.SP = 0
	pr, skillsById, err := pi.resolveMapleLifePreset(ctx, req)
	if err != nil {
		t.Fatalf("resolveMapleLifePreset: %v", err)
	}
	sg := buildPresetCharacterCreationSaga(uuid.New(), PresetCreateRestModel{AccountId: req.AccountId, WorldId: req.WorldId, Name: req.Name}, pr, skillsById)

	var payload saga.CharacterCreatePayload
	found := false
	for _, st := range sg.Steps {
		switch st.Action {
		case saga.CreateCharacter:
			payload = st.Payload.(saga.CharacterCreatePayload)
			found = true
		case saga.CreateSkill:
			t.Fatalf("expected no CreateSkill step when sp = 0, got one")
		}
	}
	if !found {
		t.Fatalf("expected a CreateCharacter step")
	}
	if payload.SP != "9,0,0,0,0,0,0,0,0,0" {
		t.Errorf("SP: expected pool unchanged, got %q", payload.SP)
	}
	if payload.Hp != 600 {
		t.Errorf("Hp: expected fixture value 600 exactly (no adjustment), got %d", payload.Hp)
	}
}

func TestCreateMapleLifeSagaPayload_MagicianMP(t *testing.T) {
	dc := &dmock.ProcessorMock{Items: mapleLifeItems(), Skills: mapleLifeSkills()}
	ctx := mapleLifeCtx(t, nil)
	pi := &ProcessorImpl{l: logrus.StandardLogger(), presetClient: &confmock.FakePresetClient{}, nameClient: validNameClient(), dataClient: dc}

	req := mapleLifeBaseRequest()
	req.ClassOrdinal = 1
	req.SP = 3
	pr, skillsById, err := pi.resolveMapleLifePreset(ctx, req)
	if err != nil {
		t.Fatalf("resolveMapleLifePreset: %v", err)
	}
	sg := buildPresetCharacterCreationSaga(uuid.New(), PresetCreateRestModel{AccountId: req.AccountId, WorldId: req.WorldId, Name: req.Name}, pr, skillsById)

	var payload saga.CharacterCreatePayload
	for _, st := range sg.Steps {
		if st.Action == saga.CreateCharacter {
			payload = st.Payload.(saga.CharacterCreatePayload)
		}
	}
	// Magician MP: seeded 300 + 29*2*3 = 474; HP unchanged at 200.
	if payload.Mp != 300+29*2*3 {
		t.Errorf("Mp: expected %d, got %d", 300+29*2*3, payload.Mp)
	}
	if payload.Hp != 200 {
		t.Errorf("Hp: expected unchanged 200, got %d", payload.Hp)
	}
}

func TestCreateMapleLifeSagaPayload_NoSkillClass(t *testing.T) {
	dc := &dmock.ProcessorMock{Items: mapleLifeItems(), Skills: mapleLifeSkills()}
	ctx := mapleLifeCtx(t, nil)
	pi := &ProcessorImpl{l: logrus.StandardLogger(), presetClient: &confmock.FakePresetClient{}, nameClient: validNameClient(), dataClient: dc}

	req := mapleLifeBaseRequest()
	req.ClassOrdinal = 2
	req.SP = 0
	pr, skillsById, err := pi.resolveMapleLifePreset(ctx, req)
	if err != nil {
		t.Fatalf("resolveMapleLifePreset: %v", err)
	}
	sg := buildPresetCharacterCreationSaga(uuid.New(), PresetCreateRestModel{AccountId: req.AccountId, WorldId: req.WorldId, Name: req.Name}, pr, skillsById)

	var payload saga.CharacterCreatePayload
	for _, st := range sg.Steps {
		if st.Action == saga.CreateCharacter {
			payload = st.Payload.(saga.CharacterCreatePayload)
		}
	}
	if payload.Hp != 250 {
		t.Errorf("Hp: expected fixture value 250 exactly, got %d", payload.Hp)
	}
	if payload.Mp != 50 {
		t.Errorf("Mp: expected fixture value 50 exactly, got %d", payload.Mp)
	}
}

// TestCreateMapleLifeNeverConsultsCreationTemplates asserts §11 A5's "the
// eleven creation-template rules do not apply to this path": a tenant whose
// Characters.Templates is empty but whose MapleLife block is populated still
// succeeds.
func TestCreateMapleLifeNeverConsultsCreationTemplates(t *testing.T) {
	cfg := mapleLifeTenantConfig(uuid.Nil) // Id overwritten by mapleLifeCtx
	if len(cfg.Characters.Templates) != 0 {
		t.Fatalf("fixture precondition: expected empty Characters.Templates, got %d", len(cfg.Characters.Templates))
	}
	ctx := mapleLifeCtx(t, &cfg)
	p := newMapleLifeProcessor(validNameClient(), &dmock.ProcessorMock{Items: mapleLifeItems(), Skills: mapleLifeSkills()})

	_, err := p.CreateMapleLife(ctx, mapleLifeBaseRequest())
	if err != nil {
		t.Fatalf("expected no error despite empty Characters.Templates, got %v", err)
	}
}
