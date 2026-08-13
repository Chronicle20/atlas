package echoofhero

import (
	"atlas-channel/data/skill/effect"
	"atlas-channel/data/skill/effect/statup"
	"errors"
	"io"
	"testing"

	channelhandler "atlas-channel/skill/handler"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/constants"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	skill2 "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
)

// Fixture ids: casterId is the cast originator; aliveA/aliveB are living
// non-caster recipients; deadC has Hp()==0; hiddenD is a hidden-GM recipient.
// None of these are wire skill ids -- they're arbitrary character ids chosen
// to be easy to eyeball in test failures.
const casterId, aliveA, aliveB, deadC, hiddenD = uint32(100), 101, 102, 103, 104

func tl() logrus.FieldLogger {
	l := logrus.New()
	l.SetOutput(io.Discard)
	return l
}

func mkRecipient(id uint32, hp uint16) channelhandler.PartyRecipient {
	return channelhandler.NewPartyRecipientBuilder().
		SetId(id).SetHp(hp).SetMaxHp(1000).SetMp(100).SetMaxMp(100).Build()
}

func testField() field.Model {
	return field.NewBuilder(0, 0, 1).Build()
}

// testInfo builds a SkillUsageInfo carrying an arbitrary (non-Echo-of-Hero)
// skill id -- applyEchoOfHero only logs it, never resolves or compares it, so
// no wire-id constant is needed here.
func testInfo() packetmodel.SkillUsageInfo {
	return packetmodel.NewSkillUsageInfoBuilder().
		SetSkillId(9999).
		SetSkillLevel(1).
		Build()
}

// testEffect builds an effect.Model with the given duration and a synthetic
// stat-up list of the given length, through the REST extract path (no
// builder exists on the model; Extract is the production construction seam),
// mirroring mprecovery_test.go's testEffect helper.
func testEffect(t *testing.T, duration int32, statUpCount int) effect.Model {
	t.Helper()
	su := make([]statup.RestModel, statUpCount)
	for i := range su {
		su[i] = statup.RestModel{Type: "echoOfHero", Amount: 100}
	}
	e, err := effect.Extract(effect.RestModel{Duration: duration, Statups: su})
	if err != nil {
		t.Fatalf("effect.Extract returned error: %v", err)
	}
	return e
}

// capture records the recipient ids applyBuff was invoked for.
type capture struct {
	applied []uint32
}

// newDeps returns an echoDeps wired to the given recipients, with isGmHidden
// defaulting to "never hidden, no error" and applyBuff recording every call
// into c. Individual tests override d.isGmHidden / d.applyBuff for the
// scenario under test, mirroring healdispel_test.go's newDeps + field
// override shape.
func newDeps(recips []channelhandler.PartyRecipient, c *capture) echoDeps {
	return echoDeps{
		selectInMap: func(field.Model) []channelhandler.PartyRecipient { return recips },
		isGmHidden:  func(uint32) (bool, error) { return false, nil },
		applyBuff: func(id uint32) error {
			c.applied = append(c.applied, id)
			return nil
		},
	}
}

func contains(ids []uint32, id uint32) bool {
	for _, v := range ids {
		if v == id {
			return true
		}
	}
	return false
}

func TestAppliesToAllLivingNonCaster(t *testing.T) {
	recips := []channelhandler.PartyRecipient{
		mkRecipient(casterId, 100),
		mkRecipient(aliveA, 100),
		mkRecipient(aliveB, 100),
	}
	var c capture
	d := newDeps(recips, &c)

	if err := applyEchoOfHero(tl(), testField(), casterId, testInfo(), testEffect(t, 5000, 1), d); err != nil {
		t.Fatalf("applyEchoOfHero returned unexpected error: %v", err)
	}
	if len(c.applied) != 2 || !contains(c.applied, aliveA) || !contains(c.applied, aliveB) {
		t.Fatalf("applied = %v, want exactly [aliveA, aliveB]", c.applied)
	}
}

func TestCasterSkippedNotDoubleBuffed(t *testing.T) {
	recips := []channelhandler.PartyRecipient{
		mkRecipient(casterId, 100),
		mkRecipient(aliveA, 100),
	}
	var c capture
	d := newDeps(recips, &c)

	if err := applyEchoOfHero(tl(), testField(), casterId, testInfo(), testEffect(t, 5000, 1), d); err != nil {
		t.Fatalf("applyEchoOfHero returned unexpected error: %v", err)
	}
	if contains(c.applied, casterId) {
		t.Fatalf("applied = %v, caster [%d] must never be double-buffed by the fan-out", c.applied, casterId)
	}
}

