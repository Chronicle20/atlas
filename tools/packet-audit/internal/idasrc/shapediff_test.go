package idasrc

import "testing"

func fc(op Primitive) FieldCall { return FieldCall{Op: op} }

func TestValidateShapeVerified(t *testing.T) {
	v, _ := ValidateShape([]FieldCall{fc(Decode1), fc(Decode4), fc(DecodeStr)},
		[]FieldCall{fc(Decode1), fc(Decode4), fc(DecodeStr)})
	if v != ShapeVerified {
		t.Errorf("v=%v want Verified", v)
	}
}

func TestValidateShapeRepresentationEquivalent(t *testing.T) {
	// hand read a 4-byte buffer; live read Decode4 — same bytes (audit tolerates).
	v, _ := ValidateShape([]FieldCall{fc(DecodeBuf)}, []FieldCall{fc(Decode4)})
	if v != ShapeVerified {
		t.Errorf("v=%v want Verified (DecodeBuf≡Decode4)", v)
	}
}

func TestValidateShapeDivergent(t *testing.T) {
	v, d := ValidateShape([]FieldCall{fc(Decode1), fc(Decode4)},
		[]FieldCall{fc(Decode1), fc(DecodeStr)})
	if v != ShapeDivergent {
		t.Errorf("v=%v want Divergent", v)
	}
	if d == "" {
		t.Error("divergent must carry a detail")
	}
}

func TestValidateShapeLengthDivergent(t *testing.T) {
	// live longer, NO Unresolved → divergent (extra field).
	v, _ := ValidateShape([]FieldCall{fc(Decode1)}, []FieldCall{fc(Decode1), fc(Decode4)})
	if v != ShapeDivergent {
		t.Errorf("v=%v want Divergent", v)
	}
}

func TestValidateShapeUnverifiable(t *testing.T) {
	// prefix matches, then live Unresolved (undecompilable span) → Unverifiable.
	v, _ := ValidateShape(
		[]FieldCall{fc(Decode1), fc(Decode4), fc(Decode4), fc(DecodeStr)},
		[]FieldCall{fc(Decode1), fc(Decode4), fc(Unresolved)})
	if v != ShapeUnverifiable {
		t.Errorf("v=%v want Unverifiable", v)
	}
}

func TestValidateShapeDivergentBeforeUnresolved(t *testing.T) {
	// divergence in the KNOWN part (before the Unresolved) still wins.
	v, _ := ValidateShape(
		[]FieldCall{fc(Decode1), fc(DecodeStr), fc(Decode4)},
		[]FieldCall{fc(Decode1), fc(Decode4), fc(Unresolved)})
	if v != ShapeDivergent {
		t.Errorf("v=%v want Divergent (known-part divergence before Unresolved)", v)
	}
}
