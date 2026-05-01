package characterimage

import "testing"

func TestFilterEquipmentDropsMountPetCash(t *testing.T) {
	in := map[int]int{
		-1:   1002357, // hat
		-11:  1402024, // weapon
		-14:  5000000, // pet — drop
		-18:  1932000, // mount saddle — drop
		-19:  1932001, // mount — drop
		-21:  1012000, // pet ring slot — drop
		-101: 1002001, // cash hat — drop
		-114: 1132001, // cash belt — drop
	}
	out := FilterEquipment(in)
	if _, ok := out[-1]; !ok {
		t.Fatal("hat dropped")
	}
	if _, ok := out[-11]; !ok {
		t.Fatal("weapon dropped")
	}
	for _, slot := range []int{-14, -18, -19, -21, -101, -114} {
		if _, ok := out[slot]; ok {
			t.Fatalf("slot %d not dropped", slot)
		}
	}
}
