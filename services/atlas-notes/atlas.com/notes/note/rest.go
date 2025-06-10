package note

import (
	"fmt"
	"strconv"
)

const (
	noteIdPattern      = "noteId"
	characterIdPattern = "characterId"
)

// RestModel is the JSON:API resource for notes
type RestModel struct {
	Id          string `json:"-"`
	CharacterId string `json:"characterId"`
	SenderId    string `json:"senderId"`
	Message     string `json:"message"`
	Flag        string `json:"flag"`
	Timestamp   string `json:"timestamp"`
}

// GetID returns the resource ID
func (n RestModel) GetID() string {
	return n.Id
}

// SetID sets the resource ID
func (n *RestModel) SetID(id string) error {
	n.Id = id
	return nil
}

// GetName returns the resource name
func (n RestModel) GetName() string {
	return "notes"
}

// Transform converts a Model domain model to a RestModel
func Transform(n Model) (RestModel, error) {
	return RestModel{
		Id:          fmt.Sprintf("%d", n.Id()),
		CharacterId: fmt.Sprintf("%d", n.CharacterId()),
		SenderId:    fmt.Sprintf("%d", n.SenderId()),
		Message:     n.Message(),
		Flag:        fmt.Sprintf("%d", n.Flag()),
		Timestamp:   n.Timestamp().Format("2006-01-02T15:04:05Z07:00"),
	}, nil
}

// Extract converts a RestModel to parameters for creating or updating a Model
func Extract(r RestModel) (Model, error) {
	characterId, err := strconv.ParseUint(r.CharacterId, 10, 32)
	if err != nil {
		return Model{}, err
	}

	senderId, err := strconv.ParseUint(r.SenderId, 10, 32)
	if err != nil {
		return Model{}, err
	}

	flag, err := strconv.ParseUint(r.Flag, 10, 8)
	if err != nil {
		return Model{}, err
	}
	return NewBuilder().
		SetCharacterId(uint32(characterId)).
		SetSenderId(uint32(senderId)).
		SetMessage(r.Message).
		SetFlag(byte(flag)).
		Build(), nil
}
