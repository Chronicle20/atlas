package ring

import (
	"testing"

	"github.com/google/uuid"
)

func TestExtract(t *testing.T) {
	id := uuid.New()
	pairId := uuid.New()

	rm := RestModel{
		Id:                 id.String(),
		PairId:             pairId.String(),
		CharacterId:        42,
		PartnerCharacterId: 77,
		AssetId:            1001,
		ItemTemplateId:     1112800,
		RingType:           string(TypeCouple),
		State:              string(StateActive),
		CashId:             9007199254740993,
		PartnerCashId:      42,
		PartnerName:        "PartnerChar",
	}

	m, err := Extract(rm)
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if m.Id() != id {
		t.Errorf("Id() = %v, want %v", m.Id(), id)
	}
	if m.PairId() != pairId {
		t.Errorf("PairId() = %v, want %v", m.PairId(), pairId)
	}
	if m.CharacterId() != 42 {
		t.Errorf("CharacterId() = %d, want 42", m.CharacterId())
	}
	if m.PartnerCharacterId() != 77 {
		t.Errorf("PartnerCharacterId() = %d, want 77", m.PartnerCharacterId())
	}
	if m.ItemTemplateId() != 1112800 {
		t.Errorf("ItemTemplateId() = %d, want 1112800", m.ItemTemplateId())
	}
	if m.Type() != TypeCouple {
		t.Errorf("Type() = %s, want %s", m.Type(), TypeCouple)
	}
	if m.State() != StateActive {
		t.Errorf("State() = %s, want %s", m.State(), StateActive)
	}
	if m.CashId() != 9007199254740993 {
		t.Errorf("CashId() = %d, want 9007199254740993", m.CashId())
	}
	if m.PartnerCashId() != 42 {
		t.Errorf("PartnerCashId() = %d, want 42", m.PartnerCashId())
	}
	if m.PartnerName() != "PartnerChar" {
		t.Errorf("PartnerName() = %s, want PartnerChar", m.PartnerName())
	}
}

func TestExtractInvalidId(t *testing.T) {
	rm := RestModel{
		Id:       "not-a-uuid",
		PairId:   uuid.New().String(),
		RingType: string(TypeCouple),
		State:    string(StateActive),
	}

	_, err := Extract(rm)
	if err == nil {
		t.Fatal("Extract: expected error for invalid id, got nil")
	}
}

func TestExtractUnknownRingType(t *testing.T) {
	rm := RestModel{
		Id:       uuid.New().String(),
		PairId:   uuid.New().String(),
		RingType: "MYSTERY",
		State:    string(StateActive),
	}

	_, err := Extract(rm)
	if err == nil {
		t.Fatal("Extract: expected error for unknown ring type, got nil")
	}
}

func TestExtractLargeCashIdSurvives(t *testing.T) {
	rm := RestModel{
		Id:       uuid.New().String(),
		PairId:   uuid.New().String(),
		RingType: string(TypeCouple),
		State:    string(StateActive),
		CashId:   9007199254740993,
	}

	m, err := Extract(rm)
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if m.CashId() != 9007199254740993 {
		t.Errorf("CashId() = %d, want 9007199254740993", m.CashId())
	}
}
