package party_quest

import (
	"atlas-saga-orchestrator/rest"
	"fmt"
	"strconv"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/jtumidanski/api2go/jsonapi"
)

const (
	EnvCommandTopic  = "COMMAND_TOPIC_PARTY_QUEST"
	CommandTypeRegister = "REGISTER"
)

func getBaseRequest() string {
	return requests.RootUrl("PARTIES")
}

func requestPartyByMemberId(memberId uint32) requests.Request[[]PartyRestModel] {
	return rest.MakeGetRequest[[]PartyRestModel](fmt.Sprintf(getBaseRequest()+"parties?filter[members.id]=%d", memberId))
}

// PartyRestModel represents a party from the atlas-parties REST API
type PartyRestModel struct {
	Id       uint32 `json:"-"`
	LeaderId uint32 `json:"leaderId"`
}

func (r PartyRestModel) GetID() string {
	return strconv.FormatUint(uint64(r.Id), 10)
}

func (r *PartyRestModel) SetID(idStr string) error {
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		return err
	}
	r.Id = uint32(id)
	return nil
}

func (r PartyRestModel) GetName() string {
	return "parties"
}

func (r PartyRestModel) GetReferences() []jsonapi.Reference {
	return []jsonapi.Reference{
		{
			Type: "members",
			Name: "members",
		},
	}
}

func (r PartyRestModel) GetReferencedIDs() []jsonapi.ReferenceID {
	return []jsonapi.ReferenceID{}
}

func (r PartyRestModel) GetReferencedStructs() []jsonapi.MarshalIdentifier {
	return []jsonapi.MarshalIdentifier{}
}

func (r *PartyRestModel) SetToOneReferenceID(_, _ string) error {
	return nil
}

func (r *PartyRestModel) SetToManyReferenceIDs(_ string, _ []string) error {
	return nil
}

func (r *PartyRestModel) SetReferencedStructs(_ map[string]map[string]jsonapi.Data) error {
	return nil
}

// ExtractParty is a pass-through extractor for PartyRestModel
func ExtractParty(r PartyRestModel) (PartyRestModel, error) {
	return r, nil
}

// Command represents a command message to atlas-party-quests
type Command[E any] struct {
	WorldId     world.Id `json:"worldId"`
	CharacterId uint32   `json:"characterId"`
	Type        string   `json:"type"`
	Body        E        `json:"body"`
}

// RegisterCommandBody represents the body of a REGISTER command
type RegisterCommandBody struct {
	QuestId   string     `json:"questId"`
	PartyId   uint32     `json:"partyId,omitempty"`
	ChannelId channel.Id `json:"channelId"`
	MapId     uint32     `json:"mapId"`
}
