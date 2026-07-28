package party

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
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

// TestGetByMemberId_ServedPreservesOrder drives a real JSON:API party
// collection document (with the members to-many relationship + included member
// resources) through the api2go unmarshal + SetReferencedStructs path and
// asserts the members come back in join order (leader at index 0).
func TestGetByMemberId_ServedPreservesOrder(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/parties") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/vnd.api+json")
		_, _ = w.Write([]byte(`{
			"data": [
				{
					"type": "parties",
					"id": "42",
					"attributes": { "leaderId": 100 },
					"relationships": {
						"members": {
							"data": [
								{ "type": "members", "id": "100" },
								{ "type": "members", "id": "200" },
								{ "type": "members", "id": "300" }
							]
						}
					}
				}
			],
			"included": [
				{ "type": "members", "id": "100", "attributes": {} },
				{ "type": "members", "id": "200", "attributes": {} },
				{ "type": "members", "id": "300", "attributes": {} }
			]
		}`))
	}))
	defer srv.Close()
	defer SetBaseURLForTest(srv.URL)()

	ctx := tenant.WithContext(context.Background(), newTestTenant(t))
	m, err := NewProcessor(logrus.New(), ctx).GetByMemberId(200)
	if err != nil {
		t.Fatalf("GetByMemberId: %v", err)
	}

	assert.Equal(t, uint32(42), m.Id())
	assert.Equal(t, character.Id(100), m.LeaderId())

	members := m.Members()
	require.Len(t, members, 3)
	assert.Equal(t, character.Id(100), members[0], "members must be in join order")
	assert.Equal(t, character.Id(200), members[1])
	assert.Equal(t, character.Id(300), members[2])
}

// TestExtractPreservesMemberOrder verifies that Extract keeps members in the
// order they appear in the RestModel (join order, leader seeded at index 0).
// No re-sorting must be applied.
func TestExtractPreservesMemberOrder(t *testing.T) {
	rm := RestModel{
		Id:       1,
		LeaderId: 100,
		Members: []MemberRestModel{
			{Id: 100}, // member[0] — leader (join-order index 0)
			{Id: 200}, // member[1] — second to join
		},
	}

	m, err := Extract(rm)
	require.NoError(t, err)

	assert.Equal(t, uint32(1), m.Id())
	assert.Equal(t, character.Id(100), m.LeaderId())

	members := m.Members()
	require.Len(t, members, 2)
	assert.Equal(t, character.Id(100), members[0], "member[0] must be the first in join order (leader)")
	assert.Equal(t, character.Id(200), members[1], "member[1] must be the second in join order")
}
