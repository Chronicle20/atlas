package parcel_test

import (
	"atlas-saga-orchestrator/kafka/message"
	parcelCustody "atlas-saga-orchestrator/kafka/message/parcel/custody"
	"atlas-saga-orchestrator/parcel"
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus/hooks/test"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func newTestProcessor(t *testing.T) (parcel.Processor, context.Context) {
	t.Helper()
	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatal(err)
	}
	ctx := tenant.WithContext(context.Background(), ten)
	l, _ := test.NewNullLogger()
	return parcel.NewProcessor(l, ctx), ctx
}

// TestParcelProcessorDispatch proves each Processor Buffer method places
// exactly one correctly-typed command message on EnvCommandTopic, without a
// live Kafka producer. Mirrors the MTS custody processor's dispatch shape.
func TestParcelProcessorDispatch(t *testing.T) {
	t.Run("accept", func(t *testing.T) {
		p, _ := newTestProcessor(t)
		mb := message.NewBuffer()
		txId := uuid.New()
		parcelId := uuid.New()

		params := parcel.AcceptToParcelParams{
			ParcelId:    parcelId,
			CharacterId: 100100,
			HasItem:     true,
			RecipientId: 200200,
			TemplateId:  1302000,
		}

		if err := p.AcceptToParcel(mb)(txId, params); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		msgs := mb.GetAll()[parcelCustody.EnvCommandTopic]
		if len(msgs) != 1 {
			t.Fatalf("expected 1 message on %s, got %d", parcelCustody.EnvCommandTopic, len(msgs))
		}

		var cmd parcelCustody.Command[parcelCustody.AcceptToParcelCommandBody]
		if err := json.Unmarshal(msgs[0].Value, &cmd); err != nil {
			t.Fatalf("unable to unmarshal command: %v", err)
		}
		if cmd.Type != parcelCustody.CommandAcceptToParcel {
			t.Fatalf("expected type %s, got %s", parcelCustody.CommandAcceptToParcel, cmd.Type)
		}
		if cmd.TransactionId != txId {
			t.Fatalf("expected transaction id %s, got %s", txId, cmd.TransactionId)
		}
		if cmd.Body.ParcelId != parcelId {
			t.Fatalf("expected parcel id %s, got %s", parcelId, cmd.Body.ParcelId)
		}
		if cmd.Body.RecipientId != 200200 {
			t.Fatalf("expected recipient id 200200, got %d", cmd.Body.RecipientId)
		}
		if cmd.Body.TemplateId != 1302000 {
			t.Fatalf("expected template id 1302000, got %d", cmd.Body.TemplateId)
		}
		if !cmd.Body.HasItem {
			t.Fatalf("expected HasItem true")
		}
	})

	t.Run("accept meso only", func(t *testing.T) {
		p, _ := newTestProcessor(t)
		mb := message.NewBuffer()
		txId := uuid.New()
		parcelId := uuid.New()

		params := parcel.AcceptToParcelParams{
			ParcelId:    parcelId,
			CharacterId: 100100,
			HasItem:     false,
			RecipientId: 200200,
			TemplateId:  0,
		}

		if err := p.AcceptToParcel(mb)(txId, params); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		msgs := mb.GetAll()[parcelCustody.EnvCommandTopic]
		if len(msgs) != 1 {
			t.Fatalf("expected 1 message on %s, got %d", parcelCustody.EnvCommandTopic, len(msgs))
		}

		var cmd parcelCustody.Command[parcelCustody.AcceptToParcelCommandBody]
		if err := json.Unmarshal(msgs[0].Value, &cmd); err != nil {
			t.Fatalf("unable to unmarshal command: %v", err)
		}
		if cmd.Body.HasItem {
			t.Fatalf("expected HasItem false")
		}
		if cmd.Body.TemplateId != 0 {
			t.Fatalf("expected template id 0, got %d", cmd.Body.TemplateId)
		}
	})

	t.Run("release", func(t *testing.T) {
		p, _ := newTestProcessor(t)
		mb := message.NewBuffer()
		txId := uuid.New()
		parcelId := uuid.New()

		if err := p.ReleaseFromParcel(mb)(txId, parcelId, 200200); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		msgs := mb.GetAll()[parcelCustody.EnvCommandTopic]
		if len(msgs) != 1 {
			t.Fatalf("expected 1 message on %s, got %d", parcelCustody.EnvCommandTopic, len(msgs))
		}

		var cmd parcelCustody.Command[parcelCustody.ReleaseFromParcelCommandBody]
		if err := json.Unmarshal(msgs[0].Value, &cmd); err != nil {
			t.Fatalf("unable to unmarshal command: %v", err)
		}
		if cmd.Type != parcelCustody.CommandReleaseFromParcel {
			t.Fatalf("expected type %s, got %s", parcelCustody.CommandReleaseFromParcel, cmd.Type)
		}
		if cmd.Body.ParcelId != parcelId {
			t.Fatalf("expected parcel id %s, got %s", parcelId, cmd.Body.ParcelId)
		}
		if cmd.Body.RecipientId != 200200 {
			t.Fatalf("expected recipient id 200200, got %d", cmd.Body.RecipientId)
		}
	})

	t.Run("restore", func(t *testing.T) {
		p, _ := newTestProcessor(t)
		mb := message.NewBuffer()
		txId := uuid.New()
		parcelId := uuid.New()

		if err := p.RestoreParcel(mb)(txId, parcelId); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		msgs := mb.GetAll()[parcelCustody.EnvCommandTopic]
		if len(msgs) != 1 {
			t.Fatalf("expected 1 message on %s, got %d", parcelCustody.EnvCommandTopic, len(msgs))
		}

		var cmd parcelCustody.Command[parcelCustody.RestoreParcelCommandBody]
		if err := json.Unmarshal(msgs[0].Value, &cmd); err != nil {
			t.Fatalf("unable to unmarshal command: %v", err)
		}
		if cmd.Type != parcelCustody.CommandRestoreParcel {
			t.Fatalf("expected type %s, got %s", parcelCustody.CommandRestoreParcel, cmd.Type)
		}
		if cmd.Body.ParcelId != parcelId {
			t.Fatalf("expected parcel id %s, got %s", parcelId, cmd.Body.ParcelId)
		}
	})

	t.Run("remove", func(t *testing.T) {
		p, _ := newTestProcessor(t)
		mb := message.NewBuffer()
		txId := uuid.New()
		parcelId := uuid.New()

		if err := p.RemoveParcel(mb)(txId, parcelId); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		msgs := mb.GetAll()[parcelCustody.EnvCommandTopic]
		if len(msgs) != 1 {
			t.Fatalf("expected 1 message on %s, got %d", parcelCustody.EnvCommandTopic, len(msgs))
		}

		var cmd parcelCustody.Command[parcelCustody.RemoveParcelCommandBody]
		if err := json.Unmarshal(msgs[0].Value, &cmd); err != nil {
			t.Fatalf("unable to unmarshal command: %v", err)
		}
		if cmd.Type != parcelCustody.CommandRemoveParcel {
			t.Fatalf("expected type %s, got %s", parcelCustody.CommandRemoveParcel, cmd.Type)
		}
		if cmd.Body.ParcelId != parcelId {
			t.Fatalf("expected parcel id %s, got %s", parcelId, cmd.Body.ParcelId)
		}
	})
}
