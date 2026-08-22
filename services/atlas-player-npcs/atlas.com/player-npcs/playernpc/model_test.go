package playernpc_test

import (
	"atlas-player-npcs/playernpc"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
)

// buildFullModel returns a Model with every field set through the Builder,
// used to exercise the model -> entity -> model round trip.
func buildFullModel(t *testing.T) playernpc.Model {
	t.Helper()

	e1, err := playernpc.NewEquipmentBuilder().
		SetId(uuid.New()).
		SetSlot(-1).
		SetItemId(1002140).
		Build()
	if err != nil {
		t.Fatalf("Failed to build equipment: %v", err)
	}
	e2, err := playernpc.NewEquipmentBuilder().
		SetId(uuid.New()).
		SetSlot(-5).
		SetItemId(1040002).
		Build()
	if err != nil {
		t.Fatalf("Failed to build equipment: %v", err)
	}
	e3, err := playernpc.NewEquipmentBuilder().
		SetId(uuid.New()).
		SetSlot(-11).
		SetItemId(1060002).
		Build()
	if err != nil {
		t.Fatalf("Failed to build equipment: %v", err)
	}

	now := time.Now().UTC().Truncate(time.Second)

	m, err := playernpc.NewBuilder().
		SetId(uuid.New()).
		SetCharacterId(123).
		SetName("Statue Guy").
		SetWorldId(0).
		SetMapId(102000004).
		SetScriptId(9901000).
		SetObjectId(555).
		SetGender(0).
		SetSkin(1).
		SetFace(20000).
		SetHair(30000).
		SetJobId(job.Id(112)).
		SetX(100).
		SetCy(200).
		SetFh(17).
		SetDir(0).
		SetWorldRank(1).
		SetOverallRank(1).
		SetWorldJobRank(1).
		SetOverallJobRank(1).
		SetCreatedAt(now).
		SetUpdatedAt(now).
		// intentionally out of slot order -- MakeEquipmentEntities/Make must
		// not depend on caller order.
		SetEquipment([]playernpc.EquipmentModel{e1, e3, e2}).
		Build()
	if err != nil {
		t.Fatalf("Failed to build model: %v", err)
	}
	return m
}

