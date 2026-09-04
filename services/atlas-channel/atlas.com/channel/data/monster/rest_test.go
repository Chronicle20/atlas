package monster

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestExtract(t *testing.T) {
	rm := RestModel{Boss: true, FixedDamage: 5, TagColor: 6, TagBackgroundColor: 1}
	if err := rm.SetID("8510000"); err != nil {
		t.Fatal(err)
	}
	m, err := Extract(rm)
	if err != nil {
		t.Fatal(err)
	}
	if m.Id() != 8510000 {
		t.Errorf("Id=%d, want 8510000", m.Id())
	}
	if !m.Boss() {
		t.Error("Boss=false, want true")
	}
	if m.FixedDamage() != 5 {
		t.Errorf("FixedDamage=%d, want 5", m.FixedDamage())
	}
	if m.TagColor() != 6 {
		t.Errorf("TagColor=%d, want 6", m.TagColor())
	}
	if m.TagBackgroundColor() != 1 {
		t.Errorf("TagBackgroundColor=%d, want 1", m.TagBackgroundColor())
	}
}

func TestTransformRoundTrip(t *testing.T) {
	m := Model{
		id:                 11,
		boss:               true,
		fixedDamage:        33,
		tagColor:           6,
		tagBackgroundColor: 1,
	}
	rm, err := Transform(m)
	if err != nil {
		t.Fatalf("transform: %v", err)
	}
	got, err := Extract(rm)
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	if !reflect.DeepEqual(got, m) {
		t.Errorf("round trip lost data:\n got %+v\nwant %+v", got, m)
	}
}

func TestUnmarshalDataPayload(t *testing.T) {
	payload := []byte(`{"boss":true,"fixed_damage":0,"tag_color":6,"tag_background_color":1,"hp":42000000,"level":80}`)

	var rm RestModel
	if err := json.Unmarshal(payload, &rm); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	m, err := Extract(rm)
	if err != nil {
		t.Fatal(err)
	}
	if !m.Boss() {
		t.Error("Boss=false, want true")
	}
	if m.FixedDamage() != 0 {
		t.Errorf("FixedDamage=%d, want 0", m.FixedDamage())
	}
	if m.TagColor() != 6 {
		t.Errorf("TagColor=%d, want 6", m.TagColor())
	}
	if m.TagBackgroundColor() != 1 {
		t.Errorf("TagBackgroundColor=%d, want 1", m.TagBackgroundColor())
	}
}
