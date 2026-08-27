package character

import (
	"reflect"
	"testing"
)

func TestTransformRoundTrip(t *testing.T) {
	m := Model{
		id:                 11,
		accountId:          22,
		worldId:            33,
		name:               "field4",
		gender:             55,
		skinColor:          66,
		face:               77,
		hair:               88,
		level:              99,
		jobId:              110,
		strength:           121,
		dexterity:          132,
		intelligence:       143,
		luck:               154,
		hp:                 165,
		maxHp:              176,
		mp:                 187,
		maxMp:              198,
		hpMpUsed:           209,
		ap:                 220,
		sp:                 "field21",
		experience:         242,
		fame:               253,
		gachaponExperience: 264,
		spawnPoint:         275,
		gm:                 286,
		x:                  297,
		y:                  308,
		meso:               319,
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
