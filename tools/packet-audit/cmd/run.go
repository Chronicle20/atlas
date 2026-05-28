package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/Chronicle20/atlas/tools/packet-audit/internal/atlaspacket"
	csvpkg "github.com/Chronicle20/atlas/tools/packet-audit/internal/csv"
	"github.com/Chronicle20/atlas/tools/packet-audit/internal/diff"
	"github.com/Chronicle20/atlas/tools/packet-audit/internal/idasrc"
	"github.com/Chronicle20/atlas/tools/packet-audit/internal/report"
	tpl "github.com/Chronicle20/atlas/tools/packet-audit/internal/template"
)

func runPipeline(opts Options, stderr io.Writer) int {
	template, err := tpl.Load(opts.Template)
	if err != nil {
		fmt.Fprintln(stderr, "template:", err)
		return 3
	}
	src, err := openIDASource(opts.IDASource)
	if err != nil {
		fmt.Fprintln(stderr, "ida-source:", err)
		return 3
	}
	reg, err := atlaspacket.NewTypeRegistry(opts.AtlasPacket)
	if err != nil {
		fmt.Fprintln(stderr, "type-registry:", err)
		return 3
	}

	ctx := atlaspacket.GuardContext{
		Region:       template.Region,
		MajorVersion: template.MajorVersion,
		MinorVersion: template.MinorVersion,
	}
	outDir := filepath.Join(opts.Output, fmt.Sprintf("%s_v%d", strings.ToLower(template.Region), template.MajorVersion))
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		fmt.Fprintln(stderr, "mkdir:", err)
		return 3
	}

	worstVerdict := diff.VerdictMatch
	var summary []report.Packet

	process := func(direction csvpkg.Direction, name string, fname string, pathHint string) {
		fields, err := src.Resolve(context.Background(), fname)
		if err != nil {
			if errors.Is(err, idasrc.ErrMCPUnavailable) {
				return
			}
			var notFound idasrc.ErrFunctionNotFound
			if errors.As(err, &notFound) {
				return
			}
			fmt.Fprintln(stderr, "resolve", fname+":", err)
			return
		}
		atlasPath, found := locateAtlasFile(opts.AtlasPacket, name, direction, pathHint)
		if !found {
			return
		}
		calls, err := atlaspacket.AnalyzeFileWithRegistry(atlasPath, name, methodName(direction), reg)
		if err != nil {
			fmt.Fprintln(stderr, "analyze", name+":", err)
			return
		}
		flat := diff.FlattenWithRegistry(calls, ctx, reg)
		rows := diff.Diff(flat, fields)
		v := worstRow(rows)
		pkt := report.Packet{
			WriterName:  name,
			IDAName:     fname,
			Address:     fields.Address,
			Variant:     fmt.Sprintf("%s/v%d", ctx.Region, ctx.MajorVersion),
			BranchDepth: branchDepth(calls),
			AtlasFile:   atlasPath,
			Rows:        rows,
			Verdict:     v,
		}
		if v > worstVerdict {
			worstVerdict = v
		}
		summary = append(summary, pkt)
		if err := report.WritePacket(outDir, pkt); err != nil {
			fmt.Fprintln(stderr, "write", name+":", err)
		}
	}

	// Only audit packets that have an explicit IDA export entry with a known FName→writer
	// mapping via candidatesFromFName. This prevents opcode-collision false positives that
	// arise when the template maps multiple writer names to the same opcode and the IDA
	// export only covers one of them.
	seen := map[string]bool{}
	for _, fname := range idaExportFunctions(opts.IDASource) {
		for _, candidate := range candidatesFromFName(fname) {
			if seen[candidate.name] {
				continue
			}
			seen[candidate.name] = true
			process(candidate.dir, candidate.name, fname, candidate.pathHint)
		}
	}

	if err := writeSummary(outDir, summary); err != nil {
		fmt.Fprintln(stderr, "summary:", err)
		return 3
	}

	switch worstVerdict {
	case diff.VerdictBlocker:
		return 1
	case diff.VerdictMinor:
		return 2
	}
	return 0
}

