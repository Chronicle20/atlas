package playernpc

import (
	"atlas-messages/character"
	msg "atlas-messages/kafka/message/playernpc"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus/hooks/test"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

func testCharacter(isGm bool) character.Model {
	gm := 0
	if isGm {
		gm = 1
	}
	return character.NewBuilder().SetId(1).SetGm(gm).SetX(100).SetY(200).Build()
}

// TestPlayerNpcCommands is the plan.md Task 21 test table.
func TestPlayerNpcCommands(t *testing.T) {
	logger, _ := test.NewNullLogger()
	ctx := context.Background()
	f := field.NewBuilder(1, 2, 102000004).Build()

	t.Run("deploy matches", func(t *testing.T) {
		executor, found := DeployCommandProducer(logger)(ctx)(f, testCharacter(true), "@playernpc add Hero")
		if !found || executor == nil {
			t.Fatalf("found = %v, executor = %v, want true, non-nil", found, executor)
		}
	})

	t.Run("deploy, non-GM", func(t *testing.T) {
		executor, found := DeployCommandProducer(logger)(ctx)(f, testCharacter(false), "@playernpc add Hero")
		if found || executor != nil {
			t.Fatalf("found = %v, executor = %v, want false, nil", found, executor)
		}
	})

	t.Run("deploy emits", func(t *testing.T) {
		c := testCharacter(true)
		target := character.NewBuilder().SetId(42).SetName("Hero").Build()

		var published msg.Command[msg.CommandDeployBody]
		var publishCalled bool
		var pinkTexts []string

		lookup := func(name string) (character.Model, error) {
			if name != "Hero" {
				t.Fatalf("lookup name = %q, want Hero", name)
			}
			return target, nil
		}
		publish := func(p model.Provider[[]kafka.Message]) error {
			publishCalled = true
			ms, err := p()
			if err != nil {
				t.Fatalf("provider() err = %v", err)
			}
			if len(ms) != 1 {
				t.Fatalf("message count = %d, want 1", len(ms))
			}
			if err := json.Unmarshal(ms[0].Value, &published); err != nil {
				t.Fatalf("unmarshal published value: %v", err)
			}
			return nil
		}
		pink := func(text string) error {
			pinkTexts = append(pinkTexts, text)
			return nil
		}

		if err := deployWithDeps(f, c, "Hero", lookup, publish, pink); err != nil {
			t.Fatalf("deployWithDeps err = %v", err)
		}
		if !publishCalled {
			t.Fatal("publish was not called")
		}
		if published.CharacterId != 42 || published.Type != msg.CommandTypeDeploy {
			t.Errorf("published = %+v", published)
		}
		if published.Body.MapId != f.MapId() || published.Body.WorldId != f.WorldId() {
			t.Errorf("published body map/world = %+v, want map %v world %v", published.Body, f.MapId(), f.WorldId())
		}
		if published.Body.Position == nil || published.Body.Position.X != c.X() || published.Body.Position.Y != c.Y() {
			t.Errorf("published position = %+v, want (%d, %d)", published.Body.Position, c.X(), c.Y())
		}
		if published.Body.EnforceEligibility {
			t.Error("EnforceEligibility = true, want false")
		}
		if published.Requester == nil {
			t.Fatal("published.Requester = nil, want populated from f and c")
		}
		if published.Requester.CharacterId != c.Id() || published.Requester.WorldId != byte(f.WorldId()) || published.Requester.ChannelId != byte(f.ChannelId()) || published.Requester.MapId != uint32(f.MapId()) {
			t.Errorf("published.Requester = %+v, want characterId %d, worldId %d, channelId %d, mapId %d", published.Requester, c.Id(), byte(f.WorldId()), byte(f.ChannelId()), uint32(f.MapId()))
		}
		if len(pinkTexts) != 1 {
			t.Fatalf("pink text calls = %d, want 1", len(pinkTexts))
		}
	})

	t.Run("remove all", func(t *testing.T) {
		c := testCharacter(true)
		target := character.NewBuilder().SetId(42).SetName("Hero").Build()

		var published msg.Command[msg.CommandRemoveBody]
		lookup := func(name string) (character.Model, error) { return target, nil }
		publish := func(p model.Provider[[]kafka.Message]) error {
			ms, err := p()
			if err != nil {
				t.Fatalf("provider() err = %v", err)
			}
			return json.Unmarshal(ms[0].Value, &published)
		}
		pink := func(text string) error { return nil }

		if err := removeWithDeps(f, c, "Hero", nil, lookup, publish, pink); err != nil {
			t.Fatalf("removeWithDeps err = %v", err)
		}
		if published.CharacterId != 42 || published.Type != msg.CommandTypeRemove {
			t.Errorf("published = %+v", published)
		}
		if published.Body.MapId != nil {
			t.Errorf("published body mapId = %v, want nil", published.Body.MapId)
		}
		if published.Requester == nil || published.Requester.CharacterId != c.Id() {
			t.Errorf("published.Requester = %+v, want characterId %d", published.Requester, c.Id())
		}
	})

	t.Run("remove map-scoped", func(t *testing.T) {
		c := testCharacter(true)
		target := character.NewBuilder().SetId(42).SetName("Hero").Build()
		mapId := f.MapId()

		var published msg.Command[msg.CommandRemoveBody]
		lookup := func(name string) (character.Model, error) { return target, nil }
		publish := func(p model.Provider[[]kafka.Message]) error {
			ms, err := p()
			if err != nil {
				t.Fatalf("provider() err = %v", err)
			}
			return json.Unmarshal(ms[0].Value, &published)
		}
		pink := func(text string) error { return nil }

		if err := removeWithDeps(f, c, "Hero", &mapId, lookup, publish, pink); err != nil {
			t.Fatalf("removeWithDeps err = %v", err)
		}
		if published.Body.MapId == nil || *published.Body.MapId != mapId {
			t.Errorf("published body mapId = %v, want %v", published.Body.MapId, mapId)
		}
		if published.Requester == nil || published.Requester.WorldId != byte(f.WorldId()) || published.Requester.ChannelId != byte(f.ChannelId()) || published.Requester.MapId != uint32(f.MapId()) {
			t.Errorf("published.Requester = %+v, want worldId %d, channelId %d, mapId %d", published.Requester, byte(f.WorldId()), byte(f.ChannelId()), uint32(f.MapId()))
		}
	})

	t.Run("unknown character", func(t *testing.T) {
		publishCalled := false
		var pinkTexts []string

		lookup := func(name string) (character.Model, error) { return character.Model{}, errors.New("not found") }
		publish := func(p model.Provider[[]kafka.Message]) error { publishCalled = true; return nil }
		pink := func(text string) error { pinkTexts = append(pinkTexts, text); return nil }

		c := testCharacter(true)
		if err := deployWithDeps(f, c, "Nobody", lookup, publish, pink); err != nil {
			t.Fatalf("deployWithDeps err = %v", err)
		}
		if publishCalled {
			t.Error("publish was called, want no command emitted")
		}
		if len(pinkTexts) != 1 {
			t.Fatalf("pink text calls = %d, want 1", len(pinkTexts))
		}
		if got := pinkTexts[0]; got != "Character not found: Nobody." {
			t.Errorf("pink text = %q, want to name Nobody as not found", got)
		}
	})

	t.Run("non-matching text", func(t *testing.T) {
		executor, found := DeployCommandProducer(logger)(ctx)(f, testCharacter(true), "@playernpcs add Hero")
		if found || executor != nil {
			t.Fatalf("found = %v, executor = %v, want false, nil", found, executor)
		}
	})

	t.Run("failure reported back", func(t *testing.T) {
		target := character.NewBuilder().SetId(42).SetName("Hero").Build()
		var pinkTexts []string

		lookup := func(name string) (character.Model, error) { return target, nil }
		publish := func(p model.Provider[[]kafka.Message]) error { return errors.New("pool_exhausted") }
		pink := func(text string) error { pinkTexts = append(pinkTexts, text); return nil }

		c := testCharacter(true)
		if err := deployWithDeps(f, c, "Hero", lookup, publish, pink); err != nil {
			t.Fatalf("deployWithDeps err = %v", err)
		}
		if len(pinkTexts) != 1 {
			t.Fatalf("pink text calls = %d, want 1", len(pinkTexts))
		}
		if got := pinkTexts[0]; !strings.Contains(got, "pool_exhausted") {
			t.Errorf("pink text = %q, want it to name pool_exhausted", got)
		}
	})
}

func TestRemoveCommandProducer_Gate(t *testing.T) {
	logger, _ := test.NewNullLogger()
	ctx := context.Background()
	f := field.NewBuilder(1, 2, 102000004).Build()

	testCases := []struct {
		name        string
		isGm        bool
		message     string
		expectFound bool
	}{
		{"remove matches", true, "@playernpc remove Hero", true},
		{"remove here matches", true, "@playernpc remove Hero here", true},
		{"non-GM does not match", false, "@playernpc remove Hero", false},
		{"non-matching text", true, "@playernpcs remove Hero", false},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			executor, found := RemoveCommandProducer(logger)(ctx)(f, testCharacter(tc.isGm), tc.message)
			if found != tc.expectFound {
				t.Fatalf("found = %v, want %v", found, tc.expectFound)
			}
			if tc.expectFound && executor == nil {
				t.Error("expected non-nil executor when found")
			}
			if !tc.expectFound && executor != nil {
				t.Error("expected nil executor when not found")
			}
		})
	}
}
