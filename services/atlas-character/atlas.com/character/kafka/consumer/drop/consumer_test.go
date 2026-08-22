package drop

import (
	drop2 "atlas-character/kafka/message/drop"
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus/hooks/test"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func TestHandleMesoAwarded(t *testing.T) {
	tests := []struct {
		name string
		run  func(t *testing.T)
	}{
		{
			// Guards FR-15: once handleDropReservation was removed, RESERVED
			// events must fall through handleMesoAwarded's type guard without
			// reaching the nil *gorm.DB passed in below - a nil DB would
			// panic if the guard did not short-circuit first.
			name: "ignores non meso awarded events",
			run: func(t *testing.T) {
				l, _ := test.NewNullLogger()
				tn, err := tenant.Create(uuid.New(), "GMS", 83, 1)
				if err != nil {
					t.Fatalf("Failed to create tenant: %v", err)
				}
				ctx := tenant.WithContext(context.Background(), tn)

				e := drop2.StatusEvent[drop2.MesoAwardedStatusEventBody]{
					Type: drop2.StatusEventTypeReserved,
				}

				handleMesoAwarded(nil)(l, ctx, e)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, tc.run)
	}
}
