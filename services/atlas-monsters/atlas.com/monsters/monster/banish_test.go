package monster

import (
	"atlas-monsters/kafka/message/system_message"
	"atlas-monsters/monster/information"
	mobskill "atlas-monsters/monster/mobskill"
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	monster2 "github.com/Chronicle20/atlas/libs/atlas-constants/monster"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

const (
	banishTemplateId = uint32(9500324)
	banishMapId      = uint32(926120410)
	banishPortal     = "st00"
	banishMessage    = "You do not belong here."
	banishCharacter  = uint32(4242)
)

// TestBanish is the fail-closed table: none of these rows may emit anything.
func TestBanish(t *testing.T) {
	tests := []struct {
		name  string
		seed  func(r *Registry, ten tenant.Model, f field.Model, ctx context.Context)
		lookp func(monsterId uint32) (information.Model, error)
	}{
		{
			name: "no live monster in field",
			seed: func(r *Registry, ten tenant.Model, f field.Model, ctx context.Context) {},
			lookp: func(_ uint32) (information.Model, error) {
				t.Fatalf("information lookup must not be called")
				return information.Model{}, nil
			},
		},
		{
			name: "wrong template alive",
			seed: func(r *Registry, ten tenant.Model, f field.Model, ctx context.Context) {
				r.CreateMonster(ctx, ten, f, 1000000, 0, 0, 0, 5, 0, 5000, 100, "", "")
			},
			lookp: func(_ uint32) (information.Model, error) {
				t.Fatalf("information lookup must not be called")
				return information.Model{}, nil
			},
		},
		{
			name: "information fetch fails",
			seed: func(r *Registry, ten tenant.Model, f field.Model, ctx context.Context) {
				r.CreateMonster(ctx, ten, f, banishTemplateId, 0, 0, 0, 5, 0, 5000, 100, "", "")
			},
			lookp: func(_ uint32) (information.Model, error) {
				return information.Model{}, errors.New("boom")
			},
		},
		{
			name: "zero banish map",
			seed: func(r *Registry, ten tenant.Model, f field.Model, ctx context.Context) {
				r.CreateMonster(ctx, ten, f, banishTemplateId, 0, 0, 0, 5, 0, 5000, 100, "", "")
			},
			lookp: func(_ uint32) (information.Model, error) {
				return information.NewModelBuilder().SetBanish(information.Banish{MapId: 0}).Build(), nil
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := GetMonsterRegistry()
			ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
			ctx := context.Background()
			r.Clear(ctx)

			prevHook := testInformationLookup
			testInformationLookup = tc.lookp
			defer func() { testInformationLookup = prevHook }()

			f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(40000)).Build()
			tc.seed(r, ten, f, ctx)

			p, events := newRecordingProcessorWithBodies(t, ten)
			err := p.Banish(f, banishCharacter, banishTemplateId)

			if err == nil {
				t.Fatalf("expected non-nil error")
			}
			if len(*events) != 0 {
				t.Fatalf("expected 0 emitted events, got %d: %v", len(*events), *events)
			}
		})
	}
}

// TestBanish_PortalNamePresent — the happy path with a WZ portal name: exactly
// one WARP carrying the target map and portal name.
func TestBanish_PortalNamePresent(t *testing.T) {
	r := GetMonsterRegistry()
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := context.Background()
	r.Clear(ctx)

	prevHook := testInformationLookup
	testInformationLookup = func(_ uint32) (information.Model, error) {
		return information.NewModelBuilder().SetBanish(information.Banish{MapId: banishMapId, PortalName: banishPortal}).Build(), nil
	}
	defer func() { testInformationLookup = prevHook }()

	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(40000)).Build()
	r.CreateMonster(ctx, ten, f, banishTemplateId, 0, 0, 0, 5, 0, 5000, 100, "", "")

	p, events := newRecordingProcessorWithBodies(t, ten)
	err := p.Banish(f, banishCharacter, banishTemplateId)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(*events) != 1 {
		t.Fatalf("expected 1 event, got %d: %v", len(*events), *events)
	}
	if (*events)[0].Topic != EnvCommandTopicPortal {
		t.Errorf("event[0].Topic = %q, want %q", (*events)[0].Topic, EnvCommandTopicPortal)
	}
	if (*events)[0].Type != "WARP" {
		t.Errorf("event[0].Type = %q, want WARP", (*events)[0].Type)
	}
	var body warpBody
	if err := json.Unmarshal((*events)[0].Body, &body); err != nil {
		t.Fatalf("decode WARP body: %v", err)
	}
	if body.CharacterId != banishCharacter {
		t.Errorf("CharacterId = %d, want %d", body.CharacterId, banishCharacter)
	}
	if body.TargetMapId != _map.Id(banishMapId) {
		t.Errorf("TargetMapId = %d, want %d", body.TargetMapId, banishMapId)
	}
	if body.TargetPortalName != banishPortal {
		t.Errorf("TargetPortalName = %q, want %q", body.TargetPortalName, banishPortal)
	}
}