func TestDeadRecipientSkipped(t *testing.T) {
	recips := []channelhandler.PartyRecipient{
		mkRecipient(casterId, 100),
		mkRecipient(deadC, 0),
	}
	var c capture
	d := newDeps(recips, &c)

	if err := applyEchoOfHero(tl(), testField(), casterId, testInfo(), testEffect(t, 5000, 1), d); err != nil {
		t.Fatalf("applyEchoOfHero returned unexpected error: %v", err)
	}
	if contains(c.applied, deadC) {
		t.Fatalf("applied = %v, dead recipient [%d] must be skipped", c.applied, deadC)
	}
}

func TestHiddenGmSkipped(t *testing.T) {
	recips := []channelhandler.PartyRecipient{
		mkRecipient(casterId, 100),
		mkRecipient(hiddenD, 100),
	}
	var c capture
	d := newDeps(recips, &c)
	d.isGmHidden = func(id uint32) (bool, error) {
		return id == hiddenD, nil
	}

	if err := applyEchoOfHero(tl(), testField(), casterId, testInfo(), testEffect(t, 5000, 1), d); err != nil {
		t.Fatalf("applyEchoOfHero returned unexpected error: %v", err)
	}
	if contains(c.applied, hiddenD) {
		t.Fatalf("applied = %v, hidden GM recipient [%d] must be skipped", c.applied, hiddenD)
	}
}

// TestHiddenCheckErrorSkipsOnlyThatRecipient guards FR-2.5: an isGmHidden
// failure for one recipient must not abort the fan-out to the rest.
func TestHiddenCheckErrorSkipsOnlyThatRecipient(t *testing.T) {
	recips := []channelhandler.PartyRecipient{
		mkRecipient(casterId, 100),
		mkRecipient(aliveA, 100),
		mkRecipient(aliveB, 100),
	}
	var c capture
	d := newDeps(recips, &c)
	d.isGmHidden = func(id uint32) (bool, error) {
		if id == aliveA {
			return false, errors.New("simulated buff-lookup failure")
		}
		return false, nil
	}

	if err := applyEchoOfHero(tl(), testField(), casterId, testInfo(), testEffect(t, 5000, 1), d); err != nil {
		t.Fatalf("applyEchoOfHero returned unexpected error: %v", err)
	}
	if contains(c.applied, aliveA) {
		t.Fatalf("applied = %v, recipient [%d] with a hidden-check error must be skipped", c.applied, aliveA)
	}
	if !contains(c.applied, aliveB) {
		t.Fatalf("applied = %v, recipient [%d] must still be applied despite [%d]'s hidden-check error", c.applied, aliveB, aliveA)
	}
}

// TestApplyErrorDoesNotAbortRemaining guards D4: an applyBuff failure for one
// recipient must not abort the fan-out to the rest.
func TestApplyErrorDoesNotAbortRemaining(t *testing.T) {
	recips := []channelhandler.PartyRecipient{
		mkRecipient(casterId, 100),
		mkRecipient(aliveA, 100),
		mkRecipient(aliveB, 100),
	}
	var c capture
	d := newDeps(recips, &c)
	d.applyBuff = func(id uint32) error {
		if id == aliveA {
			return errors.New("simulated apply failure")
		}
		c.applied = append(c.applied, id)
		return nil
	}

	if err := applyEchoOfHero(tl(), testField(), casterId, testInfo(), testEffect(t, 5000, 1), d); err != nil {
		t.Fatalf("applyEchoOfHero returned unexpected error: %v", err)
	}
	if contains(c.applied, aliveA) {
		t.Fatalf("applied = %v, recipient [%d]'s failed apply must not be recorded", c.applied, aliveA)
	}
	if !contains(c.applied, aliveB) {
		t.Fatalf("applied = %v, recipient [%d] must still be applied despite [%d]'s apply failure", c.applied, aliveB, aliveA)
	}
}

func TestZeroDurationAppliesToNobody(t *testing.T) {
	recips := []channelhandler.PartyRecipient{
		mkRecipient(casterId, 100),
		mkRecipient(aliveA, 100),
	}
	var c capture
	d := newDeps(recips, &c)

	if err := applyEchoOfHero(tl(), testField(), casterId, testInfo(), testEffect(t, 0, 1), d); err != nil {
		t.Fatalf("applyEchoOfHero returned unexpected error: %v", err)
	}
	if len(c.applied) != 0 {
		t.Fatalf("applied = %v, want none for a zero-duration effect", c.applied)
	}
}

