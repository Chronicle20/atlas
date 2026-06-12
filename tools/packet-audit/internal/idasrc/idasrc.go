package idasrc

import "context"

type Direction int

const (
	DirClientbound Direction = iota
	DirServerbound
)

type Primitive int

const (
	Decode1    Primitive = iota // ReadByte/WriteByte
	Decode2                     // ReadShort/WriteShort
	Decode4                     // ReadInt/WriteInt
	Decode8                     // ReadLong/WriteLong
	DecodeStr                   // ReadAsciiString/WriteAsciiString
	DecodeBuf                   // ReadBytes/WriteBytes
	Unresolved                  // parser could not prove this element; audit treats as a known gap
)

// RawOp maps a Primitive to its canonical export op string (the inverse of
// parsePrim's Decode* arm). This is the form written into a baseline JSON
// rawCall "op" field. Decode8 → "Decode8", DecodeStr → "DecodeStr",
// DecodeBuf → "DecodeBuf", Unresolved → "Unresolved". An out-of-range value
// returns "Unresolved" (a safe known-gap marker, never a fabricated read).
func (p Primitive) RawOp() string {
	switch p {
	case Decode1:
		return "Decode1"
	case Decode2:
		return "Decode2"
	case Decode4:
		return "Decode4"
	case Decode8:
		return "Decode8"
	case DecodeStr:
		return "DecodeStr"
	case DecodeBuf:
		return "DecodeBuf"
	case Unresolved:
		return "Unresolved"
	}
	return "Unresolved"
}

func (p Primitive) String() string {
	switch p {
	case Decode1:
		return "byte"
	case Decode2:
		return "int16"
	case Decode4:
		return "int32"
	case Decode8:
		return "int64"
	case DecodeStr:
		return "string"
	case DecodeBuf:
		return "bytes"
	case Unresolved:
		return "unresolved"
	}
	return "unknown"
}

type FieldCall struct {
	Op      Primitive
	Comment string
	Guard   string // free-form expression text; "" if unconditional
}

type Fields struct {
	Function  string
	Address   string
	Direction Direction
	Calls     []FieldCall
	// CaseLabels is the full dispatch case-label set per discriminator, collected
	// from the function's switch/if-else structure independent of whether each arm
	// reads. Populated by ParseDecompileFields (and ResolveLive). nil when not
	// collected. Used by the case<->mode bijection check.
	CaseLabels map[string]*CaseSet
	// HasMultiwayDispatch is true when the function dispatches multiple ways on one
	// discriminator: a switch with >=2 case labels, or an if/else chain with >=2
	// arms on one discriminator. A lone optional-field if, or a single-case switch,
	// is NOT multi-way. Used to decide whether an empty-dispatch #Mode entry is a
	// leaf (flat-validate) or a genuine dispatcher (unverifiable without a selector).
	HasMultiwayDispatch bool
}

// CaseSet is the ordered set of dispatch case labels seen for one discriminator,
// plus whether a default/else arm exists. Order is first-seen for determinism.
type CaseSet struct {
	cases   []int64
	seen    map[int64]bool
	Default bool
}

// NewCaseSet builds a CaseSet from case values (first-seen order, deduped). Used
// by callers that accumulate a handler's client case-set across addresses.
func NewCaseSet(vals []int64) *CaseSet {
	cs := &CaseSet{}
	for _, v := range vals {
		cs.add(v)
	}
	return cs
}

func (c *CaseSet) add(v int64) {
	if c.seen == nil {
		c.seen = map[int64]bool{}
	}
	if !c.seen[v] {
		c.seen[v] = true
		c.cases = append(c.cases, v)
	}
}

// Has reports whether v is a known case label.
func (c *CaseSet) Has(v int64) bool { return c.seen[v] }

// Values returns the case labels in first-seen order (a defensive copy).
func (c *CaseSet) Values() []int64 { return append([]int64(nil), c.cases...) }

type Source interface {
	Resolve(ctx context.Context, fname string) (Fields, error)
}