// TestBanish_PortalNameAbsent — omitempty keeps the key entirely off the wire
// when there is no WZ portal name.
func TestBanish_PortalNameAbsent(t *testing.T) {
	r := GetMonsterRegistry()
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := context.Background()
	r.Clear(ctx)

	prevHook := testInformationLookup
	testInformationLookup = func(_ uint32) (information.Model, error) {
		return information.NewModelBuilder().SetBanish(information.Banish{MapId: banishMapId, PortalName: ""}).Build(), nil
	}
	defer func() { testInformationLookup = prevHook }()

	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(40000)).Build()
	r.CreateMonster(ctx, ten, f, banishTemplateId, 0, 0, 0, 5, 0, 5000, 100, "", "")

	p, events := newRecordingProcessorWithBodies(t, ten)
	err := p.Banish(f, banishCharacter, banishTemplateId)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(*events) != 1 {
		t.Fatalf("expected 1 event, got %d: %v", len(*events), *events)
	}
	if (*events)[0].Type != "WARP" {
		t.Errorf("event[0].Type = %q, want WARP", (*events)[0].Type)
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal((*events)[0].Body, &raw); err != nil {
		t.Fatalf("decode WARP body: %v", err)
	}
	if _, ok := raw["targetPortalName"]; ok {
		t.Errorf("expected no targetPortalName key, got %v", raw)
	}
}

// TestBanish_MessagePresent — warp then message, in that order, is the point.
func TestBanish_MessagePresent(t *testing.T) {
	r := GetMonsterRegistry()
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := context.Background()
	r.Clear(ctx)

	prevHook := testInformationLookup
	testInformationLookup = func(_ uint32) (information.Model, error) {
		return information.NewModelBuilder().SetBanish(information.Banish{MapId: banishMapId, PortalName: banishPortal, Message: banishMessage}).Build(), nil
	}
	defer func() { testInformationLookup = prevHook }()

	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(40000)).Build()
	r.CreateMonster(ctx, ten, f, banishTemplateId, 0, 0, 0, 5, 0, 5000, 100, "", "")

	p, events := newRecordingProcessorWithBodies(t, ten)
	err := p.Banish(f, banishCharacter, banishTemplateId)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(*events) != 2 {
		t.Fatalf("expected 2 events, got %d: %v", len(*events), *events)
	}
	if (*events)[0].Topic != EnvCommandTopicPortal || (*events)[0].Type != "WARP" {
		t.Errorf("event[0] = %+v, want WARP on %s", (*events)[0], EnvCommandTopicPortal)
	}
	if (*events)[1].Topic != system_message.EnvCommandTopic || (*events)[1].Type != system_message.CommandSendMessage {
		t.Errorf("event[1] = %+v, want %s on %s", (*events)[1], system_message.CommandSendMessage, system_message.EnvCommandTopic)
	}
	var msgBody system_message.SendMessageBody
	if err := json.Unmarshal((*events)[1].Body, &msgBody); err != nil {
		t.Fatalf("decode SEND_MESSAGE body: %v", err)
	}
	if msgBody.MessageType != "PINK_TEXT" {
		t.Errorf("MessageType = %q, want PINK_TEXT", msgBody.MessageType)
	}
	if msgBody.Message != banishMessage {
		t.Errorf("Message = %q, want %q", msgBody.Message, banishMessage)
	}
}

