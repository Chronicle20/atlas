package chat

// Line is one captured chat line in the bounded per-character buffer.
// Stored as JSON in a timestamp-scored Redis sorted set; short-retention
// working state, not an archive.
type Line struct {
	Timestamp  int64  `json:"ts"` // unix-milli, stamped at capture (the wire carries no timestamp)
	SenderId   uint32 `json:"senderId"`
	SenderName string `json:"senderName"`
	ChatType   string `json:"type"`
	Text       string `json:"text"`
	WorldId    byte   `json:"worldId"`
	ChannelId  byte   `json:"channelId"`
	MapId      uint32 `json:"mapId"`
}
