package note

import (
	"strconv"
	"time"
)

const (
	noteIdPattern      = "noteId"
	characterIdPattern = "characterId"
)

// RestModel is the JSON:API resource for notes
type RestModel struct {
	Id          uint32    `json:"-"`
	CharacterId uint32    `json:"characterId"`
	SenderId    uint32    `json:"senderId"`
	Message     string    `json:"message"`
	Flag        byte      `json:"flag"`
	Timestamp   time.Time `json:"timestamp"`
}

// GetID returns the resource ID
func (n RestModel) GetID() string {
	return strconv.Itoa(int(n.Id))
}

// SetID sets the resource ID
func (n *RestModel) SetID(strId string) error {
	id, err := strconv.Atoi(strId)
	if err != nil {
		return err
	}
	n.Id = uint32(id)
	return nil
}

// GetName returns the resource name
func (n RestModel) GetName() string {
	return "notes"
}

// Transform converts a Model domain model to a RestModel
func Transform(n Model) (RestModel, error) {
	return RestModel{
		Id:          n.Id(),
		CharacterId: n.CharacterId(),
		SenderId:    n.SenderId(),
		Message:     n.Message(),
		Flag:        n.Flag(),
		Timestamp:   n.Timestamp(),
	}, nil
}

// Extract converts a RestModel to parameters for creating or updating a Model
func Extract(r RestModel) (Model, error) {
	return NewBuilder().
		SetId(r.Id).
		SetCharacterId(r.CharacterId).
		SetSenderId(r.SenderId).
		SetMessage(r.Message).
		SetFlag(r.Flag).
		SetTimestamp(r.Timestamp).
		Build()
}
