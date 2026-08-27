package account

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	AccountsResource = "accounts"
	AccountsById     = AccountsResource + "/%d"
	PicAttempts      = AccountsResource + "/%d/pic-attempts"
	CharacterSlots   = AccountsResource + "/%d/worlds/%d/character-slots"
)

func getBaseRequest(ctx context.Context) (string, error) {
	return requests.RootUrlFor(ctx, "ACCOUNTS")
}

func requestAccountById(ctx context.Context, id uint32) requests.Request[RestModel] {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return requests.ErrorRequest[RestModel](err)
	}
	return requests.GetRequest[RestModel](fmt.Sprintf(root+AccountsById, id))
}

// requestRecordPicAttempt mirrors atlas-login's account/requests.go — it is
// the lockout counter behind the account PIC (the credential these two
// cash-shop check ops validate). Record on both outcomes: success resets the
// counter, failure increments it.
func requestRecordPicAttempt(ctx context.Context, accountId uint32, success bool, ipAddress string, hwid string) requests.Request[PicAttemptOutputRestModel] {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return requests.ErrorRequest[PicAttemptOutputRestModel](err)
	}
	input := PicAttemptInputRestModel{Success: success, IpAddress: ipAddress, HWID: hwid}
	return requests.PostRequest[PicAttemptOutputRestModel](fmt.Sprintf(root+PicAttempts, accountId), input)
}

// requestCharacterSlots reads the per-(account, world) character-slot count
// from atlas-account's world-scoped sub-resource (task-246
// bug-b-type-must-add-a-slot.md), replacing the flat, always-4
// RestModel.CharacterSlots this account fetched before.
func requestCharacterSlots(ctx context.Context, accountId uint32, worldId byte) requests.Request[CharacterSlotRestModel] {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return requests.ErrorRequest[CharacterSlotRestModel](err)
	}
	return requests.GetRequest[CharacterSlotRestModel](fmt.Sprintf(root+CharacterSlots, accountId, worldId))
}

// requestIncrementCharacterSlots adds one slot to an (account, world) pair,
// mirroring requestRecordPicAttempt's shape -- the request body is ignored
// server-side (atlas-account's increment_account_character_slots route takes
// no input handler), so an empty CharacterSlotRestModel is sent purely to
// satisfy jsonapi.Marshal.
func requestIncrementCharacterSlots(ctx context.Context, accountId uint32, worldId byte) requests.Request[CharacterSlotRestModel] {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return requests.ErrorRequest[CharacterSlotRestModel](err)
	}
	return requests.PostRequest[CharacterSlotRestModel](fmt.Sprintf(root+CharacterSlots, accountId, worldId), CharacterSlotRestModel{})
}