// TestBanish_MessageAbsent — no WZ banish message means no SEND_MESSAGE.
func TestBanish_MessageAbsent(t *testing.T) {
	r := GetMonsterRegistry()
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := context.Background()
	r.Clear(ctx)

	prevHook := testInformationLookup
	testInformationLookup = func(_ uint32) (information.Model, error) {
		return information.NewModelBuilder().SetBanish(information.Banish{MapId: banishMapId, PortalName: banishPortal, Message: ""}).Build(), nil
	}
	defer func() { testInformationLookup = prevHook }()

	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(40000)).Build()
	r.CreateMonster(ctx, ten, f, banishTemplateId, 0, 0, 0, 5, 0, 5000, 100, "", "")

	p, events := newRecordingProcessorWithBodies(t, ten)
	err := p.Banish(f, banishCharacter, banishTemplateId)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(*events) != 1 {
		t.Fatalf("expected 1 event, got %d: %v", len(*events), *events)
	}
	for _, e := range *events {
		if e.Type == system_message.CommandSendMessage {
			t.Errorf("unexpected SEND_MESSAGE event: %+v", e)
		}
	}
}

// TestExecuteBanish_ConvergesOnSharedExecutor — the skill-129 path must reach
// the same banishCharacter as the client-initiated path.
func TestExecuteBanish_ConvergesOnSharedExecutor(t *testing.T) {
	r := GetMonsterRegistry()
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := context.Background()
	r.Clear(ctx)

	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(40000)).Build()
	m := r.CreateMonster(ctx, ten, f, banishTemplateId, 0, 0, 0, 5, 0, 5000, 100, "", "")
	uniqueId := m.UniqueId()
	if _, err := r.ControlMonster(ten, uniqueId, banishCharacter); err != nil {
		t.Fatalf("ControlMonster: %v", err)
	}
	m, err := r.GetMonster(ten, uniqueId)
	if err != nil {
		t.Fatalf("GetMonster: %v", err)
	}

	prevHook := testInformationLookup
	testInformationLookup = func(_ uint32) (information.Model, error) {
		return information.NewModelBuilder().SetBanish(information.Banish{MapId: banishMapId, PortalName: banishPortal, Message: banishMessage}).Build(), nil
	}
	defer func() { testInformationLookup = prevHook }()

	sd := mobskill.NewModelBuilder().
		SetSkillId(uint16(monster2.SkillTypeBanish)).
		SetLevel(1).
		Build()

	p, events := newRecordingProcessorWithBodies(t, ten)
	p.executeBanish(m, sd)

	if len(*events) != 2 {
		t.Fatalf("expected 2 events, got %d: %v", len(*events), *events)
	}
	if (*events)[0].Topic != EnvCommandTopicPortal || (*events)[0].Type != "WARP" {
		t.Errorf("event[0] = %+v, want WARP on %s", (*events)[0], EnvCommandTopicPortal)
	}
	var warp warpBody
	if err := json.Unmarshal((*events)[0].Body, &warp); err != nil {
		t.Fatalf("decode WARP body: %v", err)
	}
	if warp.CharacterId != banishCharacter {
		t.Errorf("CharacterId = %d, want %d", warp.CharacterId, banishCharacter)
	}
	if warp.TargetMapId != _map.Id(banishMapId) {
		t.Errorf("TargetMapId = %d, want %d", warp.TargetMapId, banishMapId)
	}
	if warp.TargetPortalName != banishPortal {
		t.Errorf("TargetPortalName = %q, want %q", warp.TargetPortalName, banishPortal)
	}

	if (*events)[1].Topic != system_message.EnvCommandTopic || (*events)[1].Type != system_message.CommandSendMessage {
		t.Errorf("event[1] = %+v, want %s on %s", (*events)[1], system_message.CommandSendMessage, system_message.EnvCommandTopic)
	}
	var msgBody system_message.SendMessageBody
	if err := json.Unmarshal((*events)[1].Body, &msgBody); err != nil {
		t.Fatalf("decode SEND_MESSAGE body: %v", err)
	}
	if msgBody.MessageType != "PINK_TEXT" {
		t.Errorf("MessageType = %q, want PINK_TEXT", msgBody.MessageType)
	}
	if msgBody.Message != banishMessage {
		t.Errorf("Message = %q, want %q", msgBody.Message, banishMessage)
	}
}
