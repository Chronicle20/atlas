package chat

type Model struct {
	timestamp  int64
	senderId   uint32
	senderName string
	chatType   string
	text       string
}

func (m Model) Timestamp() int64   { return m.timestamp }
func (m Model) SenderId() uint32   { return m.senderId }
func (m Model) SenderName() string { return m.senderName }
func (m Model) ChatType() string   { return m.chatType }
func (m Model) Text() string       { return m.text }
