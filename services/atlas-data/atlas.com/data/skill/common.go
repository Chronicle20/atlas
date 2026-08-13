package skill

import (
	"atlas-data/skill/effect"
	"atlas-data/skill/formula"
	"atlas-data/xml"
	"fmt"
	"math"
	"strconv"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// commonKind classifies a `common` child leaf.
type commonKind int

const (
	// commonExpr is evaluated once per level with formula.Evaluate.
	commonExpr commonKind = iota
	// commonVector passes through unevaluated (FR-2.4).
	commonVector
	// commonOpaque is a client animation name, never evaluated (FR-2.5).
	commonOpaque
	// commonMeta is skill-level, not per-effect (maxLevel).
	commonMeta
)

// commonKeyDef declares how one `common` child key is handled and, for
// expressions, the inclusive range its target effect field admits.
//
// The range check exists because every key lands in a narrow Go type: an
// unchecked narrowing turns an evaluated -2 into uint16(65534). A violation is
// a loud FR-7.1 failure, never a silent wrap. It applies ONLY to the `common`
// path — the `level` path's conversions are untouched (FR-6.5).
//
// The upper bound is capped at maxInt32 for EVERY key, even ones whose
// destination Go field is a uint32 (mobCount, cooltime, morph, damage,
// attackCount, moneyCon, itemCon, itemConNo). That is not the destination
// field's width — it is the transport's: every synthesized node is consumed
// by xml.GetIntegerWithDefault (xml/model.go:82-102), which parses with
// strconv.ParseInt(c.Value, 10, 32), a SIGNED 32-bit parse. A value in
// [2147483648, 4294967295] would pass a maxUint32 range check here, get
// written into the node, and then silently fail that ParseInt — degrading to
// GetIntegerWithDefault's default (0, or 1 for mobCount) one call downstream
// of the FR-7.5 guard this table exists to enforce. Do not widen a bound past
// maxInt32 to match a uint32/uint16 destination field without first checking
// whether that key is read via GetIntegerWithDefault (it is, for every key in
// this table — none go through a wider or unsigned parse).
type commonKeyDef struct {
	name string
	kind commonKind
	min  int64
	max  int64
}

const (
	minInt16 = math.MinInt16
	maxInt16 = math.MaxInt16
	minInt32 = math.MinInt32
	maxInt32 = math.MaxInt32
	// maxUint16 is a destination-field bound (e.g. hp/mp/mpCon land in a
	// uint16), which is narrower than the maxInt32 transport bound below and
	// so is safe to use directly.
	maxUint16 = math.MaxUint16
	// getEffect multiplies a non-negative `time` by 1000 to convert wz
	// seconds to milliseconds; bound it so that product still fits int32.
	maxTimeSeconds = math.MaxInt32 / 1000
)

// commonKeys is the declared table: the single source of truth for `common`
// handling, and the iteration order for synthesis (NFR-4 determinism — never
// range a Go map into serialized output). It covers all 65 distinct keys the
// GMS v95.1 archive uses.
var commonKeys = []commonKeyDef{
	// Skill-level metadata and pass-throughs.
	{name: "maxLevel", kind: commonMeta},
	{name: "action", kind: commonOpaque},
	{name: "lt", kind: commonVector},
	{name: "rb", kind: commonVector},

	// Keys the effect model already carried before task-192.
	{name: "time", kind: commonExpr, min: -1, max: maxTimeSeconds},
	{name: "hp", kind: commonExpr, min: 0, max: maxUint16},
	{name: "mp", kind: commonExpr, min: 0, max: maxUint16},
	{name: "hpCon", kind: commonExpr, min: 0, max: maxUint16},
	{name: "mpCon", kind: commonExpr, min: 0, max: maxUint16},
	{name: "prop", kind: commonExpr, min: minInt32, max: maxInt32},
	{name: "mobCount", kind: commonExpr, min: 0, max: maxInt32},
	{name: "cooltime", kind: commonExpr, min: 0, max: maxInt32},
	{name: "morph", kind: commonExpr, min: 0, max: maxInt32},
	{name: "pad", kind: commonExpr, min: minInt16, max: maxInt16},
	{name: "pdd", kind: commonExpr, min: minInt16, max: maxInt16},
	{name: "mad", kind: commonExpr, min: minInt16, max: maxInt16},
	{name: "mdd", kind: commonExpr, min: minInt16, max: maxInt16},
	{name: "acc", kind: commonExpr, min: minInt16, max: maxInt16},
	{name: "eva", kind: commonExpr, min: minInt16, max: maxInt16},
	{name: "speed", kind: commonExpr, min: minInt16, max: maxInt16},
	{name: "jump", kind: commonExpr, min: minInt16, max: maxInt16},
	{name: "x", kind: commonExpr, min: minInt16, max: maxInt16},
	{name: "y", kind: commonExpr, min: minInt16, max: maxInt16},
	{name: "damage", kind: commonExpr, min: 0, max: maxInt32},
	{name: "attackCount", kind: commonExpr, min: 0, max: maxInt32},
	{name: "bulletCount", kind: commonExpr, min: 0, max: maxUint16},
	{name: "bulletConsume", kind: commonExpr, min: 0, max: maxUint16},
	{name: "moneyCon", kind: commonExpr, min: 0, max: maxInt32},
	{name: "itemCon", kind: commonExpr, min: 0, max: maxInt32},
	{name: "itemConNo", kind: commonExpr, min: 0, max: maxInt32},

	// Keys task-192 added to the effect model.
	{name: "mhpR", kind: commonExpr, min: 0, max: maxUint16},
	{name: "mmpR", kind: commonExpr, min: 0, max: maxUint16},
	{name: "range", kind: commonExpr, min: minInt32, max: maxInt32},
	{name: "mastery", kind: commonExpr, min: minInt32, max: maxInt32},
	{name: "z", kind: commonExpr, min: minInt32, max: maxInt32},
	{name: "dot", kind: commonExpr, min: minInt32, max: maxInt32},
	{name: "cr", kind: commonExpr, min: minInt32, max: maxInt32},
	{name: "dotInterval", kind: commonExpr, min: minInt32, max: maxInt32},
	{name: "dotTime", kind: commonExpr, min: minInt32, max: maxInt32},
	{name: "damR", kind: commonExpr, min: minInt32, max: maxInt32},
	{name: "criticaldamageMin", kind: commonExpr, min: minInt32, max: maxInt32},
	{name: "v", kind: commonExpr, min: minInt32, max: maxInt32},
	{name: "ignoreMobpdpR", kind: commonExpr, min: minInt32, max: maxInt32},
	{name: "epad", kind: commonExpr, min: minInt32, max: maxInt32},
	{name: "w", kind: commonExpr, min: minInt32, max: maxInt32},
	{name: "u", kind: commonExpr, min: minInt32, max: maxInt32},
	{name: "epdd", kind: commonExpr, min: minInt32, max: maxInt32},
	{name: "emdd", kind: commonExpr, min: minInt32, max: maxInt32},
	{name: "selfDestruction", kind: commonExpr, min: minInt32, max: maxInt32},
	{name: "asrR", kind: commonExpr, min: minInt32, max: maxInt32},
	{name: "t", kind: commonExpr, min: minInt32, max: maxInt32},
	{name: "er", kind: commonExpr, min: minInt32, max: maxInt32},
	{name: "pddR", kind: commonExpr, min: minInt32, max: maxInt32},
	{name: "terR", kind: commonExpr, min: minInt32, max: maxInt32},
	{name: "madX", kind: commonExpr, min: minInt32, max: maxInt32},
	{name: "subProp", kind: commonExpr, min: minInt32, max: maxInt32},
	{name: "emhp", kind: commonExpr, min: minInt32, max: maxInt32},
	{name: "criticaldamageMax", kind: commonExpr, min: minInt32, max: maxInt32},
	{name: "expR", kind: commonExpr, min: minInt32, max: maxInt32},
	{name: "emmp", kind: commonExpr, min: minInt32, max: maxInt32},
	{name: "itemConsume", kind: commonExpr, min: minInt32, max: maxInt32},
	{name: "mddR", kind: commonExpr, min: minInt32, max: maxInt32},
	{name: "subTime", kind: commonExpr, min: minInt32, max: maxInt32},
	{name: "padX", kind: commonExpr, min: minInt32, max: maxInt32},
	{name: "mesoR", kind: commonExpr, min: minInt32, max: maxInt32},
}

var commonKeyIndex = func() map[string]commonKeyDef {
	m := make(map[string]commonKeyDef, len(commonKeys))
	for _, k := range commonKeys {
		m[k.name] = k
	}
	return m
}()

// unknownCommonKey is the fallback for a key the table does not declare. It is
// carried (it costs nothing) but logged at WARN so a future archive that
// introduces a key is noticed rather than silently dropped.
func unknownCommonKey(name string) commonKeyDef {
	return commonKeyDef{name: name, kind: commonExpr, min: minInt32, max: maxInt32}
}

// commonLeaf is one classified, already-parsed `common` child.
type commonLeaf struct {
	def  commonKeyDef
	expr formula.Expr
	src  string
}

// lookupCommonLeaf returns the raw text of a `common` child leaf. `common` is
// exactly one level deep archive-wide (FR-2.1), and its leaves are string, int
// or vector only (FR-2.2) — an int-typed leaf's decimal text parses as a
// constant expression (FR-2.6).
func lookupCommonLeaf(node *xml.Node, name string) (string, bool) {
	for _, c := range node.IntegerNodes {
		if c.Name == name {
			return c.Value, true
		}
	}
	for _, c := range node.StringNodes {
		if c.Name == name {
			return c.Value, true
		}
	}
	return "", false
}

// commonMaxLevel reads the declared maxLevel (FR-2.3, FR-5.2). A missing or
// non-positive-integer value is an FR-7.4 failure for the whole skill: with no
// level count there is nothing to expand.
func commonMaxLevel(node *xml.Node) (int, error) {
	raw, ok := lookupCommonLeaf(node, "maxLevel")
	if !ok {
		return 0, fmt.Errorf("common node has no maxLevel")
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("maxLevel %q is not an integer: %w", raw, err)
	}
	if v <= 0 || v > 255 {
		return 0, fmt.Errorf("maxLevel %d is out of the 1..255 range", v)
	}
	return v, nil
}

// logCommonFailure emits the FR-7.1 ERROR. Fields are structured, not a
// formatted sentence, so a run is greppable by skill id (NFR-3).
func logCommonFailure(l logrus.FieldLogger, t tenant.Model, jobId uint32, skillId skill.Id, key string, expression string, level int, err error) {
	l.WithFields(logrus.Fields{
		"tenant":     t.Id().String(),
		"jobId":      jobId,
		"skillId":    uint32(skillId),
		"key":        key,
		"expression": expression,
		"level":      level,
	}).WithError(err).Errorf("Unable to derive skill effect from Skill.wz common node.")
}

// synthesizeCommonNodes expands a `common` subtree into one xml.Node per level
// (design §2.1 option C). The synthetic node's Name is the level number so
// levelFromNode keeps working, and it carries no child nodes — matching a
// `level` node that has no mob/0 subtree, which is what the archive's
// one-level-deep `common` means (FR-2.1).
//
// Only canonical strconv.FormatInt output is written, so no expression text
// can reach xml.GetIntegerWithDefault's silent default (FR-7.5).
//
// It returns the synthesized nodes, the DECLARED maxLevel (FR-5.2 — the
// authoritative value, not a derived count), and the failure count.
func synthesizeCommonNodes(l logrus.FieldLogger, t tenant.Model, jobId uint32, skillId skill.Id, common *xml.Node) ([]xml.Node, int, int) {
	failures := 0

	maxLevel, err := commonMaxLevel(common)
	if err != nil {
		raw, _ := lookupCommonLeaf(common, "maxLevel")
		logCommonFailure(l, t, jobId, skillId, "maxLevel", raw, 0, err)
		return nil, 0, failures + 1
	}

	// Classify and parse ONCE per key (FR-8.3): maxLevel is at most 255, so
	// this replaces up to 255 parses per key with one.
	leaves := make([]commonLeaf, 0, len(common.IntegerNodes)+len(common.StringNodes))
	seen := make(map[string]bool, len(common.IntegerNodes)+len(common.StringNodes))
	appendLeaf := func(name string, value string) {
		if seen[name] {
			return
		}
		seen[name] = true
		def, known := commonKeyIndex[name]
		if !known {
			def = unknownCommonKey(name)
			l.WithFields(logrus.Fields{
				"tenant":  t.Id().String(),
				"jobId":   jobId,
				"skillId": uint32(skillId),
				"key":     name,
			}).Warnf("Skill.wz common node carries an undeclared key; carrying it as a plain expression.")
		}
		if def.kind != commonExpr {
			return
		}
		e, err := formula.Parse(value)
		if err != nil {
			logCommonFailure(l, t, jobId, skillId, name, value, 0, err)
			failures++
			return
		}
		leaves = append(leaves, commonLeaf{def: def, expr: e, src: value})
	}

	// Declared-table order first, then any undeclared keys in archive order.
	for _, def := range commonKeys {
		if v, ok := lookupCommonLeaf(common, def.name); ok {
			appendLeaf(def.name, v)
		}
	}
	for _, c := range common.IntegerNodes {
		appendLeaf(c.Name, c.Value)
	}
	for _, c := range common.StringNodes {
		appendLeaf(c.Name, c.Value)
	}

	nodes := make([]xml.Node, 0, maxLevel)
	for level := 1; level <= maxLevel; level++ {
		synth := xml.Node{Name: strconv.Itoa(level)}
		for _, leaf := range leaves {
			v, err := leaf.expr.Evaluate(level)
			if err != nil {
				logCommonFailure(l, t, jobId, skillId, leaf.def.name, leaf.src, level, err)
				failures++
				continue
			}
			if v < leaf.def.min || v > leaf.def.max {
				logCommonFailure(l, t, jobId, skillId, leaf.def.name, leaf.src, level,
					fmt.Errorf("value %d is outside the target field range [%d,%d]", v, leaf.def.min, leaf.def.max))
				failures++
				continue
			}
			synth.IntegerNodes = append(synth.IntegerNodes, xml.IntegerNode{
				Name:  leaf.def.name,
				Value: strconv.FormatInt(v, 10),
			})
		}
		// lt/rb vectors pass through verbatim (FR-2.4).
		synth.PointNodes = append(synth.PointNodes, common.PointNodes...)
		nodes = append(nodes, synth)
	}
	return nodes, maxLevel, failures
}

// expandCommon derives a skill's effects from its `common` subtree. Every
// effect goes through the SAME getEffect the `level` path uses, so the two
// paths share one post-processing implementation (FR-5.3) and one set of
// defaults for absent keys (FR-5.4).
func expandCommon(l logrus.FieldLogger, t tenant.Model, jobId uint32, skillId skill.Id, buff bool, common *xml.Node) ([]effect.RestModel, uint8, Stats) {
	stats := Stats{}
	nodes, maxLevel, failures := synthesizeCommonNodes(l, t, jobId, skillId, common)
	stats.Failures = failures
	if failures > 0 {
		stats.SkillsWithFailures = 1
	}

	es := make([]effect.RestModel, 0, len(nodes))
	for _, n := range nodes {
		es = append(es, getEffect(t, skillId, buff, n))
	}
	// FR-5.2: the declared maxLevel is authoritative. commonMaxLevel already
	// bounded it to 1..255, and a failed parse yields 0 with zero effects.
	return es, uint8(maxLevel), stats
}
