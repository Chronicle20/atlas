package compartment

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
)

func TestCreateAssetCommandBodyRoundTrip(t *testing.T) {
	in := Command[CreateAssetCommandBody]{
		TransactionId: uuid.New(),
		CharacterId:   42,
		InventoryType: 2,
		Type:          CommandCreateAsset,
		Body:          CreateAssetCommandBody{TemplateId: 1132010, Quantity: 1},
	}
	b, err := json.Marshal(in)
	if err != nil {
		t.Fatal(err)
	}
	var out Command[CreateAssetCommandBody]
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}
	if out.Type != CommandCreateAsset || out.Body.TemplateId != 1132010 {
		t.Fatalf("round-trip mismatch: %+v", out)
	}
}

func TestCreateResultEventDeserializesBothShapes(t *testing.T) {
	tid := uuid.New()
	created := `{"transactionId":"` + tid.String() + `","characterId":1,"type":"CREATED","body":{"type":2,"capacity":24}}`
	failed := `{"transactionId":"` + tid.String() + `","characterId":1,"type":"CREATION_FAILED","body":{"errorCode":"CREATE_ASSET_INVENTORY_FULL","message":"full"}}`

	var ce StatusEvent[CreateResultEventBody]
	if err := json.Unmarshal([]byte(created), &ce); err != nil {
		t.Fatal(err)
	}
	if ce.Type != StatusEventTypeCreated || ce.TransactionId != tid || ce.Body.Capacity != 24 {
		t.Fatalf("created parse: %+v", ce)
	}
	var fe StatusEvent[CreateResultEventBody]
	if err := json.Unmarshal([]byte(failed), &fe); err != nil {
		t.Fatal(err)
	}
	if fe.Type != StatusEventTypeCreationFailed || fe.Body.ErrorCode != CreateAssetInventoryFull {
		t.Fatalf("failed parse: %+v", fe)
	}
}
