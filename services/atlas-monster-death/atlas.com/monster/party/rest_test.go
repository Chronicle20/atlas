package party

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func newTestTenant(t *testing.T) tenant.Model {
	t.Helper()
	tm, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant: %v", err)
	}
	return tm
}

func TestExtract_MapsMembers(t *testing.T) {
	rm := RestModel{
		Id:       7,
		LeaderId: 11,
		Members: []MemberRestModel{
			{Id: 11, Name: "Leader", Level: 120, JobId: 112, WorldId: 0, ChannelId: 1, MapId: 100000000, Instance: uuid.Nil, Online: true},
			{Id: 12, Name: "Member", Level: 30, JobId: 100, WorldId: 0, ChannelId: 1, MapId: 100000000, Instance: uuid.Nil, Online: false},
		},
	}

	m, err := Extract(rm)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if m.Id() != 7 {
		t.Errorf("expected Id() == 7, got %d", m.Id())
	}
	if m.LeaderId() != 11 {
		t.Errorf("expected LeaderId() == 11, got %d", m.LeaderId())
	}
	if len(m.Members()) != 2 {
		t.Fatalf("expected 2 members, got %d", len(m.Members()))
	}

	leader := m.Members()[0]
	if leader.Id() != 11 {
		t.Errorf("expected member[0].Id() == 11, got %d", leader.Id())
	}
	if leader.Name() != "Leader" {
		t.Errorf("expected member[0].Name() == Leader, got %s", leader.Name())
	}
	if leader.Level() != 120 {
		t.Errorf("expected member[0].Level() == 120, got %d", leader.Level())
	}
	if leader.JobId() != 112 {
		t.Errorf("expected member[0].JobId() == 112, got %d", leader.JobId())
	}
	if !leader.Online() {
		t.Errorf("expected member[0].Online() == true")
	}
	if leader.Field().MapId() != _map.Id(100000000) {
		t.Errorf("expected member[0].Field().MapId() == 100000000, got %d", leader.Field().MapId())
	}
	if leader.Field().ChannelId() != channel.Id(1) {
		t.Errorf("expected member[0].Field().ChannelId() == 1, got %d", leader.Field().ChannelId())
	}

	member := m.Members()[1]
	if member.Id() != 12 {
		t.Errorf("expected member[1].Id() == 12, got %d", member.Id())
	}
	if member.Name() != "Member" {
		t.Errorf("expected member[1].Name() == Member, got %s", member.Name())
	}
	if member.Level() != 30 {
		t.Errorf("expected member[1].Level() == 30, got %d", member.Level())
	}
	if member.JobId() != 100 {
		t.Errorf("expected member[1].JobId() == 100, got %d", member.JobId())
	}
	if member.Online() {
		t.Errorf("expected member[1].Online() == false")
	}
}

