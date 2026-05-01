package characterimage

import "testing"

// Body has its own neck joint at (-4,-32) relative to its origin.
// Hair declares origin (10, 50) and a complementary neck joint at (5, 12) on
// itself. When attached to body anchored at canvas (48, 96), the hair anchor
// must be placed so hair.neck on canvas == body.neck on canvas.
func TestResolveAnchorJoinsByJointName(t *testing.T) {
	body := PartMeta{Origin: Vec{X: 19, Y: 32}, Map: map[string]Vec{"neck": {X: -4, Y: -32}}}
	bodyAnchor := Anchor{X: 48, Y: 96}

	hair := PartMeta{Origin: Vec{X: 10, Y: 50}, Map: map[string]Vec{"neck": {X: 5, Y: 12}}}

	got := ResolveAnchor(bodyAnchor, body, hair, "neck")

	// body.neck on canvas = bodyAnchor + body.map.neck
	//                     = (48,96) + (-4,-32) = (44, 64)
	// hair.origin on canvas must be at (body.neck.canvas - hair.map.neck)
	//                                  = (44 - 5, 64 - 12) = (39, 52)
	if got != (Anchor{X: 39, Y: 52}) {
		t.Fatalf("ResolveAnchor = %+v, want {39 52}", got)
	}
}
