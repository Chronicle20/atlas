package monster

import (
	charactermock "atlas-monster-death/character/mock"
	mapmock "atlas-monster-death/map/mock"
	informationmock "atlas-monster-death/monster/information/mock"
	partymock "atlas-monster-death/party/mock"
	ratesmock "atlas-monster-death/rates/mock"
	"atlas-monster-death/system_message"
	systemmessagemock "atlas-monster-death/system_message/mock"
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func newDiTestContext() context.Context {
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	return tenant.WithContext(context.Background(), ten)
}

func TestWith_ReturnsCloneAndDoesNotMutateOriginal(t *testing.T) {
	l := logrus.New()
	ctx := newDiTestContext()

	base := NewProcessor(l, ctx).(*ProcessorImpl)
	originalRp := base.rp

	clone := base.With(WithRatesProcessor(&ratesmock.ProcessorMock{})).(*ProcessorImpl)

	if clone == base {
		t.Fatalf("expected With to return a distinct clone")
	}
	if clone.rp == base.rp {
		t.Fatalf("expected clone.rp to differ from base.rp")
	}
	if base.rp != originalRp {
		t.Fatalf("expected base.rp to be unchanged by With")
	}
}

func TestWith_AppliesEveryOption(t *testing.T) {
	l := logrus.New()
	ctx := newDiTestContext()

	cp := &charactermock.ProcessorMock{}
	pp := &partymock.ProcessorMock{}
	rp := &ratesmock.ProcessorMock{}
	ip := &informationmock.ProcessorMock{}
	fp := &mapmock.ProcessorMock{}
	smp := &systemmessagemock.ProcessorMock{}
	ht := system_message.GetHintThrottle()
	cfg := ExperienceConfig{LevelInterval: 42}

	base := NewProcessor(l, ctx).(*ProcessorImpl)
	clone := base.With(
		WithCharacterProcessor(cp),
		WithPartyProcessor(pp),
		WithRatesProcessor(rp),
		WithInformationProcessor(ip),
		WithFieldProcessor(fp),
		WithSystemMessageProcessor(smp),
		WithHintThrottle(ht),
		WithExperienceConfig(cfg),
	).(*ProcessorImpl)

	if clone.cp != cp {
		t.Errorf("expected clone.cp to be the injected mock")
	}
	if clone.pp != pp {
		t.Errorf("expected clone.pp to be the injected mock")
	}
	if clone.rp != rp {
		t.Errorf("expected clone.rp to be the injected mock")
	}
	if clone.ip != ip {
		t.Errorf("expected clone.ip to be the injected mock")
	}
	if clone.fp != fp {
		t.Errorf("expected clone.fp to be the injected mock")
	}
	if clone.smp != smp {
		t.Errorf("expected clone.smp to be the injected mock")
	}
	if clone.ht != ht {
		t.Errorf("expected clone.ht to be the injected throttle")
	}
	if clone.cfg.LevelInterval != 42 {
		t.Errorf("expected clone.cfg.LevelInterval to be 42, got %d", clone.cfg.LevelInterval)
	}
}

func TestNewProcessor_BindsProductionDefaults(t *testing.T) {
	l := logrus.New()
	ctx := newDiTestContext()

	p := NewProcessor(l, ctx).(*ProcessorImpl)

	if p.cp == nil {
		t.Errorf("expected cp to be non-nil")
	}
	if p.pp == nil {
		t.Errorf("expected pp to be non-nil")
	}
	if p.rp == nil {
		t.Errorf("expected rp to be non-nil")
	}
	if p.ip == nil {
		t.Errorf("expected ip to be non-nil")
	}
	if p.fp == nil {
		t.Errorf("expected fp to be non-nil")
	}
	if p.smp == nil {
		t.Errorf("expected smp to be non-nil")
	}
	if p.ht == nil {
		t.Errorf("expected ht to be non-nil")
	}
	if p.cfg != LoadExperienceConfig() {
		t.Errorf("expected cfg to equal LoadExperienceConfig(), got %+v", p.cfg)
	}
}
