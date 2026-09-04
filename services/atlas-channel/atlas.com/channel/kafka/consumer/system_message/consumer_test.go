package system_message

import (
	system_message2 "atlas-channel/kafka/message/system_message"
	"atlas-channel/server"
	"atlas-channel/socket/writer"
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	testlog "github.com/sirupsen/logrus/hooks/test"

	channelconst "github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	opcodes "github.com/Chronicle20/atlas/libs/atlas-opcodes"
	fieldcb "github.com/Chronicle20/atlas/libs/atlas-packet/field/clientbound"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// task-290 C9d: gms_12_1 binds no FieldEffect/ContiMove writer at all -- no
// v12 IDA export exists to derive the opcode/mode table -- yet the
// change_music/boat_effect map-action seeds are replicated uniformly to
// every tenant (tools/catalog-lint enforces byte-identical map-action seeds
// with no exemption). These tests pin the runtime guard that absorbs the
// version gap: a tenant with no bound writer must skip the write and log,
// never fall through to ResolveCode's 99-sentinel crash path; a tenant WITH
// the writer bound must still write exactly as before.

func newTestTenant(t *testing.T) tenant.Model {
	t.Helper()
	tm, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant: %v", err)
	}
	return tm
}

func newTestServer(t *testing.T, tm tenant.Model) server.Model {
	t.Helper()
	ch := channelconst.NewModel(0, 1)
	return server.NewProcessor(logrus.New(), context.Background()).Register(tm, ch, "127.0.0.1", 8484)
}

// registerFieldEffectWriter mirrors template_gms_48_1.json's FieldEffect
// writer options table (the shape RegisterTenantWriterOptions/
// TenantWriterOptions produce), scoped to only the BACKGROUND_MUSIC key this
// package needs.
func registerFieldEffectWriter(t *testing.T, tenantId uuid.UUID) {
	t.Helper()
	writer.RegisterTenantWriterOptions(tenantId, []opcodes.WriterConfig{
		{OpCode: "0x54", Writer: fieldcb.FieldEffectWriter, Options: map[string]interface{}{
			"operations": map[string]interface{}{"BACKGROUND_MUSIC": float64(6)},
		}},
	})
	t.Cleanup(func() { writer.EvictTenantWriterOptions(tenantId) })
}

// registerContiMoveWriter mirrors template_gms_48_1.json's ContiMove writer
// options table.
func registerContiMoveWriter(t *testing.T, tenantId uuid.UUID) {
	t.Helper()
	writer.RegisterTenantWriterOptions(tenantId, []opcodes.WriterConfig{
		{OpCode: "0x5B", Writer: fieldcb.ContiMoveWriter, Options: map[string]interface{}{
			"operations": map[string]interface{}{
				"SHOW_STATE":     float64(10),
				"SHOW_SUB_STATE": float64(4),
				"HIDE_STATE":     float64(10),
				"HIDE_SUB_STATE": float64(5),
			},
		}},
	})
	t.Cleanup(func() { writer.EvictTenantWriterOptions(tenantId) })
}

func TestChangeMusicConfigured(t *testing.T) {
	l, _ := testlog.NewNullLogger()

	t.Run("writer bound with BACKGROUND_MUSIC operation", func(t *testing.T) {
		tm := newTestTenant(t)
		ctx := tenant.WithContext(context.Background(), tm)
		registerFieldEffectWriter(t, tm.Id())

		if !changeMusicConfigured(l, ctx) {
			t.Fatal("changeMusicConfigured() = false, want true when the tenant binds FieldEffect/BACKGROUND_MUSIC")
		}
	})

	t.Run("no writer registered for tenant (gms_12_1 shape)", func(t *testing.T) {
		tm := newTestTenant(t)
		ctx := tenant.WithContext(context.Background(), tm)
		// Deliberately do not call writer.RegisterTenantWriterOptions.

		if changeMusicConfigured(l, ctx) {
			t.Fatal("changeMusicConfigured() = true, want false when the tenant binds no FieldEffect writer")
		}
	})

	t.Run("writer bound but BACKGROUND_MUSIC operation missing", func(t *testing.T) {
		tm := newTestTenant(t)
		ctx := tenant.WithContext(context.Background(), tm)
		writer.RegisterTenantWriterOptions(tm.Id(), []opcodes.WriterConfig{
			{OpCode: "0x54", Writer: fieldcb.FieldEffectWriter, Options: map[string]interface{}{
				"operations": map[string]interface{}{"SOUND": float64(4)},
			}},
		})
		t.Cleanup(func() { writer.EvictTenantWriterOptions(tm.Id()) })

		if changeMusicConfigured(l, ctx) {
			t.Fatal("changeMusicConfigured() = true, want false when BACKGROUND_MUSIC is not a bound operation")
		}
	})

	t.Run("logs a visible warning naming the tenant and missing writer", func(t *testing.T) {
		tm := newTestTenant(t)
		ctx := tenant.WithContext(context.Background(), tm)
		hookLogger, hook := testlog.NewNullLogger()

		changeMusicConfigured(hookLogger, ctx)

		entries := hook.AllEntries()
		if len(entries) != 1 {
			t.Fatalf("log entries = %d, want 1", len(entries))
		}
		if entries[0].Level != logrus.WarnLevel {
			t.Fatalf("log level = %s, want warning (must be visible in normal operation)", entries[0].Level)
		}
		if got := entries[0].Message; !strings.Contains(got, tm.String()) || !strings.Contains(got, fieldcb.FieldEffectWriter) {
			t.Fatalf("log message = %q, want it to name the tenant and the missing [%s] writer", got, fieldcb.FieldEffectWriter)
		}
	})
}

