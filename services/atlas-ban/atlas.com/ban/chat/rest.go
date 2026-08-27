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

// SetToOneReferenceID and SetToManyReferenceIDs are required even though this
// client doesn't use relationship data: api2go's jsonapi.Unmarshal fails the
// entire decode if the response carries a relationships block and the target
// struct doesn't implement these (see libs/atlas-rest/CLAUDE.md).
func (r *RestModel) SetToOneReferenceID(_, _ string) error {
	return nil
}

func (r *RestModel) SetToManyReferenceIDs(_ string, _ []string) error {
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

func Transform(m Model) (RestModel, error) {
	return RestModel{
		Timestamp:  m.timestamp,
		SenderId:   m.senderId,
		SenderName: m.senderName,
		ChatType:   m.chatType,
		Text:       m.text,
	}, nil
}
