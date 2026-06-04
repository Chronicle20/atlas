package cashshop

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"testing"

	"atlas-channel/kafka/message/cashshop"

	cashsb "github.com/Chronicle20/atlas/libs/atlas-packet/cash/serverbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	testlog "github.com/sirupsen/logrus/hooks/test"
)

// jmsContext builds a context carrying a JMS185 tenant so region-dispatched
// decoders take the JMS branch.
func jmsContext(t *testing.T) context.Context {
	t.Helper()
	tn, err := tenant.Create(uuid.New(), "JMS", 185, 1)
	if err != nil {
		t.Fatalf("create tenant: %v", err)
	}
	return tenant.WithContext(context.Background(), tn)
}

// TestRequestPurchaseCommandProvider_JMSBuyRoutesSerial proves the JMS cash-shop
// buy flows into the existing wallet command: a JMS185 ShopOperationBuy on the
// wire (isPoints + serialNumber, no currency) decodes to a serial that the
// region-agnostic RequestPurchaseCommandProvider carries into the
// REQUEST_PURCHASE command body. Currency defaults to 0 (no currency on the JMS
// wire); the downstream atlas-cashshop wallet flow reads currency+serial.
func TestRequestPurchaseCommandProvider_JMSBuyRoutesSerial(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := jmsContext(t)

	const serial = uint32(0xAABBCCDD)

	// JMS185 buy body on the wire: isPoints(1) + serialNumber(4). No currency.
	body := make([]byte, 0, 5)
	body = append(body, 0x01) // isPoints = true
	body = binary.LittleEndian.AppendUint32(body, serial)

	req := request.Request(body)
	r := request.NewRequestReader(&req, 0)

	// Decode the (region-dispatched) JMS serverbound buy.
	sp := &cashsb.ShopOperationBuy{}
	sp.Decode(l, ctx)(&r, nil)

	if sp.SerialNumber() != serial {
		t.Fatalf("decoded serial = 0x%08x, want 0x%08x", sp.SerialNumber(), serial)
	}
	if !sp.IsPoints() {
		t.Fatalf("decoded isPoints = false, want true")
	}
	// JMS wire carries no currency; it stays zero (defaulted server-side).
	if sp.Currency() != 0 {
		t.Fatalf("decoded currency = %d, want 0 (JMS has no currency on the wire)", sp.Currency())
	}

	// Drive the decoded buy through the same producer the handler uses:
	// RequestPurchase(...) -> RequestPurchaseCommandProvider(characterId, serial, currency).
	const characterId = uint32(42)
	provider := RequestPurchaseCommandProvider(characterId, sp.SerialNumber(), sp.Currency())

	msgs, err := provider()
	if err != nil {
		t.Fatalf("provider error: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("got %d messages, want 1", len(msgs))
	}

	var cmd cashshop.Command[cashshop.RequestPurchaseCommandBody]
	if err := json.Unmarshal(msgs[0].Value, &cmd); err != nil {
		t.Fatalf("unmarshal command: %v", err)
	}

	if cmd.Type != cashshop.CommandTypeRequestPurchase {
		t.Errorf("Type = %q, want %q", cmd.Type, cashshop.CommandTypeRequestPurchase)
	}
	if cmd.CharacterId != characterId {
		t.Errorf("CharacterId = %d, want %d", cmd.CharacterId, characterId)
	}
	// The serial from the JMS wire must survive into the wallet command body.
	if cmd.Body.SerialNumber != serial {
		t.Errorf("Body.SerialNumber = 0x%08x, want 0x%08x", cmd.Body.SerialNumber, serial)
	}
	// Currency defaults to 0 for a JMS buy (carried verbatim by the command body).
	if cmd.Body.Currency != 0 {
		t.Errorf("Body.Currency = %d, want 0", cmd.Body.Currency)
	}
}