type candidate struct {
	name string
	dir  csvpkg.Direction
	// pathHint, when non-empty, constrains locateAtlasFile to a file whose path
	// contains this substring. Required for sub-domains whose struct names (e.g.
	// Operation, ErrorSimple, ErrorMessage) collide across packages.
	pathHint string
}

// candidatesFromFName converts an IDA function name into one or more
// likely atlas-packet writer/handler names.
func candidatesFromFName(fname string) []candidate {
	switch fname {
	case "CLogin::OnCheckPasswordResult":
		return []candidate{{name: "AuthSuccess", dir: csvpkg.DirClientbound}}
	case "CLogin::OnSelectWorldResult":
		return []candidate{{name: "CharacterList", dir: csvpkg.DirClientbound}}
	case "CLogin::OnWorldInformation":
		return []candidate{{name: "ServerListEntry", dir: csvpkg.DirClientbound}}
	case "CLogin::OnSelectCharacterResult":
		return []candidate{{name: "ServerIP", dir: csvpkg.DirClientbound}}
	case "CLogin::SendCheckPasswordPacket":
		return []candidate{
			{name: "Request", dir: csvpkg.DirServerbound},
			{name: "LoginHandle", dir: csvpkg.DirServerbound},
		}
	case "CLogin::SendSelectCharPacket":
		return []candidate{
			{name: "CharacterSelectedHandle", dir: csvpkg.DirServerbound},
			{name: "CharacterSelect", dir: csvpkg.DirServerbound},
		}
	case "CLogin::OnSetAccountResult":
		return []candidate{{name: "SetAccountResult", dir: csvpkg.DirClientbound}}
	case "CLogin::OnLatestConnectedWorld":
		return []candidate{{name: "SelectWorld", dir: csvpkg.DirClientbound}}
	case "CLogin::OnRecommendWorldMessage":
		return []candidate{{name: "ServerListRecommendations", dir: csvpkg.DirClientbound}}
	case "CLogin::SendCheckUserLimitPacket":
		return []candidate{{name: "ServerStatusRequest", dir: csvpkg.DirServerbound}}
	case "CLogin::OnCheckPinCodeResult":
		return []candidate{{name: "PinOperation", dir: csvpkg.DirClientbound}}
	case "CLogin::OnUpdatePinCodeResult":
		return []candidate{{name: "PinUpdate", dir: csvpkg.DirClientbound}}
	case "CLogin::OnAcceptLicense":
		return []candidate{{name: "AcceptTos", dir: csvpkg.DirServerbound}}
	case "CLogin::OnCheckUserLimitResult":
		return []candidate{{name: "ServerStatus", dir: csvpkg.DirClientbound}}
	case "CLogin::ChangeStepImmediate":
		return []candidate{{name: "ServerListRequest", dir: csvpkg.DirServerbound}}
	case "CLogin::SendLoginPacket":
		return []candidate{{name: "WorldCharacterListRequest", dir: csvpkg.DirServerbound}}
	case "CLogin::OnSetAccountResult#AfterLogin":
		return []candidate{{name: "AfterLogin", dir: csvpkg.DirServerbound}}
	case "CLogin::SendSelectCharPacket#CharacterSelectRegisterPic":
		return []candidate{{name: "CharacterSelectRegisterPic", dir: csvpkg.DirServerbound}}
	case "CLogin::SendSelectCharPacket#CharacterSelectWithPic":
		return []candidate{{name: "CharacterSelectWithPic", dir: csvpkg.DirServerbound}}
	case "CLogin::OnWorldInformation#ServerListEnd":
		return []candidate{{name: "ServerListEnd", dir: csvpkg.DirClientbound}}
	case "CLogin::SendSelectCharPacketByVAC#AllCharacterListSelectWithPicRegister":
		return []candidate{{name: "AllCharacterListSelectWithPicRegister", dir: csvpkg.DirServerbound}}
	case "CLogin::SendSelectCharPacketByVAC#AllCharacterListSelectWithPic":
		return []candidate{{name: "AllCharacterListSelectWithPic", dir: csvpkg.DirServerbound}}
	case "CLogin::SendSelectCharPacketByVAC#AllCharacterListSelect":
		return []candidate{{name: "AllCharacterListSelect", dir: csvpkg.DirServerbound}}
	case "CLogin::MakeVACDlg":
		return []candidate{{name: "AllCharacterListPong", dir: csvpkg.DirServerbound}}
	case "CLogin::OnCheckPasswordResult#AuthLoginFailed":
		return []candidate{{name: "AuthLoginFailed", dir: csvpkg.DirClientbound}}
	case "CLogin::OnCheckPasswordResult#AuthTemporaryBan":
		return []candidate{{name: "AuthTemporaryBan", dir: csvpkg.DirClientbound}}
	case "CLogin::OnCheckPasswordResult#AuthPermanentBan":
		return []candidate{{name: "AuthPermanentBan", dir: csvpkg.DirClientbound}}
	case "CLogin::SendViewAllCharPacket":
		return []candidate{{name: "AllCharacterListRequest", dir: csvpkg.DirServerbound}}
	// --- storage sub-domain (task-067) ---
	// Clientbound CTrunkDlg::OnPacket is a mode-dispatched writer; use synthetic
	// #-suffix FNames (one per atlas wire shape) to disambiguate.
	case "CTrunkDlg::OnPacket#Show":
		return []candidate{{name: "Show", dir: csvpkg.DirClientbound, pathHint: "storage/"}}
	case "CTrunkDlg::OnPacket#UpdateAssets":
		return []candidate{{name: "UpdateAssets", dir: csvpkg.DirClientbound, pathHint: "storage/"}}
	case "CTrunkDlg::OnPacket#UpdateMeso":
		return []candidate{{name: "UpdateMeso", dir: csvpkg.DirClientbound, pathHint: "storage/"}}
	case "CTrunkDlg::OnPacket#ErrorSimple":
		return []candidate{{name: "ErrorSimple", dir: csvpkg.DirClientbound, pathHint: "storage/"}}
	case "CTrunkDlg::OnPacket#ErrorMessage":
		return []candidate{{name: "ErrorMessage", dir: csvpkg.DirClientbound, pathHint: "storage/"}}
	// Serverbound CTrunkDlg senders.
	case "CTrunkDlg::SendGetItemRequest":
		return []candidate{{name: "OperationRetrieveAsset", dir: csvpkg.DirServerbound, pathHint: "storage/"}}
	case "CTrunkDlg::SendPutItemRequest":
		return []candidate{{name: "OperationStoreAsset", dir: csvpkg.DirServerbound, pathHint: "storage/"}}
	case "CTrunkDlg::SendGetMoneyRequest":
		return []candidate{{name: "OperationMeso", dir: csvpkg.DirServerbound, pathHint: "storage/"}}
	case "CTrunkDlg::OnPacket#Operation":
		return []candidate{{name: "Operation", dir: csvpkg.DirServerbound, pathHint: "storage/"}}
	// --- inventory sub-domain (task-067) ---
	// Clientbound CWvsContext::OnInventoryOperation is a mode-dispatched reader;
	// use synthetic #-suffix FNames (one per atlas wire shape) to disambiguate.
	case "CWvsContext::OnInventoryOperation#QuantityUpdate":
		return []candidate{{name: "QuantityUpdate", dir: csvpkg.DirClientbound, pathHint: "inventory/"}}
	case "CWvsContext::OnInventoryOperation#ChangeMove":
		return []candidate{{name: "ChangeMove", dir: csvpkg.DirClientbound, pathHint: "inventory/"}}
	case "CWvsContext::OnInventoryOperation#Remove":
		return []candidate{{name: "Remove", dir: csvpkg.DirClientbound, pathHint: "inventory/"}}
	case "CWvsContext::OnInventoryOperation#Add":
		return []candidate{{name: "Add", dir: csvpkg.DirClientbound, pathHint: "inventory/"}}
	case "CWvsContext::OnInventoryOperation#ChangeBatch":
		return []candidate{{name: "ChangeBatch", dir: csvpkg.DirClientbound, pathHint: "inventory/"}}
	case "CWvsContext::OnGatherItemResult":
		return []candidate{{name: "CompartmentMerge", dir: csvpkg.DirClientbound, pathHint: "inventory/"}}
	case "CWvsContext::OnSortItemResult":
		return []candidate{{name: "CompartmentSort", dir: csvpkg.DirClientbound, pathHint: "inventory/"}}
	// Serverbound CWvsContext senders.
	case "CWvsContext::SendChangeSlotPositionRequest":
		return []candidate{{name: "Move", dir: csvpkg.DirServerbound, pathHint: "inventory/"}}
	case "CWvsContext::SendGatherItemRequest":
		return []candidate{{name: "CompartmentMergeRequest", dir: csvpkg.DirServerbound, pathHint: "inventory/"}}
	case "CWvsContext::SendSortItemRequest":
		return []candidate{{name: "CompartmentSortRequest", dir: csvpkg.DirServerbound, pathHint: "inventory/"}}
	case "CWvsContext::SendStatChangeItemUseRequest":
		return []candidate{{name: "ItemUse", dir: csvpkg.DirServerbound, pathHint: "inventory/"}}
	case "CWvsContext::SendUpgradeItemUseRequest":
		return []candidate{{name: "ScrollUse", dir: csvpkg.DirServerbound, pathHint: "inventory/"}}
	// --- interaction sub-domain (task-067) ---
	// NOTE: the interaction serverbound dispatcher struct is also named `Operation`
	// (collides with storage's CTrunkDlg `Operation` under the flat report layout;
	// storage wins the dedup). The interaction dispatcher is documented in
	// docs/packets/ida-exports/_pending.md -> "OP-FAMILY-interaction" instead of a
	// separate report. Sub-op senders below each map 1:1 to an atlas wire shape.
	case "CMiniRoomBaseDlg::CheckAndSendChat":
		return []candidate{{name: "OperationChat", dir: csvpkg.DirServerbound, pathHint: "interaction/"}}
	case "CField::SendInviteTradingRoomMsg":
		return []candidate{{name: "OperationInvite", dir: csvpkg.DirServerbound, pathHint: "interaction/"}}
	case "CField::AddBlackList":
		return []candidate{{name: "OperationFieldAddToBlackList", dir: csvpkg.DirServerbound, pathHint: "interaction/"}}
	case "CField::DeleteBlackList":
		return []candidate{{name: "OperationFieldRemoveFromBlackList", dir: csvpkg.DirServerbound, pathHint: "interaction/"}}
	case "CPersonalShopDlg::OnClickBanButton":
		return []candidate{{name: "OperationPersonalStoreAddToBlackList", dir: csvpkg.DirServerbound, pathHint: "interaction/"}}
	case "CPersonalShopDlg::DeliverBlackList":
		return []candidate{{name: "OperationPersonalStoreSetBlackList", dir: csvpkg.DirServerbound, pathHint: "interaction/"}}
	case "CTradingRoomDlg::PutItem":
		return []candidate{{name: "OperationTradePutItem", dir: csvpkg.DirServerbound, pathHint: "interaction/"}}
	case "CTradingRoomDlg::PutMoney":
		return []candidate{{name: "OperationTradeAddMeso", dir: csvpkg.DirServerbound, pathHint: "interaction/"}}
	case "CTradingRoomDlg::Trade":
		return []candidate{{name: "OperationTradeConfirm", dir: csvpkg.DirServerbound, pathHint: "interaction/"}}
	case "CCashTradingRoomDlg::Trade":
		return []candidate{{name: "OperationTransaction", dir: csvpkg.DirServerbound, pathHint: "interaction/"}}
	case "CPersonalShopDlg::PutItem":
		return []candidate{{name: "OperationPersonalStorePutItem", dir: csvpkg.DirServerbound, pathHint: "interaction/"}}
	case "CPersonalShopDlg::BuyItem":
		return []candidate{{name: "OperationPersonalStoreBuy", dir: csvpkg.DirServerbound, pathHint: "interaction/"}}
	case "CPersonalShopDlg::MoveItemToInventory":
		return []candidate{{name: "OperationPersonalStoreRemoveItem", dir: csvpkg.DirServerbound, pathHint: "interaction/"}}
	case "CMemoryGameDlg::SendTurnUpCard":
		return []candidate{{name: "OperationMemoryGameFlipCard", dir: csvpkg.DirServerbound, pathHint: "interaction/"}}
	case "CMemoryGameDlg::OnTieRequest":
		return []candidate{{name: "OperationMemoryGameTieAnswer", dir: csvpkg.DirServerbound, pathHint: "interaction/"}}
	case "COmokDlg::PutStoneChecker":
		return []candidate{{name: "OperationMemoryGameMoveStone", dir: csvpkg.DirServerbound, pathHint: "interaction/"}}
	case "COmokDlg::OnRetreatRequest":
		return []candidate{{name: "OperationMemoryGameRetreatAnswer", dir: csvpkg.DirServerbound, pathHint: "interaction/"}}
	// Entrusted-merchant sub-ops (share CPersonalShopDlg senders w/ different op-bytes).
	case "CEntrustedShopDlg::AddBlackList":
		return []candidate{{name: "OperationMerchantAddToBlackList", dir: csvpkg.DirServerbound, pathHint: "interaction/"}}
	case "CEntrustedShopDlg::DeleteBlackList":
		return []candidate{{name: "OperationMerchantRemoveFromBlackList", dir: csvpkg.DirServerbound, pathHint: "interaction/"}}
	case "CPersonalShopDlg::PutItem#Merchant":
		return []candidate{{name: "OperationMerchantPutItem", dir: csvpkg.DirServerbound, pathHint: "interaction/"}}
	case "CPersonalShopDlg::BuyItem#Merchant":
		return []candidate{{name: "OperationMerchantBuy", dir: csvpkg.DirServerbound, pathHint: "interaction/"}}
	case "CPersonalShopDlg::MoveItemToInventory#Merchant":
		return []candidate{{name: "OperationMerchantRemoveItem", dir: csvpkg.DirServerbound, pathHint: "interaction/"}}
	// Clientbound CMiniRoomBaseDlg::OnPacketBase is a mode-dispatched reader;
	// synthetic #-suffix FNames disambiguate the per-mode atlas wire shapes.
	case "CMiniRoomBaseDlg::OnPacketBase#Invite":
		return []candidate{{name: "InteractionInvite", dir: csvpkg.DirClientbound, pathHint: "interaction/"}}
	case "CMiniRoomBaseDlg::OnPacketBase#InviteResult":
		return []candidate{{name: "InteractionInviteResult", dir: csvpkg.DirClientbound, pathHint: "interaction/"}}
	case "CMiniRoomBaseDlg::OnPacketBase#Enter":
		return []candidate{{name: "InteractionEnter", dir: csvpkg.DirClientbound, pathHint: "interaction/"}}
	case "CMiniRoomBaseDlg::OnPacketBase#EnterResultSuccess":
		return []candidate{{name: "InteractionEnterResultSuccess", dir: csvpkg.DirClientbound, pathHint: "interaction/"}}
	case "CMiniRoomBaseDlg::OnPacketBase#EnterResultError":
		return []candidate{{name: "InteractionEnterResultError", dir: csvpkg.DirClientbound, pathHint: "interaction/"}}
	case "CMiniRoomBaseDlg::OnPacketBase#Chat":
		return []candidate{{name: "InteractionChat", dir: csvpkg.DirClientbound, pathHint: "interaction/"}}
	case "CMiniRoomBaseDlg::OnPacketBase#Leave":
		return []candidate{{name: "InteractionLeave", dir: csvpkg.DirClientbound, pathHint: "interaction/"}}
	case "CPersonalShopDlg::OnRefresh#UpdateMerchant":
		return []candidate{{name: "InteractionUpdateMerchant", dir: csvpkg.DirClientbound, pathHint: "interaction/"}}
	// --- cash sub-domain (task-067, Phase 1d) ---
	// Clientbound: QueryResult routes through CCashShop::OnQueryCashResult (opcode 0x17F),
	// a SEPARATE dispatcher from OnCashItemResult.
	case "CCashShop::OnQueryCashResult":
		return []candidate{{name: "QueryResult", dir: csvpkg.DirClientbound, pathHint: "cash/"}}
	// Clientbound CCashShop::OnCashItemResult is a mode-dispatched reader (op-bytes 0x54-0xBC);
	// synthetic #-suffix FNames map each CashShopOperation result struct to its OnCashItemRes* sub-handler.
	case "CCashShop::OnCashItemResult#CashShopInventory":
		return []candidate{{name: "CashShopInventory", dir: csvpkg.DirClientbound, pathHint: "cash/"}}
	case "CCashShop::OnCashItemResult#WishList":
		return []candidate{{name: "WishList", dir: csvpkg.DirClientbound, pathHint: "cash/"}}
	case "CCashShop::OnCashItemResult#InventoryCapacitySuccess":
		return []candidate{{name: "InventoryCapacitySuccess", dir: csvpkg.DirClientbound, pathHint: "cash/"}}
	case "CCashShop::OnCashItemResult#InventoryCapacityFailed":
		return []candidate{{name: "InventoryCapacityFailed", dir: csvpkg.DirClientbound, pathHint: "cash/"}}
	case "CCashShop::OnCashItemResult#OperationError":
		return []candidate{{name: "OperationError", dir: csvpkg.DirClientbound, pathHint: "cash/"}}
	case "CCashShop::OnCashItemResult#CashShopPurchaseSuccess":
		return []candidate{{name: "CashShopPurchaseSuccess", dir: csvpkg.DirClientbound, pathHint: "cash/"}}
	case "CCashShop::OnCashItemResult#CashItemMovedToCashInventory":
		return []candidate{{name: "CashItemMovedToCashInventory", dir: csvpkg.DirClientbound, pathHint: "cash/"}}
	// Serverbound CCashShop senders (op-byte owned by the ShopOperation dispatcher; bodies below).
	case "CCashShop::TrySendQueryCashRequest":
		return []candidate{{name: "CheckWallet", dir: csvpkg.DirServerbound, pathHint: "cash/"}}
	case "CCashShop::OnBuy":
		return []candidate{{name: "ShopOperationBuy", dir: csvpkg.DirServerbound, pathHint: "cash/"}}
	case "CCashShop::OnBuyNormal":
		return []candidate{{name: "ShopOperationBuyNormal", dir: csvpkg.DirServerbound, pathHint: "cash/"}}
	case "CCashShop::OnBuyPackage":
		return []candidate{{name: "ShopOperationBuyPackage", dir: csvpkg.DirServerbound, pathHint: "cash/"}}
	case "CCashShop::OnBuyCouple":
		return []candidate{{name: "ShopOperationBuyCouple", dir: csvpkg.DirServerbound, pathHint: "cash/"}}
	case "CCashShop::OnBuyFriendship":
		return []candidate{{name: "ShopOperationBuyFriendship", dir: csvpkg.DirServerbound, pathHint: "cash/"}}
	case "CCashShop::SendBuyNameChangeItemPacket":
		return []candidate{{name: "ShopOperationBuyNameChange", dir: csvpkg.DirServerbound, pathHint: "cash/"}}
	case "CCashShop::SendBuyTransferWorldItemPacket":
		return []candidate{{name: "ShopOperationBuyWorldTransfer", dir: csvpkg.DirServerbound, pathHint: "cash/"}}
	case "CCashShop::OnEnableEquipSlotExt":
		return []candidate{{name: "ShopOperationEnableEquipSlot", dir: csvpkg.DirServerbound, pathHint: "cash/"}}
	case "CCashShop::RequestCashPurchaseRecord":
		return []candidate{{name: "ShopOperationGetPurchaseRecord", dir: csvpkg.DirServerbound, pathHint: "cash/"}}
	case "CCashShop::SendGiftsPacket":
		return []candidate{{name: "ShopOperationGift", dir: csvpkg.DirServerbound, pathHint: "cash/"}}
	case "CCashShop::OnIncCharacterSlotCount":
		return []candidate{{name: "ShopOperationIncreaseCharacterSlot", dir: csvpkg.DirServerbound, pathHint: "cash/"}}
	case "CCashShop::OnBuySlotInc":
		return []candidate{{name: "ShopOperationIncreaseInventory", dir: csvpkg.DirServerbound, pathHint: "cash/"}}
	case "CCashShop::OnIncTrunkCount":
		return []candidate{{name: "ShopOperationIncreaseStorage", dir: csvpkg.DirServerbound, pathHint: "cash/"}}
	case "CCashShop::OnMoveCashItemLtoS":
		return []candidate{{name: "ShopOperationMoveFromCashInventory", dir: csvpkg.DirServerbound, pathHint: "cash/"}}
	case "CCashShop::OnMoveCashItemStoL":
		return []candidate{{name: "ShopOperationMoveToCashInventory", dir: csvpkg.DirServerbound, pathHint: "cash/"}}
	case "CCashShop::OnRebateLockerItem":
		return []candidate{{name: "ShopOperationRebateLockerItem", dir: csvpkg.DirServerbound, pathHint: "cash/"}}
	case "CCashShop::OnSetWish":
		return []candidate{{name: "ShopOperationSetWishlist", dir: csvpkg.DirServerbound, pathHint: "cash/"}}
	}
	return nil
}

