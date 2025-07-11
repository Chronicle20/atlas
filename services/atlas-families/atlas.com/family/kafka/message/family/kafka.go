package family

import (
	"time"
)

// Command represents a generic command wrapper for family operations
type Command[E any] struct {
	TransactionId string `json:"transactionId"`
	CharacterId   uint32 `json:"characterId"`
	Type          string `json:"type"`
	Body          E      `json:"body"`
}

// Event represents a generic event wrapper for family operations
type Event[E any] struct {
	TransactionId string `json:"transactionId"`
	CharacterId   uint32 `json:"characterId"`
	Type          string `json:"type"`
	Body          E      `json:"body"`
}

// Command Body Types

// AddJuniorCommandBody represents the body for adding a junior to a family
type AddJuniorCommandBody struct {
	JuniorId uint32 `json:"juniorId"`
}

// RemoveMemberCommandBody represents the body for removing a member from a family
type RemoveMemberCommandBody struct {
	TargetId uint32 `json:"targetId"`
	Reason   string `json:"reason,omitempty"`
}

// BreakLinkCommandBody represents the body for breaking a family link
type BreakLinkCommandBody struct {
	Reason string `json:"reason,omitempty"`
}

// DeductRepCommandBody represents the body for deducting reputation
type DeductRepCommandBody struct {
	Amount uint32 `json:"amount"`
	Reason string `json:"reason"`
}

// AwardRepCommandBody represents the body for awarding reputation
type AwardRepCommandBody struct {
	Amount    uint32    `json:"amount"`
	Source    string    `json:"source"`
	Timestamp time.Time `json:"timestamp"`
}

// RegisterKillActivityCommandBody represents the body for registering kill activity
type RegisterKillActivityCommandBody struct {
	KillCount uint32    `json:"killCount"`
	Timestamp time.Time `json:"timestamp"`
}

// RegisterExpeditionActivityCommandBody represents the body for registering expedition activity
type RegisterExpeditionActivityCommandBody struct {
	CoinReward uint32    `json:"coinReward"`
	Timestamp  time.Time `json:"timestamp"`
}

// Event Body Types

// LinkCreatedEventBody represents the body for link created events
type LinkCreatedEventBody struct {
	SeniorId  uint32    `json:"seniorId"`
	JuniorId  uint32    `json:"juniorId"`
	Timestamp time.Time `json:"timestamp"`
}

// LinkBrokenEventBody represents the body for link broken events
type LinkBrokenEventBody struct {
	SeniorId  uint32    `json:"seniorId"`
	JuniorId  uint32    `json:"juniorId"`
	Reason    string    `json:"reason"`
	Timestamp time.Time `json:"timestamp"`
}

// TreeDissolvedEventBody represents the body for tree dissolved events
type TreeDissolvedEventBody struct {
	SeniorId       uint32    `json:"seniorId"`
	AffectedIds    []uint32  `json:"affectedIds"`
	Reason         string    `json:"reason"`
	Timestamp      time.Time `json:"timestamp"`
}

// RepGainedEventBody represents the body for reputation gained events
type RepGainedEventBody struct {
	RepGained uint32    `json:"repGained"`
	DailyRep  uint32    `json:"dailyRep"`
	Source    string    `json:"source"`
	Timestamp time.Time `json:"timestamp"`
}

// RepRedeemedEventBody represents the body for reputation redeemed events
type RepRedeemedEventBody struct {
	RepRedeemed uint32    `json:"repRedeemed"`
	Reason      string    `json:"reason"`
	Timestamp   time.Time `json:"timestamp"`
}

// RepPenalizedEventBody represents the body for reputation penalized events
type RepPenalizedEventBody struct {
	RepLost   uint32    `json:"repLost"`
	Reason    string    `json:"reason"`
	Timestamp time.Time `json:"timestamp"`
}

// RepCappedEventBody represents the body for reputation capped events
type RepCappedEventBody struct {
	AttemptedAmount uint32    `json:"attemptedAmount"`
	DailyRep        uint32    `json:"dailyRep"`
	Source          string    `json:"source"`
	Timestamp       time.Time `json:"timestamp"`
}

// RepResetEventBody represents the body for reputation reset events
type RepResetEventBody struct {
	PreviousDailyRep uint32    `json:"previousDailyRep"`
	Timestamp        time.Time `json:"timestamp"`
}