func TestSetToManyReferenceIDs_SeedsMembers(t *testing.T) {
	r := &RestModel{}
	err := r.SetToManyReferenceIDs("members", []string{"11", "12"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(r.Members) != 2 {
		t.Fatalf("expected 2 members, got %d", len(r.Members))
	}
	if r.Members[0].Id != 11 {
		t.Errorf("expected member[0].Id == 11, got %d", r.Members[0].Id)
	}
	if r.Members[1].Id != 12 {
		t.Errorf("expected member[1].Id == 12, got %d", r.Members[1].Id)
	}

	err = r.SetToManyReferenceIDs("other", []string{"99"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(r.Members) != 2 {
		t.Errorf("expected member count unchanged at 2, got %d", len(r.Members))
	}
}

func TestGetReferencedIDs_And_Structs(t *testing.T) {
	r := RestModel{
		Id:       7,
		LeaderId: 11,
		Members: []MemberRestModel{
			{Id: 11, Name: "Leader", Level: 120, JobId: 112, WorldId: 0, ChannelId: 1, MapId: 100000000, Instance: uuid.Nil, Online: true},
			{Id: 12, Name: "Member", Level: 30, JobId: 100, WorldId: 0, ChannelId: 1, MapId: 100000000, Instance: uuid.Nil, Online: false},
		},
	}

	refs := r.GetReferencedIDs()
	if len(refs) != 2 {
		t.Fatalf("expected 2 referenced IDs, got %d", len(refs))
	}
	for _, ref := range refs {
		if ref.Type != "members" {
			t.Errorf("expected ref.Type == members, got %s", ref.Type)
		}
		if ref.Name != "members" {
			t.Errorf("expected ref.Name == members, got %s", ref.Name)
		}
	}
	if refs[0].ID != "11" {
		t.Errorf("expected refs[0].ID == 11, got %s", refs[0].ID)
	}
	if refs[1].ID != "12" {
		t.Errorf("expected refs[1].ID == 12, got %s", refs[1].ID)
	}

	structs := r.GetReferencedStructs()
	if len(structs) != 2 {
		t.Fatalf("expected 2 referenced structs, got %d", len(structs))
	}
}

func TestBuilders_ProduceReadableModel(t *testing.T) {
	m := NewBuilder(7).
		SetLeaderId(11).
		AddMember(NewMemberBuilder(11).SetLevel(120).SetName("Leader").SetOnline(true).Build()).
		Build()

	if m.Id() != 7 {
		t.Errorf("expected Id() == 7, got %d", m.Id())
	}
	if m.LeaderId() != 11 {
		t.Errorf("expected LeaderId() == 11, got %d", m.LeaderId())
	}
	if len(m.Members()) != 1 {
		t.Fatalf("expected 1 member, got %d", len(m.Members()))
	}
	if m.Members()[0].Level() != 120 {
		t.Errorf("expected member[0].Level() == 120, got %d", m.Members()[0].Level())
	}
}

// TestRequestByMemberId_RoundTrip stands up an httptest server returning a
// realistic JSON:API document for a party resource, INCLUDING a
// relationships block that carries the to-many "members" relationship, and
// drives it through the real requests.GetRequest decode path. This proves
// the SetToOneReferenceID/SetToManyReferenceIDs stubs added per
// libs/atlas-rest/CLAUDE.md let api2go decode a response that carries
// relationships, and pins the wire-tag mapping for leaderId/members.
func TestRequestByMemberId_RoundTrip(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/parties") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if !strings.Contains(r.URL.RawQuery, "filter[members.id]=11") {
			t.Errorf("unexpected query: %s", r.URL.RawQuery)
		}
		w.Header().Set("Content-Type", "application/vnd.api+json")
		_, _ = w.Write([]byte(`{
			"data": [
				{
					"type": "parties",
					"id": "7",
					"attributes": {
						"leaderId": 11
					},
					"relationships": {
						"members": {
							"data": [
								{"type": "members", "id": "11"},
								{"type": "members", "id": "12"}
							]
						}
					}
				}
			],
			"included": [
				{
					"type": "members",
					"id": "11",
					"attributes": {
						"name": "Leader",
						"level": 120,
						"jobId": 112,
						"worldId": 0,
						"channelId": 1,
						"mapId": 100000000,
						"instance": "00000000-0000-0000-0000-000000000000",
						"online": true
					}
				},
				{
					"type": "members",
					"id": "12",
					"attributes": {
						"name": "Member",
						"level": 30,
						"jobId": 100,
						"worldId": 0,
						"channelId": 1,
						"mapId": 100000000,
						"instance": "00000000-0000-0000-0000-000000000000",
						"online": false
					}
				}
			]
		}`))
	}))
	defer srv.Close()

	t.Setenv("PARTIES_SERVICE_URL", srv.URL+"/")

	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)

	rms, err := requestByMemberId(ctx, 11)(logrus.New(), ctx)
	if err != nil {
		t.Fatalf("requestByMemberId: %v", err)
	}
	if len(rms) != 1 {
		t.Fatalf("expected 1 party, got %d", len(rms))
	}
	rm := rms[0]
	if rm.Id != 7 {
		t.Errorf("Id = %d, want 7", rm.Id)
	}
	if rm.LeaderId != 11 {
		t.Errorf("LeaderId = %d, want 11", rm.LeaderId)
	}
	if len(rm.Members) != 2 {
		t.Fatalf("expected 2 members, got %d", len(rm.Members))
	}
	if rm.Members[0].Name != "Leader" || rm.Members[0].Level != 120 {
		t.Errorf("member[0] = %+v, want Leader/120", rm.Members[0])
	}
	if rm.Members[1].Name != "Member" || rm.Members[1].Level != 30 {
		t.Errorf("member[1] = %+v, want Member/30", rm.Members[1])
	}
}
