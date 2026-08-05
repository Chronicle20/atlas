package report

import (
	"encoding/json"
	"testing"

	report2 "atlas-channel/kafka/message/report"
)

func TestSueCommandProviderLegacy(t *testing.T) {
	msgs, err := sueCommandProvider(1, 0, 2, 12345, "", 0x05, "spamming")()
	if err != nil {
		t.Fatalf("provider: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}
	var cmd report2.Command[report2.CreateCommandBody]
	if err := json.Unmarshal(msgs[0].Value, &cmd); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if cmd.Type != report2.CommandTypeCreate {
		t.Errorf("type: %s", cmd.Type)
	}
	b := cmd.Body
	if b.Kind != report2.KindSue || b.ReporterId != 1 || b.AccusedId != 12345 || b.AccusedName != "" ||
		b.ReasonType != 0x05 || b.Description != "spamming" || b.ChatClaim {
		t.Errorf("body mismatch: %+v", b)
	}
}

func TestSueCommandProviderV95SubCommand(t *testing.T) {
	msgs, _ := sueCommandProvider(1, 0, 2, 0, "alice", 0x05, "spamming")()
	var cmd report2.Command[report2.CreateCommandBody]
	_ = json.Unmarshal(msgs[0].Value, &cmd)
	if cmd.Body.AccusedId != 0 || cmd.Body.AccusedName != "alice" {
		t.Errorf("v95 mapping mismatch: %+v", cmd.Body)
	}
}

func TestClaimCommandProvider(t *testing.T) {
	msgs, err := claimCommandProvider(7, 0, 1, "bob", 0x03, "harassment", true, "bob: mean")()
	if err != nil {
		t.Fatalf("provider: %v", err)
	}
	var cmd report2.Command[report2.CreateCommandBody]
	if err := json.Unmarshal(msgs[0].Value, &cmd); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	b := cmd.Body
	if b.Kind != report2.KindClaim || b.ReporterId != 7 || b.AccusedName != "bob" ||
		b.ReasonType != 0x03 || b.Description != "harassment" || !b.ChatClaim || b.ChatLog != "bob: mean" {
		t.Errorf("body mismatch: %+v", b)
	}
}