func TestNoStatUpsAppliesToNobody(t *testing.T) {
	recips := []channelhandler.PartyRecipient{
		mkRecipient(casterId, 100),
		mkRecipient(aliveA, 100),
	}
	var c capture
	d := newDeps(recips, &c)

	if err := applyEchoOfHero(tl(), testField(), casterId, testInfo(), testEffect(t, 5000, 0), d); err != nil {
		t.Fatalf("applyEchoOfHero returned unexpected error: %v", err)
	}
	if len(c.applied) != 0 {
		t.Fatalf("applied = %v, want none for an effect with no stat-ups", c.applied)
	}
}

func TestEmptyMapIsNoOp(t *testing.T) {
	var c capture
	d := newDeps(nil, &c)

	if err := applyEchoOfHero(tl(), testField(), casterId, testInfo(), testEffect(t, 5000, 1), d); err != nil {
		t.Fatalf("applyEchoOfHero returned unexpected error: %v", err)
	}
	if len(c.applied) != 0 {
		t.Fatalf("applied = %v, want none for an empty map", c.applied)
	}
}

// TestRegistration proves init() installs the handler under all four Echo of
// Hero identities (precedent: resurrection_test.go's
// TestResurrection_RegistersAllThreeIds, timeleap_test.go's
// TestTimeLeapRegistered).
func TestRegistration(t *testing.T) {
	for _, id := range []skill2.Identity{
		skill2.BeginnerEchoOfHero, skill2.NoblesseEchoOfHero,
		skill2.LegendEchoOfHero, skill2.EvanEchoOfHero,
	} {
		h, ok := channelhandler.Lookup(id)
		if !ok || h == nil {
			t.Fatalf("Lookup(%v) = (%v, %v), want non-nil handler", id, h, ok)
		}
	}
}

// TestVersionResolution_UnboundOnV12AndV48 proves wire 1005 resolves to no
// identity at all on gms_v12 and gms_v48 (design §7.2): neither version ships
// any Echo of Hero variant, so the handler -- though registered -- is
// unreachable there without any version-specific code in this package.
func TestVersionResolution_UnboundOnV12AndV48(t *testing.T) {
	if _, ok := constants.For("GMS", 12, 1).Skill.Resolve(skill2.Id(1005)); ok {
		t.Fatal("gms_v12 wire 1005 resolved to an identity, want unbound")
	}
	if _, ok := constants.For("GMS", 48, 1).Skill.Resolve(skill2.Id(1005)); ok {
		t.Fatal("gms_v48 wire 1005 resolved to an identity, want unbound")
	}
}

// TestVersionResolution_BeginnerOnlyOnV61 proves gms_v61 ships only the
// Beginner variant (design §7.2): wire 1005 resolves to BeginnerEchoOfHero,
// but Noblesse's wire 10001005 resolves to nothing.
func TestVersionResolution_BeginnerOnlyOnV61(t *testing.T) {
	id, ok := constants.For("GMS", 61, 1).Skill.Resolve(skill2.Id(1005))
	if !ok {
		t.Fatal("gms_v61 wire 1005 failed to resolve to any identity")
	}
	if id != skill2.BeginnerEchoOfHero {
		t.Fatalf("gms_v61 wire 1005 resolved to %v, want BeginnerEchoOfHero", id)
	}
	if _, ok := constants.For("GMS", 61, 1).Skill.Resolve(skill2.Id(10001005)); ok {
		t.Fatal("gms_v61 wire 10001005 resolved to an identity, want unbound (Noblesse not shipped until v72)")
	}
}

// TestVersionResolution_EvanUnboundBeforeV84 proves the Evan variant is
// unbound until gms_v84 (design §7.2): wire 20011005 resolves to nothing on
// gms_v79, but resolves to EvanEchoOfHero on gms_v84.
func TestVersionResolution_EvanUnboundBeforeV84(t *testing.T) {
	if _, ok := constants.For("GMS", 79, 1).Skill.Resolve(skill2.Id(20011005)); ok {
		t.Fatal("gms_v79 wire 20011005 resolved to an identity, want unbound (Evan not shipped until v84)")
	}
	id, ok := constants.For("GMS", 84, 1).Skill.Resolve(skill2.Id(20011005))
	if !ok {
		t.Fatal("gms_v84 wire 20011005 failed to resolve to any identity")
	}
	if id != skill2.EvanEchoOfHero {
		t.Fatalf("gms_v84 wire 20011005 resolved to %v, want EvanEchoOfHero", id)
	}
}
