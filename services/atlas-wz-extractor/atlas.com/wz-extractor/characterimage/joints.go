package characterimage

import "fmt"

// Anchor is a canvas-space coordinate at which a part's `origin` is placed.
type Anchor struct{ X, Y int }

// ResolveAnchor computes the canvas anchor for a child part attached to a
// parent part via joint name. The child's `Map[joint]` describes the
// complementary point on the child sprite that should align with the
// parent's joint coordinate.
//
//	parentJointCanvas = parentAnchor + parent.Map[joint]
//	childAnchor       = parentJointCanvas - child.Map[joint]
//
// The child's `Origin` lands at `childAnchor`.
func ResolveAnchor(parentAnchor Anchor, parent, child PartMeta, joint string) Anchor {
	pj := parent.Map[joint]
	cj := child.Map[joint]
	return Anchor{
		X: parentAnchor.X + pj.X - cj.X,
		Y: parentAnchor.Y + pj.Y - cj.Y,
	}
}

// MustHaveJoint returns an error if either part lacks `joint`.
func MustHaveJoint(parent, child PartMeta, joint string) error {
	if _, ok := parent.Map[joint]; !ok {
		return fmt.Errorf("parent missing joint %q", joint)
	}
	if _, ok := child.Map[joint]; !ok {
		return fmt.Errorf("child missing joint %q", joint)
	}
	return nil
}
