package conversation

import (
	"context"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
)

func TestPickFromContextEmptyRoutesToEmptyNextState(t *testing.T) {
	mr := miniredis.RunT(t)
	rc := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	InitRegistry(rc)

	l, _ := test.NewNullLogger()
	l.SetLevel(logrus.DebugLevel)

	var tm tenant.Model
	tctx := tenant.WithContext(context.Background(), tm)

	pfc, err := NewPickFromContextBuilder().
		SetValuesContextKey("evolvablePets").
		SetNextState("confirm").
		SetEmptyNextState("noEligible").
		Build()
	if err != nil {
		t.Fatalf("build pickFromContext: %v", err)
	}
	pick, err := NewStateBuilder().SetId("pick").SetPickFromContext(pfc).Build()
	if err != nil {
		t.Fatalf("build state: %v", err)
	}
	container := testStateContainer{start: "pick", states: []StateModel{pick}}

	f := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(20000)).Build()
	ctx := NewConversationContextBuilder().
		SetField(f).
		SetCharacterId(7).
		SetNpcId(1032102).
		SetCurrentState("pick").
		SetConversation(container).
		AddContextValue("evolvablePets", ""). // empty list → must route to emptyNextState
		Build()
	GetRegistry().SetContext(tctx, ctx.CharacterId(), ctx)

	p := &ProcessorImpl{l: l, ctx: tctx, t: tm}
	if _, err := p.ProcessState(ctx); err != nil {
		t.Fatalf("ProcessState: %v", err)
	}

	got, err := GetRegistry().GetPreviousContext(tctx, ctx.CharacterId())
	if err != nil {
		t.Fatalf("GetPreviousContext: %v", err)
	}
	if got.CurrentState() != "noEligible" {
		t.Errorf("CurrentState = %q, want %q (empty values must route to emptyNextState)", got.CurrentState(), "noEligible")
	}
}
