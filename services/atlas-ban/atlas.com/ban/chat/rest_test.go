package chat

import "testing"

func TestExtract(t *testing.T) {
	rm := RestModel{
		Id:         "0",
		Timestamp:  1720540800123,
		SenderId:   7,
		SenderName: "Alice",
		ChatType:   "GENERAL",
		Text:       "hello",
	}
	m, err := Extract(rm)
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if m.Timestamp() != 1720540800123 || m.SenderId() != 7 || m.SenderName() != "Alice" ||
		m.ChatType() != "GENERAL" || m.Text() != "hello" {
		t.Errorf("mismatch: %+v", m)
	}
}
