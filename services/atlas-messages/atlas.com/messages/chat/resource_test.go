package chat

import "testing"

func TestTransform(t *testing.T) {
	line := Line{Timestamp: 123, SenderId: 7, SenderName: "Alice", ChatType: "GENERAL", Text: "hi", WorldId: 0, ChannelId: 1, MapId: 100000000}
	rm := Transform(3, line)
	if rm.GetName() != "chat-messages" {
		t.Errorf("resource name: %s", rm.GetName())
	}
	if rm.GetID() != "3" {
		t.Errorf("id: %s", rm.GetID())
	}
	if rm.Timestamp != 123 || rm.SenderId != 7 || rm.SenderName != "Alice" || rm.ChatType != "GENERAL" || rm.Text != "hi" {
		t.Errorf("attributes mismatch: %+v", rm)
	}
}

func TestParseCharacterIds(t *testing.T) {
	ids, err := parseCharacterIds("1,42")
	if err != nil || len(ids) != 2 || ids[0] != 1 || ids[1] != 42 {
		t.Errorf("parse: %v %v", ids, err)
	}
	if _, err := parseCharacterIds(""); err == nil {
		t.Error("expected error for empty")
	}
	if _, err := parseCharacterIds("1,abc"); err == nil {
		t.Error("expected error for garbage")
	}
}
