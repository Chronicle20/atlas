package map_

import "testing"

func TestExtractDoorPortal(t *testing.T) {
	rm := PortalRestModel{Name: "tp", Type: 6, X: -100, Y: 200, TargetMapId: 999}
	p, err := ExtractPortal(rm)
	if err != nil || p.Type() != 6 || p.X() != -100 || p.TargetMapId() != 999 {
		t.Fatalf("portal extract wrong: %+v err=%v", p, err)
	}
}
