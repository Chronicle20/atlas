package ring

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestTransform(t *testing.T) {
	id := uuid.New()
	pairId := uuid.New()
	createdAt := time.Now()

	m := Model{
		id:                 id,
		pairId:             pairId,
		characterId:        42,
		partnerCharacterId: 77,
		assetId:            1001,
		itemTemplateId:     1112800,
		ringType:           TypeFriendship,
		state:              StateActive,
		createdAt:          createdAt,
		cashId:             int64(9007199254740993),
		partnerCashId:      int64(-1),
		partnerName:        "PartnerChar",
	}

	rm, err := Transform(m)
	if err != nil {
		t.Fatalf("Transform: %v", err)
	}
	if rm.Id != id {
		t.Errorf("Id = %v, want %v", rm.Id, id)
	}
	if rm.PairId != pairId {
		t.Errorf("PairId = %v, want %v", rm.PairId, pairId)
	}
	if rm.CharacterId != 42 {
		t.Errorf("CharacterId = %d, want 42", rm.CharacterId)
	}
	if rm.PartnerCharacterId != 77 {
		t.Errorf("PartnerCharacterId = %d, want 77", rm.PartnerCharacterId)
	}
	if rm.AssetId != 1001 {
		t.Errorf("AssetId = %d, want 1001", rm.AssetId)
	}
	if rm.ItemTemplateId != 1112800 {
		t.Errorf("ItemTemplateId = %d, want 1112800", rm.ItemTemplateId)
	}
	if rm.RingType != string(TypeFriendship) {
		t.Errorf("RingType = %s, want %s", rm.RingType, TypeFriendship)
	}
	if rm.State != string(StateActive) {
		t.Errorf("State = %s, want %s", rm.State, StateActive)
	}
	if !rm.CreatedAt.Equal(createdAt) {
		t.Errorf("CreatedAt = %v, want %v", rm.CreatedAt, createdAt)
	}
	if rm.CashId != 9007199254740993 {
		t.Errorf("CashId = %d, want 9007199254740993", rm.CashId)
	}
	if rm.PartnerCashId != -1 {
		t.Errorf("PartnerCashId = %d, want -1", rm.PartnerCashId)
	}
	if rm.PartnerName != "PartnerChar" {
		t.Errorf("PartnerName = %s, want PartnerChar", rm.PartnerName)
	}
}

func TestRestModelGetName(t *testing.T) {
	rm := RestModel{}
	if rm.GetName() != "rings" {
		t.Errorf("GetName() = %s, want rings", rm.GetName())
	}
}

func TestRestModelGetID(t *testing.T) {
	id := uuid.New()
	rm := RestModel{Id: id}
	if rm.GetID() != id.String() {
		t.Errorf("GetID() = %s, want %s", rm.GetID(), id.String())
	}
}

func TestRestModelSetID(t *testing.T) {
	rm := &RestModel{}
	id := uuid.New()
	if err := rm.SetID(id.String()); err != nil {
		t.Fatalf("SetID: %v", err)
	}
	if rm.Id != id {
		t.Errorf("Id = %v, want %v", rm.Id, id)
	}
}

func TestRestModelSetIDEmpty(t *testing.T) {
	rm := &RestModel{Id: uuid.New()}
	if err := rm.SetID(""); err != nil {
		t.Fatalf("SetID: %v", err)
	}
	if rm.Id != uuid.Nil {
		t.Errorf("Id = %v, want nil", rm.Id)
	}
}

func TestRestModelSetIDInvalid(t *testing.T) {
	rm := &RestModel{}
	if err := rm.SetID("not-a-uuid"); err == nil {
		t.Error("SetID should fail for an invalid UUID")
	}
}