func TestBoatEffectConfigured(t *testing.T) {
	l, _ := testlog.NewNullLogger()

	t.Run("writer bound with SHOW_STATE/HIDE_STATE operations", func(t *testing.T) {
		tm := newTestTenant(t)
		ctx := tenant.WithContext(context.Background(), tm)
		registerContiMoveWriter(t, tm.Id())

		if !boatEffectConfigured(l, ctx, writer.ContiMoveShow) {
			t.Fatal("boatEffectConfigured(SHOW) = false, want true when the tenant binds ContiMove/SHOW_STATE")
		}
		if !boatEffectConfigured(l, ctx, writer.ContiMoveHide) {
			t.Fatal("boatEffectConfigured(HIDE) = false, want true when the tenant binds ContiMove/HIDE_STATE")
		}
	})

	t.Run("no writer registered for tenant (gms_12_1 shape)", func(t *testing.T) {
		tm := newTestTenant(t)
		ctx := tenant.WithContext(context.Background(), tm)
		// Deliberately do not call writer.RegisterTenantWriterOptions.

		if boatEffectConfigured(l, ctx, writer.ContiMoveShow) {
			t.Fatal("boatEffectConfigured() = true, want false when the tenant binds no ContiMove writer")
		}
	})
}

// TestHandleChangeMusic_Unconfigured_SkipsWrite pins the direction that
// killed task C9c's delete-the-seeds approach: an unbound tenant must not
// write a packet at all.
func TestHandleChangeMusic_Unconfigured_SkipsWrite(t *testing.T) {
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	sc := newTestServer(t, tm)
	// Deliberately do not register a FieldEffect writer for this tenant.

	var calls int
	orig := changeMusicAnnouncer
	changeMusicAnnouncer = func(_ logrus.FieldLogger, _ context.Context, _ server.Model, _ writer.Producer, _ uint32, _ string) error {
		calls++
		return nil
	}
	defer func() { changeMusicAnnouncer = orig }()

	h := handleChangeMusic(sc, nil)
	h(logrus.New(), ctx, system_message2.Command[system_message2.ChangeMusicBody]{
		WorldId:     sc.WorldId(),
		ChannelId:   sc.ChannelId(),
		CharacterId: 100,
		Type:        system_message2.CommandChangeMusic,
		Body:        system_message2.ChangeMusicBody{Path: "Bgm04/ArabPirate"},
	})

	if calls != 0 {
		t.Fatalf("changeMusicAnnouncer calls = %d, want 0 for a tenant with no bound FieldEffect writer", calls)
	}
}

