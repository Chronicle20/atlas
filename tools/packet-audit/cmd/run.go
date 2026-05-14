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

	process := func(direction csvpkg.Direction, name string, fname string) {
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
		atlasPath, found := locateAtlasFile(opts.AtlasPacket, name, direction)
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
			process(candidate.dir, candidate.name, fname)
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
}

// candidatesFromFName converts an IDA function name into one or more
// likely atlas-packet writer/handler names.
func candidatesFromFName(fname string) []candidate {
	switch fname {
	// --- Character domain ---
	case "CUserPool::OnUserEnterField":
		return []candidate{{name: "CharacterSpawn", dir: csvpkg.DirClientbound}}
	case "CUserRemote::OnAttack":
		// The atlas struct is Attack (shared for all 4 attack types); analyse
		// under CharacterAttackMelee so the report file has a descriptive name.
		return []candidate{{name: "Attack", dir: csvpkg.DirClientbound}}
	case "CUserRemote::OnHit":
		return []candidate{{name: "CharacterDamage", dir: csvpkg.DirClientbound}}
	case "CWvsContext::OnTemporaryStatSet":
		// Atlas struct is BuffGive (self-buff); foreign variant is BuffGiveForeign.
		return []candidate{{name: "BuffGive", dir: csvpkg.DirClientbound}}
	case "CUserRemote::OnSetTemporaryStat":
		return []candidate{{name: "BuffGiveForeign", dir: csvpkg.DirClientbound}}
	case "CUserRemote::OnMove":
		return []candidate{{name: "CharacterMovement", dir: csvpkg.DirClientbound}}
	case "CWvsContext::OnChangeSkillRecordResult":
		return []candidate{{name: "CharacterSkillChange", dir: csvpkg.DirClientbound}}
	case "CWvsContext::OnTemporaryStatReset":
		// Self buff-cancel: mask-only packet (no characterId prefix).
		// Struct is BuffCancel; writer constant = "CharacterBuffCancel".
		return []candidate{{name: "BuffCancel", dir: csvpkg.DirClientbound}}
	case "CUserRemote::OnResetTemporaryStat":
		// Foreign buff-cancel: characterId prefix + mask.
		// Struct is BuffCancelForeign; writer constant = "CharacterBuffCancelForeign".
		return []candidate{{name: "BuffCancelForeign", dir: csvpkg.DirClientbound}}
	case "CUser::OnEffect":
		// Effect family: dispatches on a leading sub-op byte (mode).
		// The pipeline can only model the outermost Decode1; sub-op enum drift
		// is deferred to _pending.md.  Generate one representative report per
		// atlas effect file so the SUMMARY covers all 3 effect files.
		return []candidate{
			{name: "EffectSimple", dir: csvpkg.DirClientbound},       // effect.go
			{name: "EffectQuest", dir: csvpkg.DirClientbound},        // effect_quest.go
			{name: "EffectSkillUse", dir: csvpkg.DirClientbound},     // effect_skill_use.go
		}
	case "CUserLocal::OnSkillCooltimeSet":
		// Struct is CharacterSkillCooldown.
		return []candidate{{name: "CharacterSkillCooldown", dir: csvpkg.DirClientbound}}
	case "CUserRemote::OnAvatarModified":
		// Struct is CharacterAppearanceUpdate.
		return []candidate{{name: "CharacterAppearanceUpdate", dir: csvpkg.DirClientbound}}
	// --- Character misc-state bucket ---
	case "CUserRemote::OnSetActivePortableChair":
		// Struct is CharacterChairShow; writer = "CharacterShowChair".
		// CUserPool::OnUserRemotePacket (case 222 = 0xDE) reads characterId, then
		// delegates remaining packet to OnSetActivePortableChair which reads Decode4 (chairId).
		return []candidate{{name: "CharacterChairShow", dir: csvpkg.DirClientbound}}
	case "CUser::OnADBoard":
		// Struct is ChalkboardUse; writer = "ChalkboardUse".
		// CUserPool::OnUserCommonPacket (case 183 = 0xB7) reads characterId, then
		// delegates remaining packet to OnADBoard which reads Decode1 (active) + optional DecodeStr.
		return []candidate{{name: "ChalkboardUse", dir: csvpkg.DirClientbound}}
	case "CUser::OnEmotion":
		// Struct is CharacterExpression; writer = "CharacterExpression".
		// CUserPool::OnUserRemotePacket (case 219 = 0xDB) reads characterId, then
		// delegates remaining packet to OnEmotion which reads Decode4 (expressionId) +
		// Decode4 (duration) + Decode1 (itemOptionFlag).
		// The local-player variant (case 232 = 0xE8) goes through CUserLocal::OnPacket
		// and has no characterId prefix.
		return []candidate{{name: "CharacterExpression", dir: csvpkg.DirClientbound}}
	case "CUserLocal::OnBalloonMsg":
		// Struct is CharacterHint; writer = "CharacterHint".
		// CUserLocal::OnPacket (case 245 = 0xF5) delegates directly (no characterId prefix).
		// Reads: DecodeStr (hint) + Decode2 (width) + Decode2 (height) + Decode1 (notAtPoint flag)
		// + if !notAtPoint: Decode4 (x) + Decode4 (y).
		return []candidate{{name: "CharacterHint", dir: csvpkg.DirClientbound}}
	case "CWvsContext::OnCharacterInfo":
		// Struct is CharacterInfo; writer = "CharacterInfo".
		// CWvsContext::OnPacket (case 61 = 0x3D) delegates directly.
		return []candidate{{name: "CharacterInfo", dir: csvpkg.DirClientbound}}
	case "CUserLocal::OnSitResult":
		// Struct is CharacterSitResult; writer = "CharacterSitResult".
		// CUserLocal::OnPacket (case 231 = 0xE7) delegates directly (no characterId prefix).
		// Reads: Decode1 (sitting flag); if 1: Decode2 (chairId).
		return []candidate{{name: "CharacterSitResult", dir: csvpkg.DirClientbound}}
	// --- Character tail bucket ---
	case "CLogin::OnDeleteCharacterResult":
		// Struct is DeleteCharacterResponse; writer = "DeleteCharacterResponse".
		// CLogin::OnDeleteCharacterResult (case 0x0F in login socket) reads
		// Decode4 (characterId) + Decode1 (result code).
		return []candidate{{name: "DeleteCharacterResponse", dir: csvpkg.DirClientbound}}
	case "CWvsContext::OnMessage":
		// Struct family is StatusMessage*; writer = "CharacterStatusMessage".
		// CWvsContext::OnPacket case 38 (0x26) delegates here; dispatches on
		// a leading mode byte (0-14) to 15 sub-handlers.  The pipeline can only
		// model the outermost Decode1 (mode byte); sub-op enum drift is deferred
		// to _pending.md "## Sub-op enum drift — character domain".
		return []candidate{{name: "StatusMessageDropPickUpInventoryFull", dir: csvpkg.DirClientbound}}
	case "CUser::ShowItemUpgradeEffect":
		// Struct is ItemUpgrade; writer = "CharacterItemUpgrade".
		// CUserPool::OnUserCommonPacket case 186 (0xBA) reads Decode4 (characterId)
		// then delegates to this function which reads 3 × Decode1 + Decode4 + 2 × Decode1.
		return []candidate{{name: "ItemUpgrade", dir: csvpkg.DirClientbound}}
	case "CFuncKeyMappedMan::OnInit":
		// Struct is CharacterKeyMap; writer = "CharacterKeyMap".
		// CFuncKeyMappedMan::OnPacket case 0x18E delegates here.
		// Reads: Decode1 (resetToDefault) + 90 × (Decode1 keyType + Decode4 keyAction).
		return []candidate{{name: "CharacterKeyMap", dir: csvpkg.DirClientbound}}
	case "CFuncKeyMappedMan::OnPetConsumeItemInit":
		// Struct is CharacterKeyMapAutoHp; writer = "CharacterKeyMapAutoHp".
		// CFuncKeyMappedMan::OnPacket case 0x18F delegates here.
		// Reads: Decode4 (HP auto-pot item ID).
		return []candidate{{name: "CharacterKeyMapAutoHp", dir: csvpkg.DirClientbound}}
	case "CFuncKeyMappedMan::OnPetConsumeMPItemInit":
		// Struct is CharacterKeyMapAutoMp; writer = "CharacterKeyMapAutoMp".
		// CFuncKeyMappedMan::OnPacket case 0x190 delegates here.
		// Reads: Decode4 (MP auto-pot item ID).
		return []candidate{{name: "CharacterKeyMapAutoMp", dir: csvpkg.DirClientbound}}
	// --- Login domain ---
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
	// --- Character spawn/list bucket ---
	case "CLogin::OnViewAllCharResult#CharacterViewAllCount":
		return []candidate{{name: "CharacterViewAllCount", dir: csvpkg.DirClientbound}}
	case "CLogin::OnViewAllCharResult#CharacterViewAllCharacters":
		return []candidate{{name: "CharacterViewAllCharacters", dir: csvpkg.DirClientbound}}
	case "CLogin::OnViewAllCharResult#CharacterViewAllSearchFailed":
		// CharacterViewAllSearchFailed and CharacterViewAllError both encode only
		// a single code byte; model both from the same dispatcher sub-path.
		return []candidate{
			{name: "CharacterViewAllSearchFailed", dir: csvpkg.DirClientbound},
			{name: "CharacterViewAllError", dir: csvpkg.DirClientbound},
		}
	case "CLogin::OnCreateNewCharacterResult":
		return []candidate{{name: "AddCharacterEntry", dir: csvpkg.DirClientbound}}
	case "CLogin::OnCreateNewCharacterResult#AddCharacterError":
		return []candidate{{name: "AddCharacterError", dir: csvpkg.DirClientbound}}
	case "CUserPool::OnUserLeaveField":
		return []candidate{{name: "CharacterDespawn", dir: csvpkg.DirClientbound}}
	case "CLogin::OnCheckDuplicatedIDResult":
		return []candidate{{name: "CharacterNameResponse", dir: csvpkg.DirClientbound}}
	// --- Character serverbound hot bucket (Task 12) ---
	case "CVecCtrlUser::EndUpdateActive":
		// Struct is Move; handler constant = "CharacterMoveHandle".
		// Client builds opcode 0x2C (44) packet in EndUpdateActive, writes
		// dr0/dr1 (GMS>83), fieldKey, dr2/dr3 (GMS>83), crc (GMS>28), dwKey/crc32 (GMS>83)
		// then delegates movement encoding to CMovePath::Encode/Flush (DecodeLoop).
		return []candidate{{name: "Move", dir: csvpkg.DirServerbound}}
	case "CWvsContext::SendStatChangeRequest":
		// Struct is HealOverTime; handler constant = "CharacterHealOverTimeHandle".
		// Client sends opcode 0x64 (100) with Encode4(updateTime)+Encode4(val)+
		// Encode2(hp)+Encode2(mp)+Encode1(option).
		return []candidate{{name: "HealOverTime", dir: csvpkg.DirServerbound}}
	case "CWvsContext::SendCharacterInfoRequest":
		// Struct is InfoRequest; handler constant = "CharacterInfoRequestHandle".
		// Client sends opcode 0x6D (109) with Encode4(updateTime)+Encode4(characterId)+
		// Encode1(bPetInfo).
		return []candidate{{name: "InfoRequest", dir: csvpkg.DirServerbound}}
	case "CUserLocal::SendSkillCancelRequest":
		// Struct is BuffCancelRequest; handler constant = "CharacterBuffCancel".
		// Client sends opcode 0x68 (104) with Encode4(nSkillID).
		return []candidate{{name: "BuffCancelRequest", dir: csvpkg.DirServerbound}}
	case "CWvsContext::SendStatChangeItemCancelRequest":
		// Struct is ItemCancel; handler constant = "CharacterItemCancelHandle".
		// Client sends opcode 0x4F (79) with Encode4(nItemID).
		return []candidate{{name: "ItemCancel", dir: csvpkg.DirServerbound}}
	// --- Character serverbound chairs/expression bucket (Task 13) ---
	case "CUserLocal::HandleXKeyDown":
		// Struct is ChairFixed; handler constant = "CharacterChairInteractionHandle".
		// Client sends opcode 0x2D (45) with Encode2(chairId).
		// chairId is the seat index from CField::FindSeatByPosition; 0xFFFF (-1) = get-up-from-chair.
		// SendGetUpFromChairRequest (CWvsContext) is a second codepath for the same opcode
		// that always sends 0xFFFF; both paths share this struct.
		return []candidate{{name: "ChairFixed", dir: csvpkg.DirServerbound}}
	case "CWvsContext::SendSitOnPortableChairRequest":
		// Struct is ChairPortable; handler constant = "CharacterChairPortableHandle".
		// Client sends opcode 0x2E (46) with Encode4(nItemID).
		return []candidate{{name: "ChairPortable", dir: csvpkg.DirServerbound}}
	case "CUserLocal::HandleLButtonClk":
		// Struct is ChalkboardClose; handler constant = "ChalkboardCloseHandle".
		// Client sends opcode 0x37 (55) with no payload (empty body).
		// Triggered when CChatBalloon::ADBoardMouseUp returns true (user closes chalkboard).
		return []candidate{{name: "ChalkboardClose", dir: csvpkg.DirServerbound}}
	case "CWvsContext::SendEmotionChange":
		// Struct is ExpressionRequest; handler constant = "CharacterExpressionHandle".
		// Client sends opcode 0x38 (56) with Encode4(emotion) + Encode4(nDuration) + Encode1(bByItemOption).
		// Emotion validated <= 0x17; cooldown 2 s between sends.
		return []candidate{{name: "ExpressionRequest", dir: csvpkg.DirServerbound}}
	case "CWvsContext::SendDropMoneyRequest":
		// Struct is DropMeso; handler constant = "CharacterDropMesoHandle".
		// Client sends opcode 0x6A (106) with Encode4(update_time) + Encode4(nAmount).
		return []candidate{{name: "DropMeso", dir: csvpkg.DirServerbound}}
	case "CFuncKeyMappedMan::SaveFuncKeyMap":
		// Struct is KeyMapChange (mode=0); handler constant = "CharacterKeyMapChangeHandle".
		// Client sends opcode 0x9F (159) with Encode4(0) + Encode4(count) +
		// for each changed slot: Encode4(keySlotIdx) + FUNCKEY_MAPPED::Encode() (nType:byte + nID:int32).
		// Per-entry layout = Encode4(keyId) + Encode1(theType) + Encode4(action) = 9 bytes.
		return []candidate{{name: "KeyMapChange", dir: csvpkg.DirServerbound}}
	case "CFuncKeyMappedMan::ChangePetConsumeItemID":
		// Same opcode 0x9F (159) as SaveFuncKeyMap but mode=1.
		// Client sends Encode4(1) + Encode4(nPetConsumeItemID).
		// Covered by KeyMapChange (mode != 0 branch); skip to avoid duplicate report.
		return nil
	case "CFuncKeyMappedMan::ChangePetConsumeMPItemID":
		// Same opcode 0x9F (159) as SaveFuncKeyMap but mode=2.
		// Client sends Encode4(2) + Encode4(nPetConsumeMPItemID).
		// Covered by KeyMapChange (mode != 0 branch); skip to avoid duplicate report.
		return nil
	// --- Character serverbound lifecycle bucket (Task 14) ---
	case "CWvsContext::SendAbilityUpRequest#DistributeAp":
		// Struct is DistributeAp; handler constant = "CharacterDistributeApHandle".
		// Client sends opcode 0x62 (98) with Encode4(update_time) + Encode4(dwFlag).
		// IDA: CWvsContext::SendAbilityUpRequest(DWORD)@0x9f61c0.
		return []candidate{{name: "DistributeAp", dir: csvpkg.DirServerbound}}
	case "CWvsContext::SendAbilityUpRequest#AutoDistributeAp":
		// Struct is AutoDistributeAp; handler constant = "CharacterAutoDistributeApHandle".
		// Client sends opcode 0x63 (99) with Encode4(update_time) + Encode4(count) +
		// [Encode4(flag) + Encode4(value)] * count.
		// IDA: CWvsContext::SendAbilityUpRequest(ZArray<StatPair>*)@0x9f63b0.
		return []candidate{{name: "AutoDistributeAp", dir: csvpkg.DirServerbound}}
	case "CWvsContext::SendSkillUpRequest":
		// Struct is DistributeSp; handler constant = "CharacterDistributeSpHandle".
		// Client sends opcode 0x66 (102) with Encode4(update_time) + Encode4(nSkillID).
		// IDA: CWvsContext::SendSkillUpRequest@0x9f2e90.
		return []candidate{{name: "DistributeSp", dir: csvpkg.DirServerbound}}
	case "CLogin::SendCheckDuplicateIDPacket":
		// Struct is CheckName; handler constant = "CharacterCheckNameHandle".
		// Client sends opcode 0x15 (21) with EncodeStr(name).
		// IDA: CLogin::SendCheckDuplicateIDPacket@0x5d5690.
		return []candidate{{name: "CheckName", dir: csvpkg.DirServerbound}}
	case "CLogin::SendNewCharPacket":
		// Struct is CreateCharacter; handler constant = "CreateCharacterHandle".
		// Client sends opcode 0x16 (22) with EncodeStr(name)+Encode4(race)+Encode2(subJob)+
		// Encode4×8(face/hair/hairColor/skinColor/top/bot/shoes/weapon)+Encode1(gender).
		// IDA: CLogin::SendNewCharPacket@0x5d7bd0 (normal path; bCharSale=false).
		return []candidate{{name: "CreateCharacter", dir: csvpkg.DirServerbound}}
	case "CLogin::SendDeleteCharPacket":
		// Struct is DeleteCharacter; handler constant = "DeleteCharacterHandle".
		// Client sends opcode 0x18 (24) with EncodeStr(pic)+Encode4(charId) (v95: PIC path).
		// IDA: CLogin::SendDeleteCharPacket@0x5d53a0 (m_bLoginOpt==1 branch).
		return []candidate{{name: "DeleteCharacter", dir: csvpkg.DirServerbound}}
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

func locateAtlasFile(root, name string, dir csvpkg.Direction) (string, bool) {
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
