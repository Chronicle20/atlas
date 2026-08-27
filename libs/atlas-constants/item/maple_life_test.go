package item

import "testing"

func TestMapleLifeItemIds(t *testing.T) {
	if MapleLifeATypeId != Id(5431000) {
		t.Errorf("MapleLifeATypeId = %d, want 5431000", MapleLifeATypeId)
	}
	if MapleLifeBTypeId != Id(5432000) {
		t.Errorf("MapleLifeBTypeId = %d, want 5432000", MapleLifeBTypeId)
	}
}