// BuffRedeemedEventBody represents the body for buff redeemed events
type BuffRedeemedEventBody struct {
	BuffType      string    `json:"buffType"`
	RepCost       uint32    `json:"repCost"`
	Duration      uint32    `json:"duration"`
	Timestamp     time.Time `json:"timestamp"`
}

// TeleportUsedEventBody represents the body for teleport used events
type TeleportUsedEventBody struct {
	TargetId   uint32    `json:"targetId"`
	RepCost    uint32    `json:"repCost"`
	World      byte      `json:"world"`
	Map        uint32    `json:"map"`
	Timestamp  time.Time `json:"timestamp"`
}

// Error Event Body Types

// RepErrorEventBody represents the body for reputation error events
type RepErrorEventBody struct {
	ErrorCode    string    `json:"errorCode"`
	ErrorMessage string    `json:"errorMessage"`
	Amount       uint32    `json:"amount"`
	Timestamp    time.Time `json:"timestamp"`
}

// LinkErrorEventBody represents the body for link error events
type LinkErrorEventBody struct {
	SeniorId     uint32    `json:"seniorId"`
	JuniorId     uint32    `json:"juniorId"`
	ErrorCode    string    `json:"errorCode"`
	ErrorMessage string    `json:"errorMessage"`
	Timestamp    time.Time `json:"timestamp"`
}

// Command Type Constants
const (
	CommandTypeAddJunior                   = "ADD_JUNIOR"
	CommandTypeRemoveMember                = "REMOVE_MEMBER"
	CommandTypeBreakLink                   = "BREAK_LINK"
	CommandTypeRedeemRep                   = "REDEEM_REP"
	CommandTypeRegisterKillActivity        = "REGISTER_KILL_ACTIVITY"
	CommandTypeRegisterExpeditionActivity  = "REGISTER_EXPEDITION_ACTIVITY"
)

// Event Type Constants
const (
	EventTypeLinkCreated        = "LINK_CREATED"
	EventTypeLinkBroken         = "LINK_BROKEN"
	EventTypeTreeDissolved      = "TREE_DISSOLVED"
	EventTypeRepGained          = "REP_GAINED"
	EventTypeRepRedeemed        = "REP_REDEEMED"
	EventTypeRepPenalized       = "REP_PENALIZED"
	EventTypeRepCapped          = "REP_CAPPED"
	EventTypeRepReset           = "REP_RESET"
	EventTypeBuffRedeemed       = "BUFF_REDEEMED"
	EventTypeTeleportUsed       = "TELEPORT_USED"
	EventTypeRepError           = "REP_ERROR"
	EventTypeLinkError          = "LINK_ERROR"
)

// Helper functions for creating typed commands and events

// NewAddJuniorCommand creates a new AddJunior command
func NewAddJuniorCommand(transactionId string, characterId uint32, juniorId uint32) Command[AddJuniorCommandBody] {
	return Command[AddJuniorCommandBody]{
		TransactionId: transactionId,
		CharacterId:   characterId,
		Type:          CommandTypeAddJunior,
		Body: AddJuniorCommandBody{
			JuniorId: juniorId,
		},
	}
}

// NewRemoveMemberCommand creates a new RemoveMember command
func NewRemoveMemberCommand(transactionId string, characterId uint32, targetId uint32, reason string) Command[RemoveMemberCommandBody] {
	return Command[RemoveMemberCommandBody]{
		TransactionId: transactionId,
		CharacterId:   characterId,
		Type:          CommandTypeRemoveMember,
		Body: RemoveMemberCommandBody{
			TargetId: targetId,
			Reason:   reason,
		},
	}
}

// NewBreakLinkCommand creates a new BreakLink command
func NewBreakLinkCommand(transactionId string, characterId uint32, reason string) Command[BreakLinkCommandBody] {
	return Command[BreakLinkCommandBody]{
		TransactionId: transactionId,
		CharacterId:   characterId,
		Type:          CommandTypeBreakLink,
		Body: BreakLinkCommandBody{
			Reason: reason,
		},
	}
}

// NewDeductRepCommand creates a new DeductRep command
func NewDeductRepCommand(transactionId string, characterId uint32, amount uint32, reason string) Command[DeductRepCommandBody] {
	return Command[DeductRepCommandBody]{
		TransactionId: transactionId,
		CharacterId:   characterId,
		Type:          CommandTypeRedeemRep,
		Body: DeductRepCommandBody{
			Amount: amount,
			Reason: reason,
		},
	}
}

