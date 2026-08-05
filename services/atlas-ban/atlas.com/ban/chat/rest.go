package chat

// RestModel mirrors the atlas-messages "chat-messages" resource.
type RestModel struct {
	Id         string `json:"-"`
	Timestamp  int64  `json:"timestamp"`
	SenderId   uint32 `json:"senderId"`
	SenderName string `json:"senderName"`
	ChatType   string `json:"chatType"`
	Text       string `json:"text"`
}

func (r RestModel) GetName() string {
	return "chat-messages"
}

func (r RestModel) GetID() string {
	return r.Id
}

func (r *RestModel) SetID(idStr string) error {
	r.Id = idStr
	return nil
}

func Extract(rm RestModel) (Model, error) {
	return Model{
		timestamp:  rm.Timestamp,
		senderId:   rm.SenderId,
		senderName: rm.SenderName,
		chatType:   rm.ChatType,
		text:       rm.Text,
	}, nil
}