// idaExportFunctions returns the list of FNames in an export JSON file.
// Returns nil if the IDASource is "mcp" or empty.
func idaExportFunctions(idaSource string) []string {
	if idaSource == "mcp" || idaSource == "" {
		return nil
	}
	src, err := idasrc.NewExportSource(idaSource)
	if err != nil {
		return nil
	}
	return src.Functions()
}

func openIDASource(s string) (idasrc.Source, error) {
	if s == "mcp" {
		return idasrc.NewMCPSource(nil), nil
	}
	return idasrc.NewExportSource(s)
}

func methodName(_ csvpkg.Direction) string {
	// Both clientbound and serverbound atlas types expose an Encode method as the
	// canonical wire serializer. Analyze Encode for both directions — the Decode
	// dispatcher methods often delegate to helper sub-methods that AnalyzeFile
	// cannot descend into from the top-level Decode body.
	return "Encode"
}

// lookupFName maps an atlas writer/handler name back to the IDA FName via the CSV.
func lookupFName(name string, dir csvpkg.Direction, cb, sb csvpkg.Map, template *tpl.Template) (string, bool) {
	var (
		opcode int
		ok     bool
		source csvpkg.Map
	)
	if dir == csvpkg.DirClientbound {
		for op, w := range template.Writers() {
			if w == name {
				opcode, ok = op, true
				break
			}
		}
		source = cb
	} else {
		for op, h := range template.Handlers() {
			if h == name {
				opcode, ok = op, true
				break
			}
		}
		source = sb
	}
	if !ok {
		return "", false
	}
	for _, row := range source.All() {
		if row.Opcode(template.Region, template.MajorVersion) == opcode {
			return row.FName, true
		}
	}
	return "", false
}

