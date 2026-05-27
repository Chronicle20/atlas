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

	process := func(direction csvpkg.Direction, name, pkg, fname string) {
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
		atlasPath, found := locateAtlasFile(opts.AtlasPacket, name, pkg, direction)
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
		writerName := qualifiedWriterName(pkg, name)
		pkt := report.Packet{
			WriterName:  writerName,
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
			fmt.Fprintln(stderr, "write", writerName+":", err)
		}
	}

	// Only audit packets that have an explicit IDA export entry with a known FName→writer
	// mapping via candidatesFromFName. This prevents opcode-collision false positives that
	// arise when the template maps multiple writer names to the same opcode and the IDA
	// export only covers one of them.
	seen := map[string]bool{}
	for _, fname := range idaExportFunctions(opts.IDASource) {
		for _, candidate := range candidatesFromFName(fname) {
			key := candidate.pkg + "::" + candidate.name
			if seen[key] {
				continue
			}
			seen[key] = true
			process(candidate.dir, candidate.name, candidate.pkg, fname)
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
	// pkg is an optional sub-domain folder hint (e.g. "monster", "drop",
	// "reactor", "pet"). When set, locateAtlasFile restricts the walk to
	// libs/atlas-packet/<pkg>/<direction>/ and the report filename becomes
	// titlecase(pkg)+name (e.g. MonsterSpawn.md), disambiguating short
	// struct names that collide across sub-domains.
	pkg string
	dir csvpkg.Direction
}

// qualifiedWriterName returns the report/file name for a candidate. When pkg
// is empty (the existing login/character routing), the writer name is just
// the struct name. When pkg is set, the writer name is titlecase(pkg)+name
// so each sub-domain's short-named structs get unique report files.
func qualifiedWriterName(pkg, name string) string {
	if pkg == "" {
		return name
	}
	return strings.ToUpper(pkg[:1]) + pkg[1:] + name
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
			{name: "EffectSimple", dir: csvpkg.DirClientbound},   // effect.go
			{name: "EffectQuest", dir: csvpkg.DirClientbound},    // effect_quest.go
			{name: "EffectSkillUse", dir: csvpkg.DirClientbound}, // effect_skill_use.go
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

	// --- Combat: monster (clientbound) ---
	// FNames verified against the canonical CSV (docs/packets/MapleStory Ops -
	// ClientBound.csv) and live GMS v95 IDA. CMobPool::OnMobPacket dispatches
	// per-mob ops to CMob::OnXxx leaf handlers; we route each leaf directly.
	case "CMobPool::OnMobEnterField":
		return []candidate{{name: "Spawn", pkg: "monster", dir: csvpkg.DirClientbound}}
	case "CMobPool::OnMobLeaveField":
		return []candidate{{name: "Destroy", pkg: "monster", dir: csvpkg.DirClientbound}}
	case "CMobPool::OnMobChangeController":
		return []candidate{{name: "Control", pkg: "monster", dir: csvpkg.DirClientbound}}
	case "CMob::OnMove":
		return []candidate{{name: "Movement", pkg: "monster", dir: csvpkg.DirClientbound}}
	case "CMob::OnCtrlAck":
		return []candidate{{name: "MovementAck", pkg: "monster", dir: csvpkg.DirClientbound}}
	case "CMob::OnStatSet":
		return []candidate{{name: "StatSet", pkg: "monster", dir: csvpkg.DirClientbound}}
	case "CMob::OnStatReset":
		return []candidate{{name: "StatReset", pkg: "monster", dir: csvpkg.DirClientbound}}
	case "CMob::OnDamaged":
		return []candidate{{name: "Damage", pkg: "monster", dir: csvpkg.DirClientbound}}
	case "CMob::OnHPIndicator":
		return []candidate{{name: "Health", pkg: "monster", dir: csvpkg.DirClientbound}}

	// --- Combat: drop (clientbound) ---
	case "CDropPool::OnDropEnterField":
		return []candidate{{name: "Spawn", pkg: "drop", dir: csvpkg.DirClientbound}}
	case "CDropPool::OnDropLeaveField":
		return []candidate{{name: "Destroy", pkg: "drop", dir: csvpkg.DirClientbound}}

	// --- Combat: reactor (clientbound) ---
	case "CReactorPool::OnReactorEnterField":
		return []candidate{{name: "Spawn", pkg: "reactor", dir: csvpkg.DirClientbound}}
	case "CReactorPool::OnReactorChangeState":
		// CSV: REACTOR_HIT — atlas Hit (writer = "ReactorHit").
		return []candidate{{name: "Hit", pkg: "reactor", dir: csvpkg.DirClientbound}}
	case "CReactorPool::OnReactorLeaveField":
		return []candidate{{name: "Destroy", pkg: "reactor", dir: csvpkg.DirClientbound}}

	// --- Combat: pet (clientbound) ---
	// CSV maps SPAWN_PET → CUser::OnPetPacket (the dispatcher for self/foreign
	// pet activation). Atlas's Activated struct writes `ownerId` + slot + active —
	// the ownerId is the characterId consumed by CUserPool::OnUserRemotePacket
	// before dispatch to CUserRemote::OnPetActivated, so we route to the foreign
	// leaf. Verified against v95 IDA OnPetPacket@0x8e02a0 (dispatcher) and
	// CUserRemote::OnPetActivated@0x9547d0 (leaf).
	case "CUserRemote::OnPetActivated":
		return []candidate{{name: "Activated", pkg: "pet", dir: csvpkg.DirClientbound}}
	case "CPet::OnMove":
		return []candidate{{name: "Movement", pkg: "pet", dir: csvpkg.DirClientbound}}
	case "CPet::OnAction":
		return []candidate{{name: "Chat", pkg: "pet", dir: csvpkg.DirClientbound}}
	case "CPet::OnActionCommand":
		return []candidate{{name: "CommandResponse", pkg: "pet", dir: csvpkg.DirClientbound}}
	case "CPet::OnLoadExceptionList":
		return []candidate{{name: "ExcludeResponse", pkg: "pet", dir: csvpkg.DirClientbound}}
	case "CWvsContext::OnCashPetFoodResult":
		return []candidate{{name: "CashFoodResult", pkg: "pet", dir: csvpkg.DirClientbound}}

	// --- Combat: monster (serverbound) ---
	case "CMob::GenerateMovePath":
		// CSV: MOVE_LIFE — atlas MovementRequest (handle = "MonsterMovementHandle").
		return []candidate{{name: "MovementRequest", pkg: "monster", dir: csvpkg.DirServerbound}}

	// --- Combat: drop (serverbound) ---
	case "CWvsContext::SendDropPickUpRequest":
		// CSV: ITEM_PICKUP — atlas PickUp (handle = "DropPickUpHandle").
		return []candidate{{name: "PickUp", pkg: "drop", dir: csvpkg.DirServerbound}}

	// --- Combat: reactor (serverbound) ---
	case "CReactorPool::FindHitReactor":
		// CSV: DAMAGE_REACTOR — atlas HitRequest (handle = "ReactorHitHandle").
		return []candidate{{name: "HitRequest", pkg: "reactor", dir: csvpkg.DirServerbound}}

	// --- Combat: pet (serverbound) ---
	case "CWvsContext::SendActivatePetRequest":
		// CSV: SPAWN_PET (serverbound) — atlas Spawn (handle = "PetSpawnHandle").
		return []candidate{{name: "Spawn", pkg: "pet", dir: csvpkg.DirServerbound}}
	case "CVecCtrlPet::EndUpdateActive":
		// CSV: MOVE_PET (serverbound) — atlas MovementRequest.
		return []candidate{{name: "MovementRequest", pkg: "pet", dir: csvpkg.DirServerbound}}
	case "CPet::DoAction":
		// CSV: PET_CHAT (serverbound) — atlas ChatRequest.
		return []candidate{{name: "ChatRequest", pkg: "pet", dir: csvpkg.DirServerbound}}
	case "CPet::ParseCommand":
		// CSV: PET_COMMAND (serverbound) — atlas Command.
		return []candidate{{name: "Command", pkg: "pet", dir: csvpkg.DirServerbound}}
	case "CPet::SendUpdateExceptionListRequest":
		// CSV: PET_EXCLUDE_ITEMS — atlas ExcludeItem.
		return []candidate{{name: "ExcludeItem", pkg: "pet", dir: csvpkg.DirServerbound}}
	case "CWvsContext::SendPetFoodItemUseRequest":
		// CSV: PET_FOOD — atlas Food.
		return []candidate{{name: "Food", pkg: "pet", dir: csvpkg.DirServerbound}}
	case "CWvsContext::SendStatChangeItemUseRequestByPetQ":
		// CSV: PET_AUTO_POT — atlas ItemUse.
		return []candidate{{name: "ItemUse", pkg: "pet", dir: csvpkg.DirServerbound}}
	case "CPet::SendDropPickUpRequest":
		// CSV: PET_LOOT — atlas DropPickUp.
		return []candidate{{name: "DropPickUp", pkg: "pet", dir: csvpkg.DirServerbound}}

	// --- Social: note ---
	// CSV: MEMO_RESULT (clientbound, opcode 0x28/40 in GMS v95) → CWvsContext::OnMemoResult
	// dispatches on a leading mode byte to 4 sub-ops. Each sub-op modelled via a #-suffixed
	// synthetic IDA entry so the pipeline produces one report per atlas struct.
	case "CWvsContext::OnMemoResult#Display":
		// mode=3 (SHOW): count byte + loop of GW_Memo::Decode entries.
		// Atlas struct: note/clientbound/display.go Display.
		return []candidate{{name: "Display", pkg: "note", dir: csvpkg.DirClientbound}}
	case "CWvsContext::OnMemoResult#SendSuccess":
		// mode=4 (SEND_SUCCESS): no additional bytes.
		// Atlas struct: note/clientbound/operation.go SendSuccess.
		return []candidate{{name: "SendSuccess", pkg: "note", dir: csvpkg.DirClientbound}}
	case "CWvsContext::OnMemoResult#SendError":
		// mode=5 (SEND_ERROR): 1 errorCode byte.
		// Atlas struct: note/clientbound/operation.go SendError.
		return []candidate{{name: "SendError", pkg: "note", dir: csvpkg.DirClientbound}}
	case "CWvsContext::OnMemoResult#Refresh":
		// mode=7 (REFRESH): no additional bytes.
		// Atlas struct: note/clientbound/operation.go Refresh.
		return []candidate{{name: "Refresh", pkg: "note", dir: csvpkg.DirClientbound}}

	// CSV: NOTE_ACTION (serverbound, opcode 0x9A/154 in GMS v95) — three FNames share
	// this opcode; each represents a different sub-operation.
	case "CWvsContext::OnMemoNotify_Receive":
		// Sub-op 2 (REQUEST): client sends op=2 to request memo list refresh.
		// Atlas struct: note/serverbound/operation.go Operation (op-byte dispatcher).
		// Verdict will be ⚠️ "op-byte dispatcher; sub-ops audited individually" — see OP-FAMILY-note in _pending.md.
		return []candidate{{name: "Operation", pkg: "note", dir: csvpkg.DirServerbound}}
	case "CMemoListDlg::SetRet":
		// Sub-op 1 (DISCARD): client sends selected memo SN list for deletion.
		// Atlas struct: note/serverbound/operation_discard.go OperationDiscard.
		return []candidate{{name: "OperationDiscard", pkg: "note", dir: csvpkg.DirServerbound}}
	case "CCashShop::OnCashItemResLoadGiftDone":
		// Sub-op 0 (SEND): client sends note with recipient name + message body.
		// Atlas struct: note/serverbound/operation_send.go OperationSend.
		// Note: CCashShop gift-accept path uses the same NOTE_ACTION opcode with op=0;
		// the wire layout (EncodeStr toName + EncodeStr message) matches atlas OperationSend.
		return []candidate{{name: "OperationSend", pkg: "note", dir: csvpkg.DirServerbound}}

	// --- Social: buddy ---
	// CSV: BUDDYLIST (clientbound, opcode 0x24/36 in GMS v95) → CWvsContext::OnFriendResult
	// dispatches on a leading mode byte to multiple sub-ops. Each atlas writer struct is
	// modelled via a #-suffixed synthetic IDA entry (same pattern as note sub-ops above).
	case "CWvsContext::OnFriendResult#CapacityUpdate":
		// mode=0x15 (21, CAPACITY_UPDATE): Decode1(nFriendMax).
		// Atlas struct: buddy/clientbound/capacity_update.go CapacityUpdate.
		return []candidate{{name: "CapacityUpdate", pkg: "buddy", dir: csvpkg.DirClientbound}}
	case "CWvsContext::OnFriendResult#ChannelChange":
		// mode=0x14 (20, CHANNEL_CHANGE): Decode4(charId) + Decode1(inShop) + Decode4(channelId).
		// Atlas struct: buddy/clientbound/channel_change.go ChannelChange.
		return []candidate{{name: "ChannelChange", pkg: "buddy", dir: csvpkg.DirClientbound}}
	case "CWvsContext::OnFriendResult#Error":
		// mode=0x0B–0x17 (error sub-ops): mode byte only; no further packet reads on success
		// path (error cases show StringPool notice dialogs). Sub-op enum deferred to _pending.md.
		// Atlas struct: buddy/clientbound/error.go Error.
		return []candidate{{name: "Error", pkg: "buddy", dir: csvpkg.DirClientbound}}
	case "CWvsContext::OnFriendResult#Invite":
		// mode=0x09 (INVITE): Decode4(origId) + DecodeStr(origName) + Decode4 + Decode4 +
		// GW_Friend::Insert(39 bytes) + Decode1(inShop).
		// Atlas struct: buddy/clientbound/invite.go Invite.
		return []candidate{{name: "Invite", pkg: "buddy", dir: csvpkg.DirClientbound}}
	case "CWvsContext::OnFriendResult#ListUpdate":
		// mode=0x07/0x0A/0x12 (LIST_UPDATE): Decode1(count) + count×39(GW_Friend) + count×4(inShop).
		// Atlas struct: buddy/clientbound/list_update.go ListUpdate.
		// ⚠️ Analyzer flattens loop; IDA loop-bound citation: CFriend::Reset@0xa10760.
		return []candidate{{name: "ListUpdate", pkg: "buddy", dir: csvpkg.DirClientbound}}
	case "CWvsContext::OnFriendResult#Update":
		// mode=0x08 (UPDATE): Decode4(charId) + GW_Friend(39 bytes) + Decode1(inShop).
		// Atlas struct: buddy/clientbound/update.go Update.
		return []candidate{{name: "Update", pkg: "buddy", dir: csvpkg.DirClientbound}}

	// CSV: BUDDYLIST_MODIFY (serverbound, opcode 0x99/153 in GMS v95) — three FNames share
	// this opcode; each represents a different sub-operation (op-byte at Encode1 position 0).
	case "CField::SendSetFriendMsg":
		// Sub-op 1 (ADD): Encode1(1) + EncodeStr(name) + EncodeStr(group).
		// Atlas struct: buddy/serverbound/operation_add.go OperationAdd.
		return []candidate{{name: "OperationAdd", pkg: "buddy", dir: csvpkg.DirServerbound}}
	case "CField::SendAcceptFriendMsg":
		// Sub-op 2 (ACCEPT): Encode1(2) + Encode4(friendId).
		// Atlas struct: buddy/serverbound/operation_accept.go OperationAccept.
		return []candidate{{name: "OperationAccept", pkg: "buddy", dir: csvpkg.DirServerbound}}
	case "CField::SendDeleteFriendMsg":
		// Sub-op 3 (DELETE): Encode1(3) + Encode4(friendId).
		// Atlas struct: buddy/serverbound/operation_delete.go OperationDelete.
		return []candidate{{name: "OperationDelete", pkg: "buddy", dir: csvpkg.DirServerbound}}

	// --- Social: messenger ---
	// CSV: MESSENGER (clientbound, opcode 0xAE/174 in GMS v95) → CUIMessenger::OnPacket
	// dispatches on a leading mode byte to 9 sub-handlers. Each atlas clientbound struct
	// is modelled via a #-suffixed synthetic IDA entry (same pattern as note/buddy sub-ops).
	case "CUIMessenger::OnPacket#Add":
		// mode=0 (OnEnter): Decode1(position) + AvatarLook::Decode + DecodeStr(name) +
		// Decode1(channelId) + Decode1(pad).
		// Atlas struct: messenger/clientbound/add.go Add.
		return []candidate{{name: "Add", pkg: "messenger", dir: csvpkg.DirClientbound}}
	case "CUIMessenger::OnPacket#Join":
		// mode=1 (OnSelfEnterResult): Decode1(position).
		// Atlas struct: messenger/clientbound/join.go Join.
		return []candidate{{name: "Join", pkg: "messenger", dir: csvpkg.DirClientbound}}
	case "CUIMessenger::OnPacket#Remove":
		// mode=2 (OnLeave): Decode1(position).
		// Atlas struct: messenger/clientbound/remove.go Remove.
		return []candidate{{name: "Remove", pkg: "messenger", dir: csvpkg.DirClientbound}}
	case "CUIMessenger::OnPacket#RequestInvite":
		// mode=3 (OnInvite, before instance check): DecodeStr(fromName) + Decode1(pad) +
		// Decode4(messengerId) + Decode1(pad). Static handler — no instance guard.
		// Atlas struct: messenger/clientbound/request_invite.go RequestInvite.
		return []candidate{{name: "RequestInvite", pkg: "messenger", dir: csvpkg.DirClientbound}}
	case "CUIMessenger::OnPacket#InviteSent":
		// mode=4 (OnInviteResult): DecodeStr(msg) + Decode1(success/bool).
		// Atlas struct: messenger/clientbound/invite_sent.go InviteSent.
		return []candidate{{name: "InviteSent", pkg: "messenger", dir: csvpkg.DirClientbound}}
	case "CUIMessenger::OnPacket#InviteDeclined":
		// mode=5 (OnBlocked): DecodeStr(blockedUser) + Decode1(declineMode).
		// Atlas struct: messenger/clientbound/invite_declined.go InviteDeclined.
		// ⚠️ declineMode sub-enum deferred to _pending.md (OP-FAMILY-messenger-decline).
		return []candidate{{name: "InviteDeclined", pkg: "messenger", dir: csvpkg.DirClientbound}}
	case "CUIMessenger::OnPacket#Chat":
		// mode=6 (OnChat): DecodeStr(chatLine — format "name : msg").
		// Atlas struct: messenger/clientbound/chat.go Chat.
		return []candidate{{name: "Chat", pkg: "messenger", dir: csvpkg.DirClientbound}}
	case "CUIMessenger::OnPacket#Update":
		// mode=7 (OnAvatar): Decode1(position) + AvatarLook::Decode.
		// ❌ Atlas Update also encodes name + channelId + pad, which OnAvatar does NOT read.
		// Atlas struct: messenger/clientbound/update.go Update.
		return []candidate{{name: "Update", pkg: "messenger", dir: csvpkg.DirClientbound}}

	// CSV: MESSENGER (serverbound, opcode 0x8F/143 in GMS v95) — multiple FNames share
	// this opcode; each encodes a different sub-op byte at Encode1 position 0.
	case "CUIMessenger::OnCreate":
		// Sub-op 0 (ENTER): Encode1(0) + Encode4(messengerId) — client accepts invite.
		// Atlas struct: messenger/serverbound/operation_answer_invite.go OperationAnswerInvite.
		return []candidate{{name: "OperationAnswerInvite", pkg: "messenger", dir: csvpkg.DirServerbound}}
	case "CUIMessenger::OnDestroy":
		// Sub-op 2 (LEAVE): Encode1(2) — client leaves/closes the messenger window.
		// Atlas struct: messenger/serverbound/operation.go Operation (op-byte dispatcher).
		// ⚠️ Operation only carries the mode byte; full op-family deferred to _pending.md
		// (OP-FAMILY-messenger-serverbound).
		return []candidate{{name: "Operation", pkg: "messenger", dir: csvpkg.DirServerbound}}
	case "CUIMessenger::SendInviteMsg":
		// Sub-op 3 (INVITE): Encode1(3) + EncodeStr(targetCharacter).
		// Atlas struct: messenger/serverbound/operation_invite.go OperationInvite.
		return []candidate{{name: "OperationInvite", pkg: "messenger", dir: csvpkg.DirServerbound}}
	case "CFadeWnd::SendCloseMessage":
		// Sub-op 5 (DECLINE): Encode1(5) + EncodeStr(fromName) + EncodeStr(myName) + Encode1(0).
		// CFadeWnd handles multiple dialog types (type=0 → messenger decline, type=1 → buddy delete,
		// type=2/3 → miniroom, type=5 → guild); only type=0 maps to messenger OperationDeclineInvite.
		// Atlas struct: messenger/serverbound/operation_decline_invite.go OperationDeclineInvite.
		return []candidate{{name: "OperationDeclineInvite", pkg: "messenger", dir: csvpkg.DirServerbound}}
	case "CUIMessenger::ProcessChat":
		// Sub-op 6 (CHAT): Encode1(6) + EncodeStr(chatLine — format "name : msg").
		// Atlas struct: messenger/serverbound/operation_chat.go OperationChat.
		return []candidate{{name: "OperationChat", pkg: "messenger", dir: csvpkg.DirServerbound}}

	// --- Social: chat (clientbound) ---
	// CSV: CHATTEXT (0xB5 / 181) and CHATTEXT1 (0xB6 / 182) both dispatch to CUser::OnChat.
	// characterId is consumed by CUserPool::OnUserRemotePacket before calling OnChat;
	// the function itself reads Decode1(isGM) + DecodeStr(msg) + Decode1(bOnlyBalloon).
	// Atlas GeneralChat writes WriteInt(characterId)+WriteBool(gm)+WriteAsciiString(msg)+WriteBool(show).
	// The characterId prefix mismatch is a known dispatcher-prefix pattern (same as CharacterSpawn etc.).
	case "CUser::OnChat":
		// CSV: CHATTEXT / CHATTEXT1 — atlas GeneralChat (clientbound chat/general.go).
		return []candidate{{name: "GeneralChat", pkg: "chat", dir: csvpkg.DirClientbound}}

	// CSV: MULTICHAT (0x96 / 150) → CField::OnGroupMessage.
	// Dispatches on a leading mode byte (Decode1): 0=buddy, 1=party, 2=guild, 3=alliance, 6=expedition.
	// Atlas MultiChat writes WriteByte(mode)+WriteAsciiString(from)+WriteAsciiString(message) — parameterised.
	// Sub-op value space: ⚠️ deferred to _pending.md (single consolidated chat row).
	case "CField::OnGroupMessage":
		// CSV: MULTICHAT — atlas MultiChat (clientbound chat/multi.go).
		return []candidate{{name: "MultiChat", pkg: "chat", dir: csvpkg.DirClientbound}}

	// CSV: WHISPER (0x97 / 151) → CField::OnWhisper.
	// Dispatches on a leading mode byte: 9/10=find, 18=receive-whisper, 34=blocked, 146=weather.
	// All atlas clientbound whisper structs (WhisperSendResult, WhisperReceive, etc.) write mode as first byte — parameterised.
	// Sub-op value space: ⚠️ deferred to _pending.md (single consolidated chat row).
	case "CField::OnWhisper":
		// Use WhisperReceive as the representative struct (mode=18 branch).
		return []candidate{{name: "WhisperReceive", pkg: "chat", dir: csvpkg.DirClientbound}}

	// CSV: SPOUSE_CHAT (0x98 / 152) → CField::OnCoupleMessage.
	// Dispatches on a leading mode byte (Decode1 - 4): mode=4 (own message), mode=5 (partner message).
	// Both sub-modes are parameterised; atlas world_message.go and whisper.go write mode as first byte.
	// Sub-op value space: ⚠️ deferred to _pending.md (single consolidated chat row).
	// Note: SPOUSE_CHAT is absent in v95 template (opcode 0 in serverbound SPOUSE_CHAT for GMS). No template
	// opcode mapping; the clientbound opcode is 0x98. Atlas has no dedicated SPOUSE_CHAT writer yet; mapping
	// via CField::OnCoupleMessage to WhisperWeather as a placeholder for pipeline coverage.
	case "CField::OnCoupleMessage":
		// Use WhisperWeather as a representative sub-op struct for pipeline coverage.
		return []candidate{{name: "WhisperWeather", pkg: "chat", dir: csvpkg.DirClientbound}}

	// CSV: SERVERMESSAGE (0x47 / 71) → CWvsContext::OnBroadcastMsg.
	// Dispatches on a leading mode byte (Decode1): multiple sub-modes for notice/megaphone/ticker/etc.
	// All atlas world_message.go and world_message_extra.go structs write mode as first byte — parameterised.
	// Sub-op value space: ⚠️ deferred to _pending.md (single consolidated chat row).
	case "CWvsContext::OnBroadcastMsg":
		// Use WorldMessageSimple as the representative struct (covers the common Notice/PopUp/Megaphone modes).
		return []candidate{{name: "WorldMessageSimple", pkg: "chat", dir: csvpkg.DirClientbound}}

	// --- Social: chat (serverbound) ---
	// CSV: GENERAL_CHAT (0x36 / 54) → CField::SendChatMsg.
	// Wire: Encode4(update_time) + EncodeStr(sText) + Encode1(bOnlyBalloon).
	// Atlas General writes: WriteInt(updateTime, GMS>83 gate) + WriteAsciiString(msg) + WriteBool(bOnlyBalloon).
	// Gate fires for v95 → all 3 fields written → matches IDA wire exactly.
	case "CField::SendChatMsg":
		// CSV: GENERAL_CHAT — atlas General (serverbound chat/general.go).
		return []candidate{{name: "General", pkg: "chat", dir: csvpkg.DirServerbound}}

	// CSV: MULTI_CHAT (0xDD / 221) → CUIStatusBar::SendGroupMessage.
	// Wire at LABEL_24: Encode4(updateTime) + Encode1(nChatTarget) + Encode1(nMemberCnt) +
	// loop(Encode4×n memberIds) + EncodeStr(sText). Atlas Multi writes chatType + recipientCount +
	// loop(recipients) + chatText — the updateTime prefix is additional in v95.
	// Sub-op value space (chatType): ⚠️ deferred to _pending.md (single consolidated chat row).
	case "CUIStatusBar::SendGroupMessage":
		// CSV: MULTI_CHAT — atlas Multi (serverbound chat/multi.go).
		return []candidate{{name: "Multi", pkg: "chat", dir: csvpkg.DirServerbound}}

	// CSV: WHISPER (0xDE / 222) → CField::SendChatMsgWhisper (and SendLocationWhisper for find queries).
	// Wire for chat path (LABEL_79): Encode1(mode=6) + Encode4(updateTime) + EncodeStr(targetName) + EncodeStr(msg).
	// Atlas Whisper writes: WriteByte(mode) + WriteInt(updateTime, GMS>=95) + WriteAsciiString(targetName) +
	// optional WriteAsciiString(msg for mode==CHAT). The atlas WhisperMode constants include FIND(5)/CHAT(6).
	// Sub-op value space (WhisperMode): ⚠️ deferred to _pending.md (single consolidated chat row).
	case "CField::SendChatMsgWhisper":
		// CSV: WHISPER — atlas Whisper (serverbound chat/whisper.go).
		return []candidate{{name: "Whisper", pkg: "chat", dir: csvpkg.DirServerbound}}

	// --- Social: party (clientbound) ---
	// CSV: PARTY_OPERATION (clientbound, opcode 0x3E/62 in GMS v95) → CWvsContext::OnPartyResult
	// dispatches on a leading mode byte to 10+ sub-ops. Each atlas clientbound struct is modelled
	// via a #-suffixed synthetic IDA entry (same pattern as note/buddy/messenger sub-ops).
	case "CWvsContext::OnPartyResult#Created":
		// mode=8: client builds local party from CharacterData; server sends Decode4(partyId) +
		// Decode4(townPortalFromId) + Decode4(townPortalToId) + Decode2(x) + Decode2(y).
		// Atlas Created writes: mode(1) + partyId(4) + 2×EmptyMapId(8) + 2×short(4) = matches.
		return []candidate{{name: "Created", pkg: "party/clientbound", dir: csvpkg.DirClientbound}}
	case "CWvsContext::OnPartyResult#Invite":
		// mode=4: Decode4(partyId) + DecodeStr(inviterName) + Decode4(inviterJob) + Decode4(inviterLevel) +
		// Decode1(autoJoinFlag). Atlas Invite writes: mode(1) + partyId(4) + name(str) + 0x00(1).
		// ⚠️ Wire shape mismatch: IDA reads 2 extra Decode4 fields (job, level) not in atlas Invite.
		return []candidate{{name: "Invite", pkg: "party/clientbound", dir: csvpkg.DirClientbound}}
	case "CWvsContext::OnPartyResult#Disband":
		// mode=12 + Decode1(isForced=0): no PARTYDATA follows when isForced branch not taken.
		// Atlas Disband writes: mode(1) + partyId(4) + targetId(4) + 0x00(1) + partyId(4).
		// ⚠️ Wire shape varies: IDA mode=12 reads Decode4(partyId)+Decode4(targetId)+Decode1(isForced)+
		// optional(Decode1+DecodeStr+PARTYDATA). The no-member path (isForced=0, different charId)
		// writes zero-fills party, consistent with Disband. Tool-limitation: branch-flattening.
		return []candidate{{name: "Disband", pkg: "party/clientbound", dir: csvpkg.DirClientbound}}
	case "CWvsContext::OnPartyResult#Error":
		// mode=9,10,13,17,18,22,29,32–34,36 (error string pool nodes; no Decode calls beyond mode byte).
		// Atlas Error writes: mode(1) + name(str). ⚠️ Many error modes read no additional data from
		// packet; the name field is resolved server-side. Sub-op enum deferred to _pending.md.
		return []candidate{{name: "Error", pkg: "party/clientbound", dir: csvpkg.DirClientbound}}
	case "CWvsContext::OnPartyResult#ChangeLeader":
		// mode=31: Decode4(newLeaderId) + Decode1(disconnected).
		// Atlas ChangeLeader writes: mode(1) + targetCharacterId(4) + disconnected(1). ✓
		return []candidate{{name: "ChangeLeader", pkg: "party/clientbound", dir: csvpkg.DirClientbound}}
	case "CWvsContext::OnPartyResult#Join":
		// mode=15: Decode4(partyId) + DecodeStr(targetName) + PARTYDATA::Decode(378 bytes).
		// Atlas Join writes: mode(1) + partyId(4) + name(str) + WritePartyData(298 bytes).
		// ⚠️ WritePartyData is 80 bytes short of PARTYDATA buffer (missing aTownPortal skillId
		// + aPQReward + aPQRewardType + dwPQRewardMobTemplateID + bPQReward).
		return []candidate{{name: "Join", pkg: "party/clientbound", dir: csvpkg.DirClientbound}}
	case "CWvsContext::OnPartyResult#Left":
		// mode=12 + Decode1(isForced=1): Decode4(partyId)+Decode4(targetId)+Decode1(1)+Decode1(isForced)+
		// DecodeStr(targetName)+PARTYDATA::Decode(378 bytes).
		// Atlas Left writes: mode(1) + partyId(4) + targetId(4) + 0x01(1) + forced(1) + name(str) +
		// WritePartyData(298 bytes). ⚠️ Same WritePartyData 80-byte shortfall as Join/Update.
		return []candidate{{name: "Left", pkg: "party/clientbound", dir: csvpkg.DirClientbound}}
	case "CWvsContext::OnPartyResult#Update":
		// mode=7/38: Decode4(partyId) + PARTYDATA::Decode(378 bytes).
		// Atlas Update writes: mode(1) + partyId(4) + WritePartyData(298 bytes).
		// ⚠️ WritePartyData is 80 bytes short of PARTYDATA buffer (missing portal skillId field +
		// PQ reward fields). Hot-path: 4-variant byte-output sweep mandatory for any fix.
		return []candidate{{name: "Update", pkg: "party/clientbound", dir: csvpkg.DirClientbound}}
	case "CWvsContext::OnPartyResult#OperationBody":
		// operation_body.go is a helper that assembles sub-op packets via WithResolvedCode;
		// it carries no independent wire shape. Model via the dispatcher synthetic entry.
		// ⚠️ OP-FAMILY-party: sub-op enum drift deferred to _pending.md.
		return []candidate{{name: "OperationBody", pkg: "party/clientbound", dir: csvpkg.DirClientbound}}

	// CSV: UPDATE_PARTYMEMBER_HP (clientbound, opcode 0xD6/214 in GMS v95) → CUserRemote::OnReceiveHP.
	// IDA reads: Decode4(hp) + Decode4(maxHp). The characterId is consumed upstream by
	// CUserPool::OnUserRemotePacket before OnReceiveHP is called (dispatcher-prefix pattern).
	// Atlas MemberHP writes: WriteInt(characterId) + WriteInt(hp) + WriteInt(maxHp) — characterId
	// is the dispatcher-level prefix; the packet body visible to OnReceiveHP is just hp+maxHp.
	// ✓ Wire is correct: server emits charId + hp + maxHp; dispatcher consumes charId then calls OnReceiveHP.
	// Hot-path: 4-variant byte-output sweep mandatory for any encoder fix.
	case "CUserRemote::OnReceiveHP":
		// CSV: UPDATE_PARTYMEMBER_HP — atlas MemberHP (party/clientbound/member_hp.go).
		return []candidate{{name: "MemberHP", pkg: "party/clientbound", dir: csvpkg.DirClientbound}}

	// --- Social: party (serverbound) ---
	// CSV: PARTY_OPERATION (serverbound, opcode 0x91/145 in GMS v95). Multiple CField functions
	// share this opcode; each encodes a different op-byte at Encode1 position 0.
	case "CField::SendCreateNewPartyMsg":
		// op=1 (CREATE): Encode1(1). No further fields.
		// Atlas Operation (serverbound/operation.go) is the op-byte dispatcher; op=1 maps to create.
		// ⚠️ OP-FAMILY-party-serverbound: op-byte family deferred to _pending.md.
		return []candidate{{name: "Operation", pkg: "party/serverbound", dir: csvpkg.DirServerbound}}
	case "CField::SendWithdrawPartyMsg":
		// op=2 (LEAVE): Encode1(2) + Encode1(0). Atlas serverbound has no separate Leave struct;
		// the op=2 sub-op is handled by the Operation dispatcher.
		// ⚠️ OP-FAMILY-party-serverbound: op-byte family deferred to _pending.md.
		// Note: Encode1(0) trailing byte not in atlas Operation (only carries op byte).
		return []candidate{{name: "Operation", pkg: "party/serverbound", dir: csvpkg.DirServerbound}}
	case "CField::SendKickPartyMsg":
		// op=5 (EXPEL): Encode1(5) + Encode4(targetCharacterId).
		// Atlas OperationExpel writes: WriteInt(targetCharacterId). ✓ (op byte consumed by dispatcher)
		return []candidate{{name: "OperationExpel", pkg: "party/serverbound", dir: csvpkg.DirServerbound}}
	case "CField::SendChangePartyBossMsg":
		// op=6 (CHANGE_LEADER): Encode1(6) + Encode4(targetCharacterId).
		// Atlas OperationChangeLeader writes: WriteInt(targetCharacterId). ✓
		return []candidate{{name: "OperationChangeLeader", pkg: "party/serverbound", dir: csvpkg.DirServerbound}}
	case "CField::SendJoinPartyMsg":
		// op=4 (INVITE): Encode1(4) + EncodeStr(targetName).
		// Atlas OperationInvite writes: WriteAsciiString(name). ✓ (op byte consumed by dispatcher)
		return []candidate{{name: "OperationInvite", pkg: "party/serverbound", dir: csvpkg.DirServerbound}}

	// CSV: PARTY_RESULT (serverbound, opcode 0x92/146 in GMS v95) → reject/accept invite response.
	// IDA path: CUIFadeYesNo::OnMouseButton sends op=0x16 (accept) or op=0x17/0x18 (decline/ignore).
	// Atlas InviteReject writes: WriteByte(unk) + WriteAsciiString(from). ✓
	// (The unk byte maps to the response code; 'from' is the inviter name.)
	case "CWvsContext::OnPartyResult#InviteReject":
		// DENY_PARTY_REQUEST (serverbound) → atlas InviteReject (serverbound/invite_reject.go).
		// ⚠️ No direct IDA function found for DENY_PARTY_REQUEST; modelled via synthetic entry.
		// Wire: Encode1(declineCode) + EncodeStr(inviterName) — matches atlas layout.
		return []candidate{{name: "InviteReject", pkg: "party/serverbound", dir: csvpkg.DirServerbound}}

	// CSV: PARTY_RESULT (serverbound, opcode 0x92/146 in GMS v95) also maps to OperationJoin.
	// SendJoinPartyMsg (op=4) triggers server → client PARTY_OPERATION mode 15 (join flow).
	// The direct serverbound join packet is op=3 or a separate opcode.
	case "CField::SendJoinPartyMsg#OperationJoin":
		// Synthetic entry: client sends party join accept with partyId after receiving an invite.
		// Atlas OperationJoin writes: WriteInt(partyId). ✓
		return []candidate{{name: "OperationJoin", pkg: "party/serverbound", dir: csvpkg.DirServerbound}}
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

func locateAtlasFile(root, name, pkg string, dir csvpkg.Direction) (string, bool) {
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
		// When pkg is set, restrict to /<pkg>/<expectDir>/ so combat sub-domains
		// (monster/drop/reactor/pet) with colliding short struct names route
		// to the correct file.
		if pkg != "" {
			pkgNeedle := string(os.PathSeparator) + pkg + string(os.PathSeparator) + expectDir + string(os.PathSeparator)
			if !strings.Contains(path, pkgNeedle) {
				return nil
			}
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