func TestPlayerNpcEntityRoundTrip(t *testing.T) {
	tenantId := uuid.New()

	t.Run("model -> entity -> model round trip", func(t *testing.T) {
		m := buildFullModel(t)

		entity := m.ToEntity(tenantId)
		equipmentEntities := playernpc.MakeEquipmentEntities(tenantId, entity.Id, m)

		got, err := playernpc.Make(entity, equipmentEntities)
		if err != nil {
			t.Fatalf("Make() unexpected err = %v", err)
		}

		if got.Id() != m.Id() {
			t.Errorf("Id() = %v, want %v", got.Id(), m.Id())
		}
		if got.CharacterId() != m.CharacterId() {
			t.Errorf("CharacterId() = %v, want %v", got.CharacterId(), m.CharacterId())
		}
		if got.Name() != m.Name() {
			t.Errorf("Name() = %v, want %v", got.Name(), m.Name())
		}
		if got.WorldId() != m.WorldId() {
			t.Errorf("WorldId() = %v, want %v", got.WorldId(), m.WorldId())
		}
		if got.MapId() != m.MapId() {
			t.Errorf("MapId() = %v, want %v", got.MapId(), m.MapId())
		}
		if got.ScriptId() != m.ScriptId() {
			t.Errorf("ScriptId() = %v, want %v", got.ScriptId(), m.ScriptId())
		}
		if got.ObjectId() != m.ObjectId() {
			t.Errorf("ObjectId() = %v, want %v", got.ObjectId(), m.ObjectId())
		}
		if got.Gender() != m.Gender() {
			t.Errorf("Gender() = %v, want %v", got.Gender(), m.Gender())
		}
		if got.Skin() != m.Skin() {
			t.Errorf("Skin() = %v, want %v", got.Skin(), m.Skin())
		}
		if got.Face() != m.Face() {
			t.Errorf("Face() = %v, want %v", got.Face(), m.Face())
		}
		if got.Hair() != m.Hair() {
			t.Errorf("Hair() = %v, want %v", got.Hair(), m.Hair())
		}
		if got.JobId() != m.JobId() {
			t.Errorf("JobId() = %v, want %v", got.JobId(), m.JobId())
		}
		if got.X() != m.X() {
			t.Errorf("X() = %v, want %v", got.X(), m.X())
		}
		if got.Cy() != m.Cy() {
			t.Errorf("Cy() = %v, want %v", got.Cy(), m.Cy())
		}
		if got.Fh() != m.Fh() {
			t.Errorf("Fh() = %v, want %v", got.Fh(), m.Fh())
		}
		if got.RX0() != m.RX0() {
			t.Errorf("RX0() = %v, want %v", got.RX0(), m.RX0())
		}
		if got.RX1() != m.RX1() {
			t.Errorf("RX1() = %v, want %v", got.RX1(), m.RX1())
		}
		if got.Dir() != m.Dir() {
			t.Errorf("Dir() = %v, want %v", got.Dir(), m.Dir())
		}
		if got.WorldRank() != m.WorldRank() {
			t.Errorf("WorldRank() = %v, want %v", got.WorldRank(), m.WorldRank())
		}
		if got.OverallRank() != m.OverallRank() {
			t.Errorf("OverallRank() = %v, want %v", got.OverallRank(), m.OverallRank())
		}
		if got.WorldJobRank() != m.WorldJobRank() {
			t.Errorf("WorldJobRank() = %v, want %v", got.WorldJobRank(), m.WorldJobRank())
		}
		if got.OverallJobRank() != m.OverallJobRank() {
			t.Errorf("OverallJobRank() = %v, want %v", got.OverallJobRank(), m.OverallJobRank())
		}
		if !got.CreatedAt().Equal(m.CreatedAt()) {
			t.Errorf("CreatedAt() = %v, want %v", got.CreatedAt(), m.CreatedAt())
		}
		if !got.UpdatedAt().Equal(m.UpdatedAt()) {
			t.Errorf("UpdatedAt() = %v, want %v", got.UpdatedAt(), m.UpdatedAt())
		}
	})

	t.Run("dir defaults to 1", func(t *testing.T) {
		m, err := playernpc.NewBuilder().
			SetCharacterId(1).
			SetName("Defaulted").
			Build()
		if err != nil {
			t.Fatalf("Build() unexpected err = %v", err)
		}
		if m.Dir() != 1 {
			t.Errorf("Dir() = %v, want 1", m.Dir())
		}
	})

	t.Run("computed rx", func(t *testing.T) {
		m, err := playernpc.NewBuilder().
			SetCharacterId(1).
			SetName("Positioned").
			SetX(100).
			Build()
		if err != nil {
			t.Fatalf("Build() unexpected err = %v", err)
		}
		if m.RX0() != 150 {
			t.Errorf("RX0() = %v, want 150", m.RX0())
		}
		if m.RX1() != 50 {
			t.Errorf("RX1() = %v, want 50", m.RX1())
		}
	})

	t.Run("job category stored", func(t *testing.T) {
		m, err := playernpc.NewBuilder().
			SetCharacterId(1).
			SetName("Job Category").
			SetJobId(job.Id(112)).
			Build()
		if err != nil {
			t.Fatalf("Build() unexpected err = %v", err)
		}
		if m.JobId() != 100 {
			t.Errorf("JobId() = %v, want 100", m.JobId())
		}
	})

	t.Run("equipment child collection survives in slot order", func(t *testing.T) {
		m := buildFullModel(t)

		entity := m.ToEntity(tenantId)
		equipmentEntities := playernpc.MakeEquipmentEntities(tenantId, entity.Id, m)

		got, err := playernpc.Make(entity, equipmentEntities)
		if err != nil {
			t.Fatalf("Make() unexpected err = %v", err)
		}

		equipment := got.Equipment()
		if len(equipment) != 3 {
			t.Fatalf("len(Equipment()) = %v, want 3", len(equipment))
		}
		wantSlots := []int16{-11, -5, -1}
		for i, w := range wantSlots {
			if equipment[i].Slot() != w {
				t.Errorf("Equipment()[%d].Slot() = %v, want %v", i, equipment[i].Slot(), w)
			}
		}
	})
}