func locateAtlasFile(root, name string, dir csvpkg.Direction, pathHint string) (string, bool) {
	needle := "type " + name + " struct"
	var hit string
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		// Match on direction: clientbound vs serverbound folders
		expectDir := "clientbound"
		if dir == csvpkg.DirServerbound {
			expectDir = "serverbound"
		}
		if !strings.Contains(path, string(os.PathSeparator)+expectDir+string(os.PathSeparator)) {
			return nil
		}
		// pathHint disambiguates struct-name collisions across packages (e.g. the
		// generic Operation/ErrorSimple/ErrorMessage names used by several domains).
		if pathHint != "" && !strings.Contains(path, pathHint) {
			return nil
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		if strings.Contains(string(b), needle) {
			hit = path
			return filepath.SkipAll
		}
		return nil
	})
	return hit, hit != ""
}

func branchDepth(calls []atlaspacket.Call) int {
	maxd := 0
	for _, c := range calls {
		if c.Guard == nil {
			continue
		}
		d := strings.Count(c.Guard.String(), "&&") + 1
		if d > maxd {
			maxd = d
		}
	}
	return maxd
}

func worstRow(rows []diff.Row) diff.Verdict {
	w := diff.VerdictMatch
	for _, r := range rows {
		if r.Verdict > w {
			w = r.Verdict
		}
	}
	return w
}

func writeSummary(outDir string, summary []report.Packet) error {
	var b strings.Builder
	b.WriteString("# Audit summary\n\n")
	b.WriteString("| Packet | Verdict | Atlas file |\n|---|---|---|\n")
	for _, p := range summary {
		fmt.Fprintf(&b, "| [%s](%s.md) | %s | `%s` |\n", p.WriterName, p.WriterName, p.Verdict.Symbol(), p.AtlasFile)
	}
	return os.WriteFile(filepath.Join(outDir, "SUMMARY.md"), []byte(b.String()), 0o644)
}
