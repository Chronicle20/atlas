package holding

import "testing"

// TestExtract asserts the holding RestModel -> Model extraction carries the
// fields the ENTER_MTS holding announce renders into an ITCITEM (itcSn, owner,
// template, quantity).
func TestExtract(t *testing.T) {
	m, err := Extract(RestModel{
		Id:         "11111111-1111-1111-1111-111111111111",
		WorldId:    1,
		ItcSn:      4242,
		OwnerId:    100100,
		Origin:     "purchased",
		TemplateId: 1302000,
		Quantity:   3,
	})
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	if m.Id() != "11111111-1111-1111-1111-111111111111" {
		t.Errorf("id = %q", m.Id())
	}
	if byte(m.WorldId()) != 1 {
		t.Errorf("worldId = %d, want 1", byte(m.WorldId()))
	}
	if m.ItcSn() != 4242 {
		t.Errorf("itcSn = %d, want 4242", m.ItcSn())
	}
	if m.OwnerId() != 100100 {
		t.Errorf("ownerId = %d, want 100100", m.OwnerId())
	}
	if m.TemplateId() != 1302000 {
		t.Errorf("templateId = %d, want 1302000", m.TemplateId())
	}
	if m.Quantity() != 3 {
		t.Errorf("quantity = %d, want 3", m.Quantity())
	}
}

// TestResource asserts the holding read endpoint path template matches atlas-mts's
// GET /characters/{characterId}/mts/holding.
func TestResource(t *testing.T) {
	if Resource != "characters/%d/mts/holding" {
		t.Errorf("Resource = %q, want characters/%%d/mts/holding", Resource)
	}
}