// NewRegisterKillActivityCommand creates a new RegisterKillActivity command
func NewRegisterKillActivityCommand(transactionId string, characterId uint32, killCount uint32) Command[RegisterKillActivityCommandBody] {
	return Command[RegisterKillActivityCommandBody]{
		TransactionId: transactionId,
		CharacterId:   characterId,
		Type:          CommandTypeRegisterKillActivity,
		Body: RegisterKillActivityCommandBody{
			KillCount: killCount,
			Timestamp: time.Now(),
		},
	}
}

// NewRegisterExpeditionActivityCommand creates a new RegisterExpeditionActivity command
func NewRegisterExpeditionActivityCommand(transactionId string, characterId uint32, coinReward uint32) Command[RegisterExpeditionActivityCommandBody] {
	return Command[RegisterExpeditionActivityCommandBody]{
		TransactionId: transactionId,
		CharacterId:   characterId,
		Type:          CommandTypeRegisterExpeditionActivity,
		Body: RegisterExpeditionActivityCommandBody{
			CoinReward: coinReward,
			Timestamp:  time.Now(),
		},
	}
}

// NewLinkCreatedEvent creates a new LinkCreated event
func NewLinkCreatedEvent(transactionId string, characterId uint32, seniorId uint32, juniorId uint32) Event[LinkCreatedEventBody] {
	return Event[LinkCreatedEventBody]{
		TransactionId: transactionId,
		CharacterId:   characterId,
		Type:          EventTypeLinkCreated,
		Body: LinkCreatedEventBody{
			SeniorId:  seniorId,
			JuniorId:  juniorId,
			Timestamp: time.Now(),
		},
	}
}

// NewLinkBrokenEvent creates a new LinkBroken event
func NewLinkBrokenEvent(transactionId string, characterId uint32, seniorId uint32, juniorId uint32, reason string) Event[LinkBrokenEventBody] {
	return Event[LinkBrokenEventBody]{
		TransactionId: transactionId,
		CharacterId:   characterId,
		Type:          EventTypeLinkBroken,
		Body: LinkBrokenEventBody{
			SeniorId:  seniorId,
			JuniorId:  juniorId,
			Reason:    reason,
			Timestamp: time.Now(),
		},
	}
}

// NewRepGainedEvent creates a new RepGained event
func NewRepGainedEvent(transactionId string, characterId uint32, repGained uint32, dailyRep uint32, source string) Event[RepGainedEventBody] {
	return Event[RepGainedEventBody]{
		TransactionId: transactionId,
		CharacterId:   characterId,
		Type:          EventTypeRepGained,
		Body: RepGainedEventBody{
			RepGained: repGained,
			DailyRep:  dailyRep,
			Source:    source,
			Timestamp: time.Now(),
		},
	}
}

// NewRepRedeemedEvent creates a new RepRedeemed event
func NewRepRedeemedEvent(transactionId string, characterId uint32, repRedeemed uint32, reason string) Event[RepRedeemedEventBody] {
	return Event[RepRedeemedEventBody]{
		TransactionId: transactionId,
		CharacterId:   characterId,
		Type:          EventTypeRepRedeemed,
		Body: RepRedeemedEventBody{
			RepRedeemed: repRedeemed,
			Reason:      reason,
			Timestamp:   time.Now(),
		},
	}
}

// NewRepErrorEvent creates a new RepError event
func NewRepErrorEvent(transactionId string, characterId uint32, errorCode string, errorMessage string, amount uint32) Event[RepErrorEventBody] {
	return Event[RepErrorEventBody]{
		TransactionId: transactionId,
		CharacterId:   characterId,
		Type:          EventTypeRepError,
		Body: RepErrorEventBody{
			ErrorCode:    errorCode,
			ErrorMessage: errorMessage,
			Amount:       amount,
			Timestamp:    time.Now(),
		},
	}
}

// NewLinkErrorEvent creates a new LinkError event
func NewLinkErrorEvent(transactionId string, characterId uint32, seniorId uint32, juniorId uint32, errorCode string, errorMessage string) Event[LinkErrorEventBody] {
	return Event[LinkErrorEventBody]{
		TransactionId: transactionId,
		CharacterId:   characterId,
		Type:          EventTypeLinkError,
		Body: LinkErrorEventBody{
			SeniorId:     seniorId,
			JuniorId:     juniorId,
			ErrorCode:    errorCode,
			ErrorMessage: errorMessage,
			Timestamp:    time.Now(),
		},
	}
}