// TestHandleChangeMusic_Configured_WritesExactlyAsBefore pins that the guard
// does not regress the ten tenants that DO bind FieldEffect.
func TestHandleChangeMusic_Configured_WritesExactlyAsBefore(t *testing.T) {
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	sc := newTestServer(t, tm)
	registerFieldEffectWriter(t, tm.Id())

	type call struct {
		characterId uint32
		path        string
	}
	var calls []call
	orig := changeMusicAnnouncer
	changeMusicAnnouncer = func(_ logrus.FieldLogger, _ context.Context, _ server.Model, _ writer.Producer, characterId uint32, path string) error {
		calls = append(calls, call{characterId: characterId, path: path})
		return nil
	}
	defer func() { changeMusicAnnouncer = orig }()

	h := handleChangeMusic(sc, nil)
	h(logrus.New(), ctx, system_message2.Command[system_message2.ChangeMusicBody]{
		WorldId:     sc.WorldId(),
		ChannelId:   sc.ChannelId(),
		CharacterId: 100,
		Type:        system_message2.CommandChangeMusic,
		Body:        system_message2.ChangeMusicBody{Path: "Bgm04/ArabPirate"},
	})

	if len(calls) != 1 {
		t.Fatalf("changeMusicAnnouncer calls = %d, want 1 for a tenant that binds FieldEffect", len(calls))
	}
	if calls[0].characterId != 100 || calls[0].path != "Bgm04/ArabPirate" {
		t.Fatalf("changeMusicAnnouncer call = %+v, want {characterId:100 path:Bgm04/ArabPirate}", calls[0])
	}
}

// TestHandleBoatEffect_Unconfigured_SkipsWrite mirrors the change_music case
// for the ContiMove writer.
func TestHandleBoatEffect_Unconfigured_SkipsWrite(t *testing.T) {
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	sc := newTestServer(t, tm)
	// Deliberately do not register a ContiMove writer for this tenant.

	var calls int
	orig := boatEffectAnnouncer
	boatEffectAnnouncer = func(_ logrus.FieldLogger, _ context.Context, _ server.Model, _ writer.Producer, _ uint32, _ writer.ContiMoveKey) error {
		calls++
		return nil
	}
	defer func() { boatEffectAnnouncer = orig }()

	h := handleBoatEffect(sc, nil)
	h(logrus.New(), ctx, system_message2.Command[system_message2.BoatEffectBody]{
		WorldId:     sc.WorldId(),
		ChannelId:   sc.ChannelId(),
		CharacterId: 100,
		Type:        system_message2.CommandBoatEffect,
		Body:        system_message2.BoatEffectBody{Show: true},
	})

	if calls != 0 {
		t.Fatalf("boatEffectAnnouncer calls = %d, want 0 for a tenant with no bound ContiMove writer", calls)
	}
}

// TestHandleBoatEffect_Configured_WritesExactlyAsBefore pins that the guard
// does not regress the ten tenants that DO bind ContiMove, for both the
// show and hide arms.
func TestHandleBoatEffect_Configured_WritesExactlyAsBefore(t *testing.T) {
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	sc := newTestServer(t, tm)
	registerContiMoveWriter(t, tm.Id())

	type call struct {
		characterId uint32
		key         writer.ContiMoveKey
	}
	var calls []call
	orig := boatEffectAnnouncer
	boatEffectAnnouncer = func(_ logrus.FieldLogger, _ context.Context, _ server.Model, _ writer.Producer, characterId uint32, key writer.ContiMoveKey) error {
		calls = append(calls, call{characterId: characterId, key: key})
		return nil
	}
	defer func() { boatEffectAnnouncer = orig }()

	h := handleBoatEffect(sc, nil)
	h(logrus.New(), ctx, system_message2.Command[system_message2.BoatEffectBody]{
		WorldId:     sc.WorldId(),
		ChannelId:   sc.ChannelId(),
		CharacterId: 100,
		Type:        system_message2.CommandBoatEffect,
		Body:        system_message2.BoatEffectBody{Show: true},
	})
	h(logrus.New(), ctx, system_message2.Command[system_message2.BoatEffectBody]{
		WorldId:     sc.WorldId(),
		ChannelId:   sc.ChannelId(),
		CharacterId: 101,
		Type:        system_message2.CommandBoatEffect,
		Body:        system_message2.BoatEffectBody{Show: false},
	})

	if len(calls) != 2 {
		t.Fatalf("boatEffectAnnouncer calls = %d, want 2 for a tenant that binds ContiMove", len(calls))
	}
	if calls[0].characterId != 100 || calls[0].key != writer.ContiMoveShow {
		t.Fatalf("show call = %+v, want {characterId:100 key:SHOW}", calls[0])
	}
	if calls[1].characterId != 101 || calls[1].key != writer.ContiMoveHide {
		t.Fatalf("hide call = %+v, want {characterId:101 key:HIDE}", calls[1])
	}
}
