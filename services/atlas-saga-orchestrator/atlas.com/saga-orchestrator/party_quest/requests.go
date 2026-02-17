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
	EnvCommandTopic    = "COMMAND_TOPIC_PARTY_QUEST"
	CommandTypeRegister = "REGISTER"
	CommandTypeLeave    = "LEAVE"
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

// MemberRestModel represents a party member from the atlas-parties REST API
type MemberRestModel struct {
	Id        uint32     `json:"-"`
	Name      string     `json:"name"`
	Level     byte       `json:"level"`
	JobId     uint16     `json:"jobId"`
	WorldId   world.Id   `json:"worldId"`
	ChannelId channel.Id `json:"channelId"`
	MapId     uint32     `json:"mapId"`
	Online    bool       `json:"online"`
}

func (r MemberRestModel) GetName() string {
	return "members"
}

func (r MemberRestModel) GetID() string {
	return strconv.FormatUint(uint64(r.Id), 10)
}

func (r *MemberRestModel) SetID(idStr string) error {
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		return err
	}
	r.Id = uint32(id)
	return nil
}

// ExtractMember is a pass-through extractor for MemberRestModel
func ExtractMember(r MemberRestModel) (MemberRestModel, error) {
	return r, nil
}

func requestPartyMembers(partyId uint32) requests.Request[[]MemberRestModel] {
	return rest.MakeGetRequest[[]MemberRestModel](fmt.Sprintf(getBaseRequest()+"parties/%d/members", partyId))
}

// ConditionRestModel represents a start requirement condition from atlas-party-quests
type ConditionRestModel struct {
	Type        string `json:"type"`
	Operator    string `json:"operator"`
	Value       uint32 `json:"value"`
	ReferenceId uint32 `json:"referenceId"`
}

// DefinitionRestModel represents a party quest definition from atlas-party-quests
type DefinitionRestModel struct {
	Id                string               `json:"-"`
	StartRequirements []ConditionRestModel `json:"startRequirements"`
}

func (r DefinitionRestModel) GetName() string {
	return "definitions"
}

func (r DefinitionRestModel) GetID() string {
	return r.Id
}

func (r *DefinitionRestModel) SetID(idStr string) error {
	r.Id = idStr
	return nil
}

func (r DefinitionRestModel) GetReferences() []jsonapi.Reference {
	return []jsonapi.Reference{}
}

func (r DefinitionRestModel) GetReferencedIDs() []jsonapi.ReferenceID {
	return []jsonapi.ReferenceID{}
}

func (r DefinitionRestModel) GetReferencedStructs() []jsonapi.MarshalIdentifier {
	return []jsonapi.MarshalIdentifier{}
}

func (r *DefinitionRestModel) SetToOneReferenceID(_, _ string) error {
	return nil
}

func (r *DefinitionRestModel) SetToManyReferenceIDs(_ string, _ []string) error {
	return nil
}

func (r *DefinitionRestModel) SetReferencedStructs(_ map[string]map[string]jsonapi.Data) error {
	return nil
}

// ExtractDefinition is a pass-through extractor for DefinitionRestModel
func ExtractDefinition(r DefinitionRestModel) (DefinitionRestModel, error) {
	return r, nil
}

func getPartyQuestsBaseRequest() string {
	return requests.RootUrl("PARTY_QUESTS")
}

func requestDefinitionByQuestId(questId string) requests.Request[DefinitionRestModel] {
	return rest.MakeGetRequest[DefinitionRestModel](fmt.Sprintf(getPartyQuestsBaseRequest()+"party-quests/definitions/quest/%s", questId))
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

// LeaveCommandBody represents the body of a LEAVE command
type LeaveCommandBody struct {
}
