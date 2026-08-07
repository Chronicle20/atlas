package skill

import (
	"atlas-data/xml"
	"context"
	"strconv"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func commonTestContext(t *testing.T) (logrus.FieldLogger, *test.Hook, context.Context) {
	t.Helper()
	l, hook := test.NewNullLogger()
	tn, err := tenant.Create(uuid.New(), "GMS", 95, 1)
	if err != nil {
		t.Fatal(err)
	}
	return l, hook, tenant.WithContext(context.Background(), tn)
}

func readSkills(t *testing.T, l logrus.FieldLogger, ctx context.Context, data string) []RestModel {
	t.Helper()
	models, err := Read(l)(ctx)(xml.FromByteArrayProvider([]byte(data)))()
	if err != nil {
		t.Fatal(err)
	}
	return models
}

// TestCommonExpansion covers FR-5.1/FR-5.2: a common node expands to exactly
// maxLevel effects, evaluated at x = 1..maxLevel, and MaxLevel is the declared
// value. Values are skill 1001003's real v95.1 expressions.
func TestCommonExpansion(t *testing.T) {
	const xmlData = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<imgdir name="100.img">
  <imgdir name="skill">
    <imgdir name="1001003">
      <imgdir name="common">
        <int name="maxLevel" value="20"/>
        <string name="mpCon" value="6+2*u(x/5)"/>
        <string name="time" value="100+10*x"/>
      </imgdir>
    </imgdir>
  </imgdir>
</imgdir>`

	l, _, ctx := commonTestContext(t)
	models := readSkills(t, l, ctx, xmlData)
	if len(models) != 1 {
		t.Fatalf("len(models) = %d, want 1", len(models))
	}
	m := models[0]
	if m.MaxLevel != 20 {
		t.Fatalf("MaxLevel = %d, want 20", m.MaxLevel)
	}
	if len(m.Effects) != 20 {
		t.Fatalf("len(Effects) = %d, want 20", len(m.Effects))
	}
	// 6+2*u(1/5) = 6 + 2*1 = 8; time 100+10*1 = 110 s -> 110000 ms.
	if m.Effects[0].MPConsume != 8 {
		t.Fatalf("Effects[0].MPConsume = %d, want 8", m.Effects[0].MPConsume)
	}
	if m.Effects[0].Duration != 110000 {
		t.Fatalf("Effects[0].Duration = %d, want 110000", m.Effects[0].Duration)
	}
	// 6+2*u(20/5) = 6 + 2*4 = 14; time 100+10*20 = 300 s -> 300000 ms.
	if m.Effects[19].MPConsume != 14 {
		t.Fatalf("Effects[19].MPConsume = %d, want 14", m.Effects[19].MPConsume)
	}
	if m.Effects[19].Duration != 300000 {
		t.Fatalf("Effects[19].Duration = %d, want 300000", m.Effects[19].Duration)
	}
}

// TestCommonXKeyIsNotTheVariable pins FR-4.1: the common child named `x` is a
// skill parameter, never the expression variable. Modelled on skill 1101004,
// whose common/x is the literal "-2".
func TestCommonXKeyIsNotTheVariable(t *testing.T) {
	const xmlData = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<imgdir name="110.img">
  <imgdir name="skill">
    <imgdir name="1101004">
      <imgdir name="common">
        <int name="maxLevel" value="3"/>
        <string name="x" value="-2"/>
        <string name="mpCon" value="x"/>
      </imgdir>
    </imgdir>
  </imgdir>
</imgdir>`

	l, _, ctx := commonTestContext(t)
	m := readSkills(t, l, ctx, xmlData)[0]
	for i, ef := range m.Effects {
		if ef.X != -2 {
			t.Fatalf("Effects[%d].X = %d, want -2 at every level", i, ef.X)
		}
		if ef.MPConsume != uint16(i+1) {
			t.Fatalf("Effects[%d].MPConsume = %d, want %d (the level)", i, ef.MPConsume, i+1)
		}
	}
}

// TestCommonWinsOverLevel pins FR-1.2 and its accepted regression: skills
// 2211002/2211006 carry both subtrees; common wins, so maxLevel is 20 (not
// 30) and the level-only keys mad/mastery are lost.
func TestCommonWinsOverLevel(t *testing.T) {
	const xmlData = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<imgdir name="2211.img">
  <imgdir name="skill">
    <imgdir name="2211002">
      <imgdir name="common">
        <int name="maxLevel" value="20"/>
        <string name="mpCon" value="20+6*d(x/4)"/>
        <string name="damage" value="100+2*x"/>
      </imgdir>
      <imgdir name="level">
        <imgdir name="1">
          <int name="mpCon" value="21"/>
          <int name="mad" value="120"/>
          <int name="mastery" value="40"/>
        </imgdir>
        <imgdir name="2">
          <int name="mpCon" value="22"/>
          <int name="mad" value="125"/>
          <int name="mastery" value="45"/>
        </imgdir>
      </imgdir>
    </imgdir>
  </imgdir>
</imgdir>`

	l, _, ctx := commonTestContext(t)
	m := readSkills(t, l, ctx, xmlData)[0]
	if m.MaxLevel != 20 {
		t.Fatalf("MaxLevel = %d, want 20 (common wins over level's 2)", m.MaxLevel)
	}
	if len(m.Effects) != 20 {
		t.Fatalf("len(Effects) = %d, want 20", len(m.Effects))
	}
	// 20+6*d(1/4) = 20 + 6*0 = 20, NOT the level subtree's 21.
	if m.Effects[0].MPConsume != 20 {
		t.Fatalf("Effects[0].MPConsume = %d, want 20", m.Effects[0].MPConsume)
	}
	// Accepted regression: mad/mastery live only under `level` and are lost.
	if m.Effects[0].MagicAttack != 0 {
		t.Fatalf("Effects[0].MagicAttack = %d, want 0 (accepted FR-1.2 regression)", m.Effects[0].MagicAttack)
	}
	if m.Effects[0].Mastery != 0 {
		t.Fatalf("Effects[0].Mastery = %d, want 0 (accepted FR-1.2 regression)", m.Effects[0].Mastery)
	}
}

// TestCommonMaxLevelNodeTypes covers FR-2.3: maxLevel is an int node 632x and
// a string node 3x (13100004, 1320005, 32120001). Both must parse.
func TestCommonMaxLevelNodeTypes(t *testing.T) {
	const xmlData = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<imgdir name="1310.img">
  <imgdir name="skill">
    <imgdir name="13100004">
      <imgdir name="common">
        <string name="maxLevel" value="10"/>
        <string name="mpCon" value="x"/>
      </imgdir>
    </imgdir>
    <imgdir name="13100005">
      <imgdir name="common">
        <int name="maxLevel" value="5"/>
        <string name="mpCon" value="x"/>
      </imgdir>
    </imgdir>
  </imgdir>
</imgdir>`

	l, _, ctx := commonTestContext(t)
	got := map[string]uint8{}
	for _, m := range readSkills(t, l, ctx, xmlData) {
		got[strconv.Itoa(int(m.Id))] = m.MaxLevel
	}
	if got["13100004"] != 10 {
		t.Fatalf("13100004 MaxLevel = %d, want 10 (string-typed node)", got["13100004"])
	}
	if got["13100005"] != 5 {
		t.Fatalf("13100005 MaxLevel = %d, want 5 (int-typed node)", got["13100005"])
	}
}

// TestCommonIntLeavesAndPassThrough covers FR-2.4/2.5/2.6: int-typed leaves
// parse as constants, lt/rb vectors pass through unevaluated, and `action` is
// never evaluated.
func TestCommonIntLeavesAndPassThrough(t *testing.T) {
	const xmlData = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<imgdir name="311.img">
  <imgdir name="skill">
    <imgdir name="3111003">
      <imgdir name="common">
        <int name="maxLevel" value="2"/>
        <int name="time" value="0"/>
        <int name="x" value="1"/>
        <string name="action" value="slashStorm2"/>
        <vector name="lt" x="-150" y="-100"/>
        <vector name="rb" x="150" y="100"/>
      </imgdir>
    </imgdir>
  </imgdir>
</imgdir>`

	l, _, ctx := commonTestContext(t)
	m := readSkills(t, l, ctx, xmlData)[0]
	for i, ef := range m.Effects {
		if ef.X != 1 {
			t.Fatalf("Effects[%d].X = %d, want 1 (int leaf)", i, ef.X)
		}
		if ef.LT == nil || ef.LT.X != -150 || ef.LT.Y != -100 {
			t.Fatalf("Effects[%d].LT = %+v, want (-150,-100)", i, ef.LT)
		}
		if ef.RB == nil || ef.RB.X != 150 || ef.RB.Y != 100 {
			t.Fatalf("Effects[%d].RB = %+v, want (150,100)", i, ef.RB)
		}
		// time 0 is > -1, so getEffect converts seconds -> ms and flags overTime.
		if ef.Duration != 0 || !ef.OverTime {
			t.Fatalf("Effects[%d] duration/overTime = (%d,%v), want (0,true)", i, ef.Duration, ef.OverTime)
		}
	}
}

// TestCommonNeitherSubtree covers FR-1.3: a skill with neither subtree yields
// zero effects and does not panic.
func TestCommonNeitherSubtree(t *testing.T) {
	const xmlData = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<imgdir name="100.img">
  <imgdir name="skill">
    <imgdir name="1000000">
      <int name="skillType" value="0"/>
    </imgdir>
  </imgdir>
</imgdir>`

	l, _, ctx := commonTestContext(t)
	m := readSkills(t, l, ctx, xmlData)[0]
	if m.MaxLevel != 0 || len(m.Effects) != 0 {
		t.Fatalf("(MaxLevel,len(Effects)) = (%d,%d), want (0,0)", m.MaxLevel, len(m.Effects))
	}
}

// TestSynthesizedNodesAreCanonicalIntegers pins FR-7.5 / design §2.2: no
// expression text ever reaches xml.GetIntegerWithDefault, because synthesis
// writes only canonical strconv output.
func TestSynthesizedNodesAreCanonicalIntegers(t *testing.T) {
	common := xml.Node{
		Name: "common",
		IntegerNodes: []xml.IntegerNode{
			{Name: "maxLevel", Value: "3"},
		},
		StringNodes: []xml.StringNode{
			{Name: "mpCon", Value: "6+2*u(x/5)"},
			{Name: "damage", Value: " 375+5*x"},
			{Name: "x", Value: "-2"},
		},
	}
	l, _, ctx := commonTestContext(t)
	tn := tenant.MustFromContext(ctx)
	nodes, maxLevel, failures := synthesizeCommonNodes(l, tn, 100, 1001003, &common)
	if maxLevel != 3 {
		t.Fatalf("maxLevel = %d, want 3", maxLevel)
	}
	if failures != 0 {
		t.Fatalf("failures = %d, want 0", failures)
	}
	if len(nodes) != 3 {
		t.Fatalf("len(nodes) = %d, want 3", len(nodes))
	}
	for _, n := range nodes {
		if _, err := strconv.Atoi(n.Name); err != nil {
			t.Fatalf("node name %q is not a level number", n.Name)
		}
		for _, in := range n.IntegerNodes {
			if _, err := strconv.ParseInt(in.Value, 10, 64); err != nil {
				t.Fatalf("synthesized %s = %q does not round-trip through ParseInt", in.Name, in.Value)
			}
		}
		if len(n.StringNodes) != 0 {
			t.Fatalf("synthesized node carries %d string nodes, want 0", len(n.StringNodes))
		}
	}
}

// TestCommonExprAboveMaxInt32IsRangeViolation pins the fix for a
// review-caught FR-7.5 regression: `damage` (and its uint32 siblings) lands
// in a synthesized xml.IntegerNode consumed by xml.GetIntegerWithDefault
// (xml/model.go), which parses with strconv.ParseInt(..., 32) — a SIGNED
// 32-bit parse — regardless of the destination Go field's width. A value in
// [2147483648, 4294967295] must therefore be rejected as a range violation
// at synthesis time (loud, counted), never written into the node: writing it
// would let ParseInt fail downstream and silently degrade to
// GetIntegerWithDefault's default, exactly the failure mode this table
// exists to prevent.
func TestCommonExprAboveMaxInt32IsRangeViolation(t *testing.T) {
	common := xml.Node{
		Name: "common",
		IntegerNodes: []xml.IntegerNode{
			{Name: "maxLevel", Value: "1"},
		},
		StringNodes: []xml.StringNode{
			// 2147483647 + 1 = 2147483648, one past math.MaxInt32, well
			// within maxUint32 (the field's Go width) — exactly the gap the
			// review finding identified.
			{Name: "damage", Value: "2147483647+x"},
		},
	}
	l, hook, ctx := commonTestContext(t)
	tn := tenant.MustFromContext(ctx)
	nodes, maxLevel, failures := synthesizeCommonNodes(l, tn, 100, 1001003, &common)
	if maxLevel != 1 {
		t.Fatalf("maxLevel = %d, want 1", maxLevel)
	}
	if failures != 1 {
		t.Fatalf("failures = %d, want 1 (the out-of-range damage value)", failures)
	}
	if len(nodes) != 1 {
		t.Fatalf("len(nodes) = %d, want 1", len(nodes))
	}
	for _, in := range nodes[0].IntegerNodes {
		if in.Name == "damage" {
			t.Fatalf("synthesized node carries damage=%q, want it dropped by the range check", in.Value)
		}
	}
	entries := hook.AllEntries()
	if len(entries) != 1 {
		t.Fatalf("logged %d entries, want 1 ERROR", len(entries))
	}
	if entries[0].Level != logrus.ErrorLevel {
		t.Fatalf("logged at %s, want ERROR", entries[0].Level)
	}
	if entries[0].Data["key"] != "damage" {
		t.Fatalf("logged key = %v, want \"damage\"", entries[0].Data["key"])
	}
}
