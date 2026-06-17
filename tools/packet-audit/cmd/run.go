package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
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

	process := func(c candidate, fname string) {
		direction, name, pkg := c.dir, c.name, c.pkg
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
		// Serverbound family-operation wrapper: Atlas writes the packet as a
		// separate [Operation/BBS sub-op] struct followed by the per-operation
		// body. The audit's candidate points only at the body, so reconstruct the
		// full wire by prepending the wrapper's shape — BUT only when the client
		// export actually carries the sub-op as its leading field(s). Faithful
		// baselines include it (compose → [wrapper, body] aligns with the client's
		// [sub-op, body]); incomplete baselines that omit the real sub-op byte are
		// left body-only so the audit neither manufactures a mismatch nor silently
		// blesses the omission. See the prefixName doc on `candidate`.
		if c.prefixName != "" {
			ppath, pok := locateAtlasFile(opts.AtlasPacket, c.prefixName, c.prefixPkg, direction)
			if !pok {
				fmt.Fprintln(stderr, "locate prefix", c.prefixName+": not found")
				return
			}
			pcalls, perr := atlaspacket.AnalyzeFileWithRegistry(ppath, c.prefixName, methodName(direction), reg)
			if perr != nil {
				fmt.Fprintln(stderr, "analyze prefix", c.prefixName+":", perr)
				return
			}
			if exportCarriesPrefix(reg, ctx, pcalls, fields.Calls) {
				calls = append(append([]atlaspacket.Call{}, pcalls...), calls...)
			}
		} else if c.prefixSubOps > 0 {
			// Wrapper-less serverbound sub-op: families like buddy (BUDDYLIST_MODIFY)
			// and npc-shop read the sub-op byte in the channel handler/dispatcher,
			// with NO Operation wrapper struct in the packet lib. Synthesize the
			// 1-byte sub-op prefix(es) and compose adaptively, same as a wrapper.
			pcalls := make([]atlaspacket.Call, 0, c.prefixSubOps)
			for i := 0; i < c.prefixSubOps; i++ {
				pcalls = append(pcalls, atlaspacket.Call{Kind: atlaspacket.KindWrite, Op: atlaspacket.Encode1})
			}
			if exportCarriesPrefix(reg, ctx, pcalls, fields.Calls) {
				calls = append(append([]atlaspacket.Call{}, pcalls...), calls...)
			}
		}
		flat := diff.FlattenWithRegistry(calls, ctx, reg)
		rows := diff.Diff(flat, fields)
		v := worstRow(rows)
		// Flat-diff-invalid reclassification: when the Atlas writer branches on a
		// condition the analyzer could not reduce to a version predicate (a
		// data-dependent field like m.enteringField, or a version-derived local
		// the flatten doesn't trace), the flat positional diff cannot faithfully
		// compare it — it merges/picks one arm while the client reads the runtime
		// arm. Such a ❌ is a modeling limitation, not a verified wire bug, so cap
		// it to 🔍 (Deferred). Only Blocker verdicts are touched, so clean ✅/⚠️
		// are never affected.
		flatInvalid := false
		if v == diff.VerdictBlocker && (hasUnresolvedBranch(calls) || clientReadsConditional(fields)) {
			v = diff.VerdictDeferred
			flatInvalid = true
		}
		// Absent-feature packet: the client never implements this read, so the
		// flat all-Atlas-extra diff is meaningless. Mark N/A (Deferred), never a
		// blocker — a version-generic Atlas writer that no client reads is not a
		// wire bug.
		if a, ok := src.(interface{ IsAbsent(string) bool }); ok && a.IsAbsent(fname) && v == diff.VerdictBlocker {
			v = diff.VerdictDeferred
			flatInvalid = true
		}
		writerName := qualifiedWriterName(pkg, name)
		if c.reportName != "" {
			writerName = c.reportName
		}
		pkt := report.Packet{
			WriterName:  writerName,
			IDAName:     fname,
			Address:     fields.Address,
			Variant:     fmt.Sprintf("%s/v%d", ctx.Region, ctx.MajorVersion),
			BranchDepth: branchDepth(calls),
			AtlasFile:   repoRelAtlasFile(atlasPath),
			Rows:        rows,
			Verdict:     v,
			FlatInvalid: flatInvalid,
		}
		if v.Severity() > worstVerdict.Severity() {
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
	for _, sc := range selectCandidates(idaExportFunctions(opts.IDASource)) {
		process(sc.candidate, sc.fname)
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
	// prefixName/prefixPkg, when set, name an Atlas writer whose Encode shape is
	// analyzed and PREPENDED to this candidate's body shape. They model a
	// serverbound family-operation wrapper — e.g. guild/serverbound `Operation`
	// (a 1-byte sub-op) or messenger/serverbound `Operation` (a 1-byte mode) —
	// that Atlas writes as a SEPARATE struct before the per-operation body, while
	// the client writes that sub-op inline as its leading field. Without this the
	// audit compares the client's [sub-op, body] against Atlas's body-only shape
	// and every field mis-aligns by one (a false ❌ that is really a modeling gap,
	// not a wire bug): the wrapper byte IS written/read by Atlas, just in the
	// family router rather than the leaf the candidate points at.
	prefixName string
	prefixPkg  string
	// prefixSubOps, when > 0, synthesizes that many leading 1-byte sub-op fields
	// to compose adaptively onto the body — for wrapper-LESS serverbound families
	// (buddy BUDDYLIST_MODIFY, npc-shop action) whose sub-op byte is read by the
	// channel handler rather than a packet-lib Operation struct. Mutually
	// exclusive with prefixName.
	prefixSubOps int
	// reportName, when set, overrides the derived qualifiedWriterName(pkg,name)
	// for the report file / matrix WriterName WITHOUT changing locateAtlasFile's
	// pkg-scoped struct resolution. Needed when the derived name would collide
	// across directions on the flat (writerName-keyed) audit dir — e.g. summon
	// clientbound SummonMove and serverbound Move both derive "SummonMove";
	// the serverbound side overrides to "SummonMoveHandle" so its report and
	// matrix cell (summon/serverbound/SummonMoveHandle) stay distinct.
	reportName string
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

// selectedCandidate pairs a resolved candidate with the IDA FName that won its
// (pkg, name) slot.
type selectedCandidate struct {
	candidate candidate
	fname     string
}

// orderedExportFNames returns fnames in a deterministic order that also encodes
// candidate-selection precedence. candidatesFromFName is many-to-one in places:
// a bare dispatcher root (e.g. CWvsContext::OnGuildResult) and its enriched
// "#"-suffixed synthetic sub-function entry (CWvsContext::OnGuildResult#RequestAgreement)
// can map to the same pkg::name candidate. The synthetic entry carries the full
// field list and is the correct model, so it must win. Ordering "#"-suffixed
// entries first — then lexicographically — gives a stable winner under the
// first-claim dedup in selectCandidates. Without this, Functions() returns map
// keys in random order and the per-packet verdict flipped between runs.
func orderedExportFNames(fnames []string) []string {
	out := append([]string(nil), fnames...)
	sort.SliceStable(out, func(i, j int) bool {
		hi := strings.Contains(out[i], "#")
		hj := strings.Contains(out[j], "#")
		if hi != hj {
			return hi
		}
		return out[i] < out[j]
	})
	return out
}

// selectCandidates resolves the deterministic winning FName for each
// (pkg, name) candidate produced by the export's FNames. The first FName (in
// orderedExportFNames order) to claim a given pkg::name key wins; later FNames
// mapping to the same key are skipped.
func selectCandidates(fnames []string) []selectedCandidate {
	seen := map[string]bool{}
	var out []selectedCandidate
	for _, fname := range orderedExportFNames(fnames) {
		for _, c := range candidatesFromFName(fname) {
			key := c.pkg + "::" + c.name
			if seen[key] {
				continue
			}
			seen[key] = true
			out = append(out, selectedCandidate{candidate: c, fname: fname})
		}
	}
	return out
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
	// --- task-092 Cluster-B/C: catch/taming + monster-book (character domain) ---
	case "CWvsContext::OnBridleMobCatchFail":
		// task-092 Cluster-B: BRIDLE_MOB_CATCH_FAIL — atlas BridleMobCatchFail
		// (writer = "BridleMobCatchFail"). Decode1 reason + Decode4 itemId +
		// trailing Decode4 (read, discarded). Byte-identical across all 5 versions.
		return []candidate{{name: "BridleMobCatchFail", pkg: "character", dir: csvpkg.DirClientbound}}
	case "CWvsContext::OnSetTamingMobInfo":
		// task-092 Cluster-B: SET_TAMING_MOB_INFO — atlas SetTamingMobInfo (writer =
		// "SetTamingMobInfo", pre-existing encoder). Decode4×4 (charId, level, exp,
		// fatigue) + Decode1 (levelUp). Byte-identical across all 5 versions.
		return []candidate{{name: "SetTamingMobInfo", pkg: "character", dir: csvpkg.DirClientbound}}
	case "CWvsContext::OnMonsterBookSetCard":
		// task-092 Cluster-C: MONSTER_BOOK_SET_CARD — atlas SetCard
		// (character/clientbound/monsterbook; writer = "MonsterBookSetCard"). Decode1
		// (added flag) + Decode4 cardId + Decode4 count. Same layout across versions.
		return []candidate{{name: "SetCard", pkg: "character", dir: csvpkg.DirClientbound}}
	case "CWvsContext::OnMonsterBookSetCover":
		// task-092 Cluster-C: MONSTER_BOOK_SET_COVER — atlas SetCover
		// (character/clientbound/monsterbook; writer = "MonsterBookSetCover"). Single
		// Decode4 (cover cardId). Same layout across versions.
		return []candidate{{name: "SetCover", pkg: "character", dir: csvpkg.DirClientbound}}
	case "CUserLocal::SetMonsterBookCover":
		// task-092 Cluster-C: MONSTER_BOOK_COVER (serverbound) — atlas Cover
		// (character/serverbound/monsterbook; handle = "MonsterBookCover"). The client
		// sends one Encode4 (cover cardId); CUserLocal::SetMonsterBookCover is the
		// named cover setter the send site delegates to (send site itself is
		// unnamed/inlined). v84: unnamed in the IDB → no export entry → blocker cell.
		return []candidate{{name: "Cover", pkg: "character", dir: csvpkg.DirServerbound}}
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

	// --- misc domain (task-069) ---
	case "CWvsContext::OnStatChanged":
		return []candidate{{name: "Changed", dir: csvpkg.DirClientbound}}
	case "CClientSocket::OnMigrateCommand":
		// ChannelChange collides with buddy/clientbound/channel_change.go
		// (CWvsContext::OnFriendResult#ChannelChange). pkg="channel" routes the
		// migrate-command channel change to channel/clientbound/change.go.
		return []candidate{{name: "ChannelChange", pkg: "channel", dir: csvpkg.DirClientbound}}
	case "CField::SendTransferChannelRequest":
		return []candidate{{name: "ChannelChangeRequest", dir: csvpkg.DirServerbound}}
	case "CUserLocal::OnOpenUI":
		return []candidate{{name: "Open", dir: csvpkg.DirClientbound}}
	case "CUserLocal::OnSetStandAloneMode":
		return []candidate{{name: "Disable", dir: csvpkg.DirClientbound}}
	case "CUserLocal::OnSetDirectionMode":
		return []candidate{{name: "Lock", dir: csvpkg.DirClientbound}}

	// --- fame bucket (task-069, sub-phase 2e) ---
	case "CWvsContext::OnGivePopularityResult#GiveResponse":
		return []candidate{{name: "GiveResponse", dir: csvpkg.DirClientbound}}
	case "CWvsContext::OnGivePopularityResult#ReceiveResponse":
		return []candidate{{name: "ReceiveResponse", dir: csvpkg.DirClientbound}}
	case "CWvsContext::OnGivePopularityResult#ErrorResponse":
		return []candidate{{name: "ErrorResponse", dir: csvpkg.DirClientbound}}
	case "CWvsContext::SendGivePopularityRequest":
		return []candidate{{name: "Change", dir: csvpkg.DirServerbound}}

	// --- merchant bucket (task-069, sub-phase 2f) ---
	case "CWvsContext::OnEntrustedShopCheckResult#OpenShop":
		return []candidate{{name: "OpenShop", dir: csvpkg.DirClientbound}}
	case "CWvsContext::OnEntrustedShopCheckResult#ErrorSimple":
		return []candidate{{name: "ErrorSimple", dir: csvpkg.DirClientbound}}
	case "CWvsContext::OnEntrustedShopCheckResult#ShopSearch":
		return []candidate{{name: "ShopSearch", dir: csvpkg.DirClientbound}}
	case "CWvsContext::OnEntrustedShopCheckResult#ShopRename":
		return []candidate{{name: "ShopRename", dir: csvpkg.DirClientbound}}
	case "CWvsContext::OnEntrustedShopCheckResult#RemoteShopWarp":
		return []candidate{{name: "RemoteShopWarp", dir: csvpkg.DirClientbound}}
	case "CWvsContext::OnEntrustedShopCheckResult#ConfirmManage":
		return []candidate{{name: "ConfirmManage", dir: csvpkg.DirClientbound}}
	case "CWvsContext::OnEntrustedShopCheckResult#FreeFormNotice":
		return []candidate{{name: "FreeFormNotice", dir: csvpkg.DirClientbound}}

	// --- quest bucket (task-069, sub-phase 2g) ---
	case "CWvsContext::OnScriptProgressMessage":
		return []candidate{{name: "ScriptProgress", dir: csvpkg.DirClientbound}}
	case "CWvsContext::ResignQuest#Action":
		return []candidate{{name: "Action", dir: csvpkg.DirServerbound}}
	case "CQuest::StartQuest#ActionScriptStart":
		return []candidate{{name: "ActionScriptStart", dir: csvpkg.DirServerbound}}
	case "CQuest::StartQuest#ActionScriptEnd":
		return []candidate{{name: "ActionScriptEnd", dir: csvpkg.DirServerbound}}

	// --- account bucket (task-069, sub-phase 2h) ---
	case "CLogin::OnCheckPinCodeResult#RegisterPin":
		return []candidate{{name: "RegisterPin", dir: csvpkg.DirServerbound}}
	case "CLogin::SendSetGenderPacket":
		return []candidate{{name: "SetGender", dir: csvpkg.DirServerbound}}

	// --- socket bucket (task-069, sub-phase 2i) ---
	case "CClientSocket::OnConnect#Hello":
		return []candidate{{name: "Hello", dir: csvpkg.DirClientbound}}
	case "CClientSocket::OnAliveReq#PingReceive":
		return []candidate{{name: "Ping", dir: csvpkg.DirClientbound}}
	case "CClientSocket::OnConnect#ChannelConnect":
		return []candidate{{name: "ChannelConnect", dir: csvpkg.DirServerbound}}
	case "CClientSocket::OnAliveReq#PongSend":
		return []candidate{{name: "Pong", dir: csvpkg.DirServerbound}}
	case "CClientSocket::OnConnect#StartError":
		return []candidate{{name: "StartError", dir: csvpkg.DirServerbound}}
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
	// task-092 Stage 4: the four serverbound attack requests. All share the
	// model.AttackInfo wire structure; each links to a thin per-op wrapper in
	// character/serverbound (AttackMeleeRequest/...Ranged/...Magic/...Touch) that
	// embeds AttackInfo, so each registry op gets a distinct packet/evidence —
	// mirroring how clientbound CUserRemote::OnAttack maps to the shared Attack
	// struct. The channel handlers (CharacterMelee/Ranged/Magic/TouchAttack) decode
	// the same model.AttackInfo directly; the wrappers are the matrix's per-op codec
	// representation and verify the identical wire structure.
	case "CUserLocal::TryDoingNormalAttack", "CUserLocal::TryDoingMeleeAttack":
		// CLOSE_RANGE_ATTACK. The v83 registry primary fname is TryDoingNormalAttack
		// (TryDoingMeleeAttack is an alt); both are basic-melee senders decoded by the
		// same AttackInfo(AttackTypeMelee). Map either to the shared wrapper.
		return []candidate{{name: "AttackMeleeRequest", pkg: "character", dir: csvpkg.DirServerbound}}
	case "CUserLocal::TryDoingShootAttack":
		// RANGED_ATTACK (0x2D v83). alt: TryDoingSmoothingMovingShootAttack.
		return []candidate{{name: "AttackRangedRequest", pkg: "character", dir: csvpkg.DirServerbound}}
	case "CUserLocal::TryDoingMagicAttack":
		// MAGIC_ATTACK (0x2E v83).
		return []candidate{{name: "AttackMagicRequest", pkg: "character", dir: csvpkg.DirServerbound}}
	case "CUserLocal::TryDoingBodyAttack":
		// TOUCH_MONSTER_ATTACK (0x2F v83). AttackTypeEnergy variant.
		return []candidate{{name: "AttackTouchRequest", pkg: "character", dir: csvpkg.DirServerbound}}
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
	case "CMob::OnAffected":
		// task-092 Cluster-A: MOB_AFFECTED — atlas MobAffected (writer =
		// "MobAffected"). Decode4 skillId + Decode2 delay.
		return []candidate{{name: "MobAffected", pkg: "monster", dir: csvpkg.DirClientbound}}
	case "CMob::OnSpecialEffectBySkill":
		// task-092 Cluster-A: MONSTER_SPECIAL_EFFECT_BY_SKILL — atlas
		// MonsterSpecialEffectBySkill (writer = "MonsterSpecialEffectBySkill").
		// Decode4 skillId (v83/v84/v87/jms); +Decode4 characterId +Decode2 delay (v95).
		return []candidate{{name: "MonsterSpecialEffectBySkill", pkg: "monster", dir: csvpkg.DirClientbound}}
	case "CMob::OnSuspendReset":
		// task-092 Cluster-A: RESET_MONSTER_ANIMATION — atlas
		// ResetMonsterAnimation (writer = "ResetMonsterAnimation"). Single Decode1 bool.
		return []candidate{{name: "ResetMonsterAnimation", pkg: "monster", dir: csvpkg.DirClientbound}}
	case "CMob::OnCatchEffect":
		// task-092 Cluster-B: CATCH_MONSTER — atlas CatchMonster (writer =
		// "CatchMonster"). Decode1 result (v83/v84/v87/jms); +Decode1 success (v95).
		// jms OnCatchEffect is unnamed (pins against CMob::ShowCatchEffect).
		return []candidate{{name: "CatchMonster", pkg: "monster", dir: csvpkg.DirClientbound}}
	case "CMob::ShowCatchEffect":
		// jms-only alias: jms CMob::OnCatchEffect is unnamed (dispatched via
		// sub_6EAE5F); the named ShowCatchEffect is the export key that carries the
		// 1×Decode1 catch layout for the jms CATCH_MONSTER evidence pin.
		return []candidate{{name: "CatchMonster", pkg: "monster", dir: csvpkg.DirClientbound}}
	case "CMob::OnEffectByItem":
		// task-092 Cluster-B: CATCH_MONSTER_WITH_ITEM — atlas CatchMonsterWithItem
		// (writer = "CatchMonsterWithItem"). Decode4 itemId + Decode1 result, all versions.
		return []candidate{{name: "CatchMonsterWithItem", pkg: "monster", dir: csvpkg.DirClientbound}}
	case "CMobPool::OnMobCrcKeyChanged":
		// One IDA function backs BOTH directions: it reads the clientbound
		// MOB_CRC_KEY_CHANGED (Decode4 crcKey) and emits the serverbound
		// MOB_CRC_KEY_CHANGED_REPLY (empty payload) acknowledgement. task-092
		// Cluster-D models them as two structs.
		return []candidate{
			{name: "MobCrcKeyChanged", pkg: "monster", dir: csvpkg.DirClientbound},
			{name: "MobCrcKeyChangedReply", pkg: "monster", dir: csvpkg.DirServerbound},
		}

	// --- Combat: Monster Carnival (task-092 Cluster E) ---
	// Carnival codecs live under monster/carnival/{clientbound,serverbound}; the
	// struct names are globally unique so pkg is left empty (locateAtlasFile finds
	// the file by struct name; PacketID = monster/carnival/<dir>/<Struct>).
	case "CField_MonsterCarnival::OnEnter":
		// MONSTER_CARNIVAL_START — Decode1 team, 6x Decode2 CP, then per-summon-slot
		// Decode1 loop. All 5 versions.
		return []candidate{{name: "MonsterCarnivalStart", dir: csvpkg.DirClientbound}}
	case "CField_MonsterCarnival::OnPersonalCP":
		// MONSTER_CARNIVAL_OBTAINED_CP — 2x Decode2 (cp,total). All 5 versions.
		return []candidate{{name: "MonsterCarnivalObtainedCP", dir: csvpkg.DirClientbound}}
	case "CField_MonsterCarnival::OnTeamCP":
		// MONSTER_CARNIVAL_PARTY_CP — Decode1 team + 2x Decode2 (cp,total). All 5 versions.
		return []candidate{{name: "MonsterCarnivalPartyCP", dir: csvpkg.DirClientbound}}
	case "CField_MonsterCarnival::OnRequestResult":
		// One dispatcher fn backs TWO distinct clientbound ops, demuxed on the
		// OnPacket arg: SUMMON (arg != 0: Decode1 tab, Decode1 idx, DecodeStr name)
		// and MESSAGE (arg == 0: a single Decode1 selector; strings from StringPool).
		// Both candidates are emitted; each op-row grades worst-of and matches its
		// own marker. All 5 versions.
		return []candidate{
			{name: "MonsterCarnivalSummon", dir: csvpkg.DirClientbound},
			{name: "MonsterCarnivalMessage", dir: csvpkg.DirClientbound},
		}
	case "CField_MonsterCarnival::OnProcessForDeath":
		// MONSTER_CARNIVAL_DIED — Decode1 team, DecodeStr name, Decode1 lostCp. All 5 versions.
		return []candidate{{name: "MonsterCarnivalDied", dir: csvpkg.DirClientbound}}
	case "CField_MonsterCarnival::OnShowMemberOutMsg":
		// MONSTER_CARNIVAL_LEAVE — 2x Decode1 (leader,team) + DecodeStr name. All 5 versions.
		return []candidate{{name: "MonsterCarnivalLeave", dir: csvpkg.DirClientbound}}
	case "CField_MonsterCarnival::OnShowGameResult":
		// MONSTER_CARNIVAL_RESULT — a single Decode1 outcome selector. All 5 versions.
		return []candidate{{name: "MonsterCarnivalResult", dir: csvpkg.DirClientbound}}
	case "CUIMonsterCarnival::RequestSend":
		// MONSTER_CARNIVAL (serverbound) — Encode1 tab + Encode4 (idx-1). All 5 versions.
		return []candidate{{name: "MonsterCarnival", dir: csvpkg.DirServerbound}}

	// --- Combat: monster version-tail (task-092 Cluster F) ---
	case "CMob::OnIncMobChargeCount":
		// INC_MOB_CHARGE_COUNT — atlas IncMobChargeCount. Two Decode4
		// (chargeCount, attackReady). v83/v84/v87/v95; jms version-absent.
		return []candidate{{name: "IncMobChargeCount", pkg: "monster", dir: csvpkg.DirClientbound}}
	case "CMob::OnMobSkillDelay":
		// MOB_SKILL_DELAY — atlas MobSkillDelay. Four Decode4 (delay, skillId,
		// skillLevel, option). v84/v87/v95/jms; v83 version-absent (no dispatcher case).
		return []candidate{{name: "MobSkillDelay", pkg: "monster", dir: csvpkg.DirClientbound}}
	case "CMob::OnMobSpeaking":
		// MOB_SPEAKING — atlas MobSpeaking. Two Decode4 forwarded to TrySpeaking.
		// All five versions.
		return []candidate{{name: "MobSpeaking", pkg: "monster", dir: csvpkg.DirClientbound}}
	case "CMob::OnMobAttackedByMob":
		// MOB_ATTACKED_BY_MOB — atlas MobAttackedByMob. Decode1 attackIndex +
		// Decode4 damage, then (attackIndex>-2) Decode4 mobTemplateId + Decode1 left.
		// All five versions.
		return []candidate{{name: "MobAttackedByMob", pkg: "monster", dir: csvpkg.DirClientbound}}
	case "CMob::OnNextAttack":
		// MOB_NEXT_ATTACK — atlas MobNextAttack. Single Decode4. v95-only.
		return []candidate{{name: "MobNextAttack", pkg: "monster", dir: csvpkg.DirClientbound}}
	case "CMob::OnEscortReturnBefore":
		// MOB_ESCORT_RETURN_BEFORE — atlas MobEscortReturnBefore. Single Decode4.
		// v95 + jms.
		return []candidate{{name: "MobEscortReturnBefore", pkg: "monster", dir: csvpkg.DirClientbound}}
	case "CMob::OnEscortStopEndPermmision":
		// MOB_ESCORT_STOP — atlas MobEscortStop. Empty payload (handler takes no
		// CInPacket). v95 (jms dispatches but has no registry row).
		return []candidate{{name: "MobEscortStop", pkg: "monster", dir: csvpkg.DirClientbound}}
	case "CMob::OnEscortStopSay":
		// MOB_ESCORT_STOP_SAY — atlas MobEscortStopSay. Decode4 duration +
		// Decode4 chatBalloon + Decode1 weather + Decode1 hasText (DecodeStr text +
		// Decode4 action). v95 + jms.
		return []candidate{{name: "MobEscortStopSay", pkg: "monster", dir: csvpkg.DirClientbound}}
	case "CMob::OnEscortFullPath":
		// MOB_ESCORT_FULL_PATH — atlas MobEscortFullPath. Decode4 mode + Decode4
		// count + per-waypoint loop + tail + arrive/reset bools. v95 + jms.
		return []candidate{{name: "MobEscortFullPath", pkg: "monster", dir: csvpkg.DirClientbound}}
	case "CMob::SendCollisionEscort":
		// MOB_ESCORT_COLLISION — atlas MobEscortCollision. Two Encode4 (mobCrc,
		// dest). v95 + jms.
		return []candidate{{name: "MobEscortCollision", pkg: "monster", dir: csvpkg.DirServerbound}}
	case "CMob::SendRequestEscortPath":
		// MOB_REQUEST_ESCORT_INFO — atlas MobRequestEscortInfo. Single Encode4
		// (mobCrc). v95 + jms.
		return []candidate{{name: "MobRequestEscortInfo", pkg: "monster", dir: csvpkg.DirServerbound}}
	case "CMob::SendEscortStopEndRequest":
		// MOB_ESCORT_STOP_END_REQUEST — atlas MobEscortStopEndRequest. Single
		// Encode4 (mobCrc). v95 + jms.
		return []candidate{{name: "MobEscortStopEndRequest", pkg: "monster", dir: csvpkg.DirServerbound}}

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

	// --- Combat: summon (clientbound, writers) ---
	// All six summon writers dispatch through CSummonedPool::OnPacket@0x75ac70
	// (v95 PDB build), which reads Decode4(charId) then routes the per-op leaf.
	// The export entries (CSummonedPool::On*) bake in the charId + oid prefix
	// (same pattern as CMob::OnMove baking dwMobId). The atlas clientbound structs
	// carry the Summon prefix in their names (SummonSpawn, …) and are globally
	// unique among clientbound files, so pkg is left empty (writerName == struct
	// name → PacketID summon/clientbound/SummonSpawn). v95 reference (PDB build).
	case "CSummonedPool::OnCreated":
		return []candidate{{name: "SummonSpawn", dir: csvpkg.DirClientbound}}
	case "CSummonedPool::OnRemoved":
		return []candidate{{name: "SummonRemove", dir: csvpkg.DirClientbound}}
	case "CSummonedPool::OnMove":
		return []candidate{{name: "SummonMove", dir: csvpkg.DirClientbound}}
	case "CSummonedPool::OnAttack":
		return []candidate{{name: "SummonAttack", dir: csvpkg.DirClientbound}}
	case "CSummonedPool::OnSkill":
		return []candidate{{name: "SummonSkill", dir: csvpkg.DirClientbound}}
	case "CSummonedPool::OnHit":
		return []candidate{{name: "SummonDamage", dir: csvpkg.DirClientbound}}

	// --- Combat: summon (serverbound, handlers) ---
	// Client send sites (COutPacket). Atlas serverbound structs are generically
	// named (Move/Attack/Damage) and collide with character/inventory serverbound
	// structs, so pkg="summon" restricts locateAtlasFile to summon/serverbound/
	// (writerName → SummonMove/SummonAttack/SummonDamage). v95 reference (PDB build).
	case "CVecCtrlSummoned::EndUpdateActive":
		return []candidate{{name: "Move", pkg: "summon", dir: csvpkg.DirServerbound, reportName: "SummonMoveHandle"}}
	case "CSummoned::TryDoingAttackManual":
		return []candidate{{name: "Attack", pkg: "summon", dir: csvpkg.DirServerbound, reportName: "SummonAttackHandle"}}
	case "CSummoned::SetDamaged":
		return []candidate{{name: "Damage", pkg: "summon", dir: csvpkg.DirServerbound, reportName: "SummonDamageHandle"}}

	// --- Combat: monster (serverbound) ---
	case "CMob::GenerateMovePath":
		// CSV: MOVE_LIFE — atlas MovementRequest (handle = "MonsterMovementHandle").
		return []candidate{{name: "MovementRequest", pkg: "monster", dir: csvpkg.DirServerbound}}
	case "CMob::SendDropPickUpRequest":
		// task-092 Cluster-D: MOB_DROP_PICKUP_REQUEST — atlas MobDropPickupRequest
		// (handle = "MobDropPickupRequest"). Two Encode4 (mobCrc, dropId).
		return []candidate{{name: "MobDropPickupRequest", pkg: "monster", dir: csvpkg.DirServerbound}}
	case "CMob::Update":
		// task-092 Cluster-A: CMob::Update (the mob per-frame tick) builds THREE
		// distinct serverbound COutPackets at distinct send-sites/opcodes:
		// FIELD_DAMAGE_MOB (mobCrc+damage), MOB_DAMAGE_MOB_FRIENDLY (mobCrc+charId
		// +attackerMobCrc), and MOB_SKILL_DELAY_END (mobCrc+skillId+lvl+value, absent
		// in v83). All three share this one fname; each is its own atlas struct.
		return []candidate{
			{name: "FieldDamageMob", pkg: "monster", dir: csvpkg.DirServerbound},
			// MOB_DAMAGE_MOB_FRIENDLY already exists as character/serverbound
			// MonsterDamageFriendly (3xEncode4: attacker, observer, attacked) — reuse it.
			{name: "MonsterDamageFriendly", pkg: "character", dir: csvpkg.DirServerbound},
			{name: "MobSkillDelayEnd", pkg: "monster", dir: csvpkg.DirServerbound},
		}
	case "CMob::SetDamagedByMob":
		// task-092 Cluster-A: MOB_DAMAGE_MOB — atlas MobDamageMob (handle =
		// "MobDamageMob"). 3xEncode4 + Encode1 + Encode4 + Encode1 + 2xEncode2.
		return []candidate{{name: "MobDamageMob", pkg: "monster", dir: csvpkg.DirServerbound}}
	case "CMob::TryFirstSelfDestruction":
		// task-092 Cluster-A: MONSTER_BOMB — atlas MonsterBomb (handle =
		// "MonsterBomb"). Single Encode4 (mobId). v84 sender unnamed in IDB.
		return []candidate{{name: "MonsterBomb", pkg: "monster", dir: csvpkg.DirServerbound}}
	case "CMob::UpdateTimeBomb":
		// task-092 Cluster-A: MOB_TIME_BOMB_END — atlas MobTimeBombEnd (handle =
		// "MobTimeBombEnd"). mobCrc + [boss x,y] + localUser x,y. Standalone only in
		// v95/jms (inlined into CMob::Update in v83/v84/v87).
		return []candidate{{name: "MobTimeBombEnd", pkg: "monster", dir: csvpkg.DirServerbound}}
	case "CUserLocal::SendBanMapByMobRequest":
		// task-092 Cluster-D: MOB_BANISH_PLAYER — atlas MobBanishPlayer
		// (handle = "MobBanishPlayer"), in character/serverbound. One Encode4
		// (mobTemplateId). v83/v84 senders are inlined into CUserLocal::Update.
		return []candidate{{name: "MobBanishPlayer", pkg: "character", dir: csvpkg.DirServerbound}}

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
		return []candidate{{name: "OperationAdd", pkg: "buddy", dir: csvpkg.DirServerbound, prefixSubOps: 1}}
	case "CField::SendAcceptFriendMsg":
		// Sub-op 2 (ACCEPT): Encode1(2) + Encode4(friendId).
		// Atlas struct: buddy/serverbound/operation_accept.go OperationAccept.
		return []candidate{{name: "OperationAccept", pkg: "buddy", dir: csvpkg.DirServerbound, prefixSubOps: 1}}
	case "CField::SendDeleteFriendMsg":
		// Sub-op 3 (DELETE): Encode1(3) + Encode4(friendId).
		// Atlas struct: buddy/serverbound/operation_delete.go OperationDelete.
		return []candidate{{name: "OperationDelete", pkg: "buddy", dir: csvpkg.DirServerbound, prefixSubOps: 1}}

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
		return []candidate{{name: "OperationAnswerInvite", pkg: "messenger", dir: csvpkg.DirServerbound, prefixName: "Operation", prefixPkg: "messenger"}}
	case "CUIMessenger::OnDestroy":
		// Sub-op 2 (LEAVE): Encode1(2) — client leaves/closes the messenger window.
		// Atlas struct: messenger/serverbound/operation.go Operation (op-byte dispatcher).
		// ⚠️ Operation only carries the mode byte; full op-family deferred to _pending.md
		// (OP-FAMILY-messenger-serverbound).
		return []candidate{{name: "Operation", pkg: "messenger", dir: csvpkg.DirServerbound}}
	case "CUIMessenger::SendInviteMsg":
		// Sub-op 3 (INVITE): Encode1(3) + EncodeStr(targetCharacter).
		// Atlas struct: messenger/serverbound/operation_invite.go OperationInvite.
		return []candidate{{name: "OperationInvite", pkg: "messenger", dir: csvpkg.DirServerbound, prefixName: "Operation", prefixPkg: "messenger"}}
	case "CFadeWnd::SendCloseMessage":
		// Sub-op 5 (DECLINE): Encode1(5) + EncodeStr(fromName) + EncodeStr(myName) + Encode1(0).
		// CFadeWnd handles multiple dialog types (type=0 → messenger decline, type=1 → buddy delete,
		// type=2/3 → miniroom, type=5 → guild); only type=0 maps to messenger OperationDeclineInvite.
		// Atlas struct: messenger/serverbound/operation_decline_invite.go OperationDeclineInvite.
		return []candidate{{name: "OperationDeclineInvite", pkg: "messenger", dir: csvpkg.DirServerbound, prefixName: "Operation", prefixPkg: "messenger"}}
	case "CUIMessenger::ProcessChat":
		// Sub-op 6 (CHAT): Encode1(6) + EncodeStr(chatLine — format "name : msg").
		// Atlas struct: messenger/serverbound/operation_chat.go OperationChat.
		return []candidate{{name: "OperationChat", pkg: "messenger", dir: csvpkg.DirServerbound, prefixName: "Operation", prefixPkg: "messenger"}}

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
		// CSV: MULTICHAT — atlas MultiChat (relocated to field/clientbound/multi.go, task-096 R-MARK).
		return []candidate{{name: "MultiChat", pkg: "field", dir: csvpkg.DirClientbound}}

	// CSV: WHISPER (clientbound) → CField::OnWhisper. A field-domain dispatcher
	// (relocated chat→field, task-096 R-MARK) that branches on a leading mode
	// byte to 8 sub-ops. Each atlas clientbound whisper struct is modelled via a
	// #-suffixed synthetic IDA entry so the pipeline produces one report per
	// struct (same pattern as messenger/buddy/note clientbound sub-ops). The op
	// row grades worst-of across all 8 #-suffix writers.
	case "CField::OnWhisper#Receive":
		// mode=0x12 (RECEIVE): DecodeStr(from) + Decode1(channel) + Decode1(gm) + DecodeStr(msg).
		// Atlas struct: field/clientbound/whisper.go WhisperReceive.
		return []candidate{{name: "WhisperReceive", pkg: "field", dir: csvpkg.DirClientbound}}
	case "CField::OnWhisper#SendResult":
		// mode=0x0A/0x8A (SEND_RESULT): DecodeStr(target) + Decode1(result).
		// Atlas struct: field/clientbound/whisper.go WhisperSendResult.
		return []candidate{{name: "WhisperSendResult", pkg: "field", dir: csvpkg.DirClientbound}}
	case "CField::OnWhisper#FindResultMap":
		// mode=0x09/0x48 sub=1 (FIND on map): DecodeStr(target) + Decode1(=1) + Decode4(mapId) +
		// [Decode4(x) + Decode4(y) when mode 0x09].
		// Atlas struct: field/clientbound/whisper.go WhisperFindResultMap.
		return []candidate{{name: "WhisperFindResultMap", pkg: "field", dir: csvpkg.DirClientbound}}
	case "CField::OnWhisper#FindResultCashShop":
		// mode=0x09/0x48 sub=2 (FIND in cash shop): DecodeStr(target) + Decode1(=2) + Decode4(-1).
		// Atlas struct: field/clientbound/whisper.go WhisperFindResultCashShop.
		return []candidate{{name: "WhisperFindResultCashShop", pkg: "field", dir: csvpkg.DirClientbound}}
	case "CField::OnWhisper#FindResultChannel":
		// mode=0x09/0x48 sub=3 (FIND on channel): DecodeStr(target) + Decode1(=3) + Decode4(channel).
		// Atlas struct: field/clientbound/whisper.go WhisperFindResultChannel.
		return []candidate{{name: "WhisperFindResultChannel", pkg: "field", dir: csvpkg.DirClientbound}}
	case "CField::OnWhisper#FindResultError":
		// mode=0x09/0x48 sub=0/else (FIND not found): DecodeStr(target) + Decode1(=0) + Decode4(0).
		// Atlas struct: field/clientbound/whisper.go WhisperFindResultError.
		return []candidate{{name: "WhisperFindResultError", pkg: "field", dir: csvpkg.DirClientbound}}
	case "CField::OnWhisper#Error":
		// mode=0x22 (ERROR): DecodeStr(target) + Decode1(whispersEnabled).
		// Atlas struct: field/clientbound/whisper.go WhisperError.
		return []candidate{{name: "WhisperError", pkg: "field", dir: csvpkg.DirClientbound}}
	case "CField::OnWhisper#Weather":
		// mode=0x92 (WEATHER): DecodeStr(from) + Decode1 + DecodeStr(msg).
		// Atlas struct: field/clientbound/whisper.go WhisperWeather.
		return []candidate{{name: "WhisperWeather", pkg: "field", dir: csvpkg.DirClientbound}}

	// CSV: SPOUSE_CHAT → CField::OnCoupleMessage (task-096 R-CB; field/clientbound/SpouseChat).
	// Per-version clientbound opcodes: v83 0x88, v84 0x8B, v87 0x90, v95 0x98. jms VERSION-ABSENT.
	// Dispatches on a leading mode byte (Decode1): mode=4 (own/sender message), mode=5 (partner
	// message). The export's flat read is the maximal mode-4 branch:
	//   Decode1(mode) + DecodeStr(sender) + Decode1(flag) + DecodeStr(chatText).
	// Atlas SpouseChat writes that representative shape flat.
	case "CField::OnCoupleMessage":
		return []candidate{{name: "SpouseChat", pkg: "field", dir: csvpkg.DirClientbound}}

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
		// CSV: GENERAL_CHAT — atlas General (relocated to field/serverbound/general.go, task-096 R-MARK).
		return []candidate{{name: "General", pkg: "field", dir: csvpkg.DirServerbound}}

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

	// WHISPER (serverbound) primary registry fname is CField::SendLocationWhisper
	// (the find-friend send-site); SendChatMsgWhisper is the chat-msg sibling. Both
	// emit the same WHISPER opcode and the same atlas chat.Whisper codec decodes
	// both. Per-version wire (IDA): mode byte + Encode4(get_update_time) [v87+/jms]
	// + EncodeStr(target) + optional EncodeStr(msg for mode==Chat).
	case "CField::SendLocationWhisper":
		return []candidate{{name: "Whisper", pkg: "chat", dir: csvpkg.DirServerbound}}

	// CSV: SPOUSE_CHAT (serverbound) → CUIStatusBar::SendCoupleMessage. The client
	// reads the partner name from the local marriage record and sends two strings
	// with NO leading mode byte / NO updateTime: EncodeStr(spouseName) +
	// EncodeStr(message). Per-version opcodes: v84 0x7B, v87 0x7F, v95 0x8E; jms
	// version-absent. Atlas struct field/serverbound/CoupleMessage (named to avoid
	// the qualified writer-name collision with field/clientbound/SpouseChat).
	case "CUIStatusBar::SendCoupleMessage":
		return []candidate{{name: "CoupleMessage", pkg: "field", dir: csvpkg.DirServerbound}}

	// --- Social: party (clientbound) ---
	// CSV: PARTY_OPERATION (clientbound, opcode 0x3E/62 in GMS v95) → CWvsContext::OnPartyResult
	// dispatches on a leading mode byte to 10+ sub-ops. Each atlas clientbound struct is modelled
	// via a #-suffixed synthetic IDA entry (same pattern as note/buddy/messenger sub-ops).
	case "CWvsContext::OnPartyResult#Created":
		// mode=8: client builds local party from CharacterData; server sends Decode4(partyId) +
		// Decode4(townPortalFromId) + Decode4(townPortalToId) + Decode2(x) + Decode2(y).
		// Atlas Created writes: mode(1) + partyId(4) + 2×EmptyMapId(8) + 2×short(4) = matches.
		return []candidate{{name: "Created", pkg: "party", dir: csvpkg.DirClientbound}}
	case "CWvsContext::OnPartyResult#Invite":
		// mode=4: Decode4(partyId) + DecodeStr(inviterName) + Decode4(inviterJob) + Decode4(inviterLevel) +
		// Decode1(autoJoinFlag). Atlas Invite writes: mode(1) + partyId(4) + name(str) + jobId(4) + level(4) + 0x00(1). ✓ fixed.
		// ⚠️ Tool-limitation: mode byte (dispatcher prefix) causes row-0 width mismatch.
		return []candidate{{name: "Invite", pkg: "party", dir: csvpkg.DirClientbound}}
	case "CWvsContext::OnPartyResult#Disband":
		// mode=12 + Decode1(isForced=0): no PARTYDATA follows when isForced branch not taken.
		// Atlas Disband writes: mode(1) + partyId(4) + targetId(4) + 0x00(1) + partyId(4).
		// ⚠️ Wire shape varies: IDA mode=12 reads Decode4(partyId)+Decode4(targetId)+Decode1(isForced)+
		// optional(Decode1+DecodeStr+PARTYDATA). The no-member path (isForced=0, different charId)
		// writes zero-fills party, consistent with Disband. Tool-limitation: branch-flattening.
		return []candidate{{name: "Disband", pkg: "party", dir: csvpkg.DirClientbound}}
	case "CWvsContext::OnPartyResult#Error":
		// mode=9,10,13,17,18,22,29,32–34,36 (error string pool nodes; no Decode calls beyond mode byte).
		// Atlas Error writes: mode(1) + name(str). ⚠️ Many error modes read no additional data from
		// packet; the name field is resolved server-side. Sub-op enum deferred to _pending.md.
		return []candidate{{name: "Error", pkg: "party", dir: csvpkg.DirClientbound}}
	case "CWvsContext::OnPartyResult#ChangeLeader":
		// mode=31: Decode4(newLeaderId) + Decode1(disconnected).
		// Atlas ChangeLeader writes: mode(1) + targetCharacterId(4) + disconnected(1). ✓
		return []candidate{{name: "ChangeLeader", pkg: "party", dir: csvpkg.DirClientbound}}
	case "CWvsContext::OnPartyResult#Join":
		// mode=15: Decode4(partyId) + DecodeStr(targetName) + PARTYDATA::Decode(378 bytes).
		// Atlas Join writes: mode(1) + partyId(4) + name(str) + WritePartyData(378 bytes). ✓ fixed.
		// ⚠️ Tool-limitation: mode byte (dispatcher prefix) causes row-0 width mismatch.
		return []candidate{{name: "Join", pkg: "party", dir: csvpkg.DirClientbound}}
	case "CWvsContext::OnPartyResult#Left":
		// mode=12 + Decode1(isForced=1): Decode4(partyId)+Decode4(targetId)+Decode1(1)+Decode1(isForced)+
		// DecodeStr(targetName)+PARTYDATA::Decode(378 bytes).
		// Atlas Left writes: mode(1) + partyId(4) + targetId(4) + 0x01(1) + forced(1) + name(str) +
		// WritePartyData(378 bytes). ✓ fixed.
		// ⚠️ Tool-limitation: mode byte (dispatcher prefix) causes row-0 width mismatch.
		return []candidate{{name: "Left", pkg: "party", dir: csvpkg.DirClientbound}}
	case "CWvsContext::OnPartyResult#Update":
		// mode=7/38: Decode4(partyId) + PARTYDATA::Decode(378 bytes).
		// Atlas Update writes: mode(1) + partyId(4) + WritePartyData(378 bytes). ✓ fixed.
		// ⚠️ Tool-limitation: mode byte (dispatcher prefix) causes row-0 width mismatch. HOT PATH.
		return []candidate{{name: "Update", pkg: "party", dir: csvpkg.DirClientbound}}
	// operation_body.go has no exported struct (it's pure helper functions); no candidate entry.

	// CSV: UPDATE_PARTYMEMBER_HP (clientbound, opcode 0xD6/214 in GMS v95) → CUserRemote::OnReceiveHP.
	// IDA reads: Decode4(hp) + Decode4(maxHp). The characterId is consumed upstream by
	// CUserPool::OnUserRemotePacket before OnReceiveHP is called (dispatcher-prefix pattern).
	// Atlas MemberHP writes: WriteInt(characterId) + WriteInt(hp) + WriteInt(maxHp) — characterId
	// is the dispatcher-level prefix; the packet body visible to OnReceiveHP is just hp+maxHp.
	// ✓ Wire is correct: server emits charId + hp + maxHp; dispatcher consumes charId then calls OnReceiveHP.
	// Hot-path: 4-variant byte-output sweep mandatory for any encoder fix.
	case "CUserRemote::OnReceiveHP":
		// CSV: UPDATE_PARTYMEMBER_HP — atlas MemberHP (party/clientbound/member_hp.go).
		return []candidate{{name: "MemberHP", pkg: "party", dir: csvpkg.DirClientbound}}

	// --- Social: party (serverbound) ---
	// CSV: PARTY_OPERATION (serverbound, opcode 0x91/145 in GMS v95). Multiple CField functions
	// share this opcode; each encodes a different op-byte at Encode1 position 0.
	case "CField::SendCreateNewPartyMsg":
		// op=1 (CREATE): Encode1(1). No further fields.
		// Atlas Operation (serverbound/operation.go) is the op-byte dispatcher; op=1 maps to create.
		// ⚠️ OP-FAMILY-party-serverbound: op-byte family deferred to _pending.md.
		return []candidate{{name: "Operation", pkg: "party", dir: csvpkg.DirServerbound}}
	case "CField::SendWithdrawPartyMsg":
		// op=2 (LEAVE): Encode1(2) + Encode1(0). Atlas serverbound has no separate Leave struct;
		// the op=2 sub-op is handled by the Operation dispatcher.
		// ⚠️ OP-FAMILY-party-serverbound: op-byte family deferred to _pending.md.
		// Note: Encode1(0) trailing byte not in atlas Operation (only carries op byte).
		return []candidate{{name: "Operation", pkg: "party", dir: csvpkg.DirServerbound}}
	case "CField::SendKickPartyMsg":
		// op=5 (EXPEL): Encode1(5) + Encode4(targetCharacterId).
		// Atlas OperationExpel writes: WriteInt(targetCharacterId). ✓ (op byte consumed by dispatcher)
		return []candidate{{name: "OperationExpel", pkg: "party", dir: csvpkg.DirServerbound, prefixName: "Operation", prefixPkg: "party"}}
	case "CField::SendChangePartyBossMsg":
		// op=6 (CHANGE_LEADER): Encode1(6) + Encode4(targetCharacterId).
		// Atlas OperationChangeLeader writes: WriteInt(targetCharacterId). ✓
		return []candidate{{name: "OperationChangeLeader", pkg: "party", dir: csvpkg.DirServerbound, prefixName: "Operation", prefixPkg: "party"}}
	case "CField::SendJoinPartyMsg":
		// op=4 (INVITE): Encode1(4) + EncodeStr(targetName).
		// Atlas OperationInvite writes: WriteAsciiString(name). ✓ (op byte consumed by dispatcher)
		return []candidate{{name: "OperationInvite", pkg: "party", dir: csvpkg.DirServerbound, prefixName: "Operation", prefixPkg: "party"}}

	// CSV: PARTY_RESULT (serverbound, opcode 0x92/146 in GMS v95) → reject/accept invite response.
	// IDA path: CUIFadeYesNo::OnMouseButton sends op=0x16 (accept) or op=0x17/0x18 (decline/ignore).
	// Atlas InviteReject writes: WriteByte(unk) + WriteAsciiString(from). ✓
	// (The unk byte maps to the response code; 'from' is the inviter name.)
	case "CWvsContext::OnPartyResult#InviteReject":
		// DENY_PARTY_REQUEST (serverbound) → atlas InviteReject (serverbound/invite_reject.go).
		// ⚠️ No direct IDA function found for DENY_PARTY_REQUEST; modelled via synthetic entry.
		// Wire: Encode1(declineCode) + EncodeStr(inviterName) — matches atlas layout.
		return []candidate{{name: "InviteReject", pkg: "party", dir: csvpkg.DirServerbound}}

	// CSV: PARTY_RESULT (serverbound, opcode 0x92/146 in GMS v95) also maps to OperationJoin.
	// SendJoinPartyMsg (op=4) triggers server → client PARTY_OPERATION mode 15 (join flow).
	// The direct serverbound join packet is op=3 or a separate opcode.
	case "CField::SendJoinPartyMsg#OperationJoin":
		// Synthetic entry: client sends party join accept with partyId after receiving an invite.
		// Atlas OperationJoin writes: WriteInt(partyId). ✓
		return []candidate{{name: "OperationJoin", pkg: "party", dir: csvpkg.DirServerbound, prefixName: "Operation", prefixPkg: "party"}}

	// --- Social: guild (clientbound) ---
	// CSV: GUILD_OPERATION (clientbound, opcode 0x041/65 in GMS v95) → CWvsContext::OnGuildResult
	// dispatches on a leading mode byte (Decode1) to 20+ sub-handlers. Each atlas struct is modelled
	// via a #-suffixed synthetic IDA entry. Both mode byte and sub-op enum drift deferred to _pending.md.
	case "CWvsContext::OnGuildResult":
		// Top-level dispatcher — reads mode byte only; sub-ops audited individually.
		// ⚠️ OP-FAMILY-guild-clientbound: op-byte family deferred to _pending.md.
		return []candidate{{name: "RequestAgreement", pkg: "guild", dir: csvpkg.DirClientbound}}
	case "CWvsContext::OnGuildResult#RequestAgreement":
		// case 3: Decode1(mode) + Decode4(partyId) + DecodeStr(leaderName) + DecodeStr(guildName).
		// Atlas RequestAgreement writes: mode(1) + partyId(4) + leaderName(str) + guildName(str). ✓
		return []candidate{{name: "RequestAgreement", pkg: "guild", dir: csvpkg.DirClientbound}}
	case "CWvsContext::OnGuildResult#Invite":
		// case 5: Decode1(mode) + Decode4(guildId) + DecodeStr(inviterName) + Decode4(v21) + Decode4(nSkillID).
		// Atlas Invite writes: mode(1) + guildId(4) + originatorName(str) — MISSING 2 trailing Decode4 fields.
		// ❌ Real wire bug: client reads 2 extra int32 fields that atlas does not send.
		// IDA address 0xa0d664 (case 5 body).
		return []candidate{{name: "Invite", pkg: "guild", dir: csvpkg.DirClientbound}}
	case "CWvsContext::OnGuildResult#ErrorMessage":
		// cases 30,33,35,37,38,40,42,43,44,47,50,54,58,61: mode byte only, no further reads.
		// Atlas ErrorMessage writes: mode(1). ✓ (mode-only arms)
		return []candidate{{name: "ErrorMessage", pkg: "guild", dir: csvpkg.DirClientbound}}
	case "CWvsContext::OnGuildResult#ErrorMessageWithTarget":
		// cases 55,56,57: Decode1(mode) + DecodeStr(targetName).
		// Atlas ErrorMessageWithTarget writes: mode(1) + target(str). ✓
		return []candidate{{name: "ErrorMessageWithTarget", pkg: "guild", dir: csvpkg.DirClientbound}}
	case "CWvsContext::OnGuildResult#EmblemChange":
		// case 69: Decode1(mode) + Decode4(guildId) + Decode2(nMarkBg) + Decode1(nMarkBgColor) + Decode2(nMark) + Decode1(nMarkColor).
		// Atlas EmblemChange writes: mode(1) + guildId(4) + logoBackground(2) + logoBackgroundColor(1) + logo(2) + logoColor(1). ✓
		return []candidate{{name: "EmblemChange", pkg: "guild", dir: csvpkg.DirClientbound}}
	case "CWvsContext::OnGuildResult#MemberStatusUpdate":
		// case 63: Decode1(mode) + Decode4(guildId) + Decode4(charId) + Decode1(online).
		// Atlas MemberStatusUpdate writes: mode(1) + guildId(4) + characterId(4) + WriteBool(online)(1). ✓
		return []candidate{{name: "MemberStatusUpdate", pkg: "guild", dir: csvpkg.DirClientbound}}
	case "CWvsContext::OnGuildResult#MemberTitleUpdate":
		// case 66: Decode1(mode) + Decode4(guildId) + Decode4(charId) + Decode1(newGrade).
		// Atlas MemberTitleUpdate writes: mode(1) + guildId(4) + characterId(4) + title(1). ✓
		return []candidate{{name: "MemberTitleUpdate", pkg: "guild", dir: csvpkg.DirClientbound}}
	case "CWvsContext::OnGuildResult#NoticeChange":
		// case 71: Decode1(mode) + Decode4(guildId) + DecodeStr(notice).
		// Atlas NoticeChange writes: mode(1) + guildId(4) + notice(str). ✓
		return []candidate{{name: "NoticeChange", pkg: "guild", dir: csvpkg.DirClientbound}}
	case "CWvsContext::OnGuildResult#MemberLeft":
		// case 46: Decode1(mode) + Decode4(guildId) + Decode4(charId) + DecodeStr(name).
		// Atlas MemberLeft writes: mode(1) + guildId(4) + characterId(4) + name(str). ✓
		return []candidate{{name: "MemberLeft", pkg: "guild", dir: csvpkg.DirClientbound}}
	case "CWvsContext::OnGuildResult#MemberExpel":
		// case 49: Decode1(mode) + Decode4(guildId) + Decode4(charId) + DecodeStr(name).
		// Atlas MemberExpel writes: mode(1) + guildId(4) + characterId(4) + name(str). ✓
		return []candidate{{name: "MemberExpel", pkg: "guild", dir: csvpkg.DirClientbound}}
	case "CWvsContext::OnGuildResult#MemberJoined":
		// case 41: Decode1(mode) + Decode4(guildId) + Decode4(charId) + GUILDMEMBER::Decode(37 raw bytes).
		// GUILDMEMBER::Decode@0x4f2b40 → DecodeBuffer(37): name[13]+job[4]+level[4]+title[4]+online[4]+sig[4]+allianceTitle[4].
		// Atlas MemberJoined writes: mode(1) + guildId(4) + characterId(4) + GuildMember.Encode(37 bytes). ✓ on wire bytes.
		// ⚠️ Tool-limitation: DecodeBuf (single call, 37 bytes) vs atlas 7 explicit writes — tool may report ❌.
		return []candidate{{name: "MemberJoined", pkg: "guild", dir: csvpkg.DirClientbound}}
	case "CWvsContext::OnGuildResult#TitleChange":
		// case 64: Decode1(mode) + Decode4(guildId) + 5×DecodeStr(titles).
		// Atlas TitleChange writes: mode(1) + guildId(4) + 5×WriteAsciiString(title). ✓
		return []candidate{{name: "TitleChange", pkg: "guild", dir: csvpkg.DirClientbound}}
	case "CWvsContext::OnGuildResult#Disband":
		// case 52: Decode1(mode) + Decode4(guildId).
		// Atlas Disband writes: mode(1) + guildId(4). ✓
		return []candidate{{name: "Disband", pkg: "guild", dir: csvpkg.DirClientbound}}
	case "CWvsContext::OnGuildResult#CapacityChange":
		// case 60: Decode1(mode) + Decode4(guildId) + Decode1(nMaxMemberNum).
		// Atlas CapacityChange writes: mode(1) + guildId(4) + WriteInt(capacity)(4).
		// ❌ Real wire bug: IDA reads Decode1 (1 byte) but atlas emits WriteInt (4 bytes).
		return []candidate{{name: "CapacityChange", pkg: "guild", dir: csvpkg.DirClientbound}}
	case "CWvsContext::OnGuildResult#Info":
		// Info packet (sub-op 0x1A=26 in GUILD_OPERATION).
		// Wire per GUILDDATA::Decode@0x4fb760: id(4)+name(str)+5×str(titles)+byte(count)+
		// buf(count×4 charIds)+buf(count×37 members)+int(capacity)+short(logoBg)+byte(logoBgColor)+
		// short(logo)+byte(logoColor)+str(notice)+int(points)+int(allianceId).
		// Atlas Info.Encode writes: WriteByte(0x1A)+WriteBool(inGuild)+same field sequence.
		// ⚠️ Tool-limitation: packed DecodeBuf for charIds and members vs atlas explicit writes;
		// flat analyzer cannot model member-loop expansion. Wire is ✓ on bytes.
		return []candidate{{name: "Info", pkg: "guild", dir: csvpkg.DirClientbound}}

	// CSV: GUILD_NAME_CHANGED (clientbound, opcode 0x0CA/202 in GMS v95) → CUserRemote::OnGuildNameChanged.
	// characterId is consumed upstream by CUserPool::OnUserRemotePacket (dispatcher-prefix pattern).
	// OnGuildNameChanged reads: DecodeStr(newGuildName). Atlas ForeignNameChanged writes: WriteInt(charId)+WriteAsciiString(name).
	// ✓ Wire correct: server sends charId+name; dispatcher strips charId then calls OnGuildNameChanged.
	case "CUserRemote::OnGuildNameChanged":
		return []candidate{{name: "ForeignNameChanged", pkg: "guild", dir: csvpkg.DirClientbound}}

	// CSV: GUILD_MARK_CHANGED (clientbound, opcode 0x0CB/203 in GMS v95) → CUserRemote::OnGuildMarkChanged.
	// characterId is consumed upstream (dispatcher-prefix pattern). Reads: Decode2+Decode1+Decode2+Decode1.
	// Atlas ForeignEmblemChanged writes: WriteInt(charId)+logoBackground(2)+logoBackgroundColor(1)+logo(2)+logoColor(1). ✓
	case "CUserRemote::OnGuildMarkChanged":
		return []candidate{{name: "ForeignEmblemChanged", pkg: "guild", dir: csvpkg.DirClientbound}}

	// --- Social: guild BBS (clientbound) ---
	// CSV: GUILD_BBS_PACKET (clientbound, opcode 0x03B/59 in GMS v95) → CWvsContext::OnGuildBBSPacket →
	// CUIGuildBBS::OnGuildBBSPacket dispatches on (Decode1 - 6): 0=list, 1=view, 2=not-found.
	case "CWvsContext::OnGuildBBSPacket":
		// Top-level dispatcher — op-byte only (delegates to CUIGuildBBS::OnGuildBBSPacket).
		// ⚠️ OP-FAMILY-guild-bbs-clientbound: deferred to _pending.md.
		return []candidate{{name: "BBSThreadList", pkg: "guild", dir: csvpkg.DirClientbound}}
	case "CUIGuildBBS::OnGuildBBSPacket#BBSThreadList":
		// (Decode1-6)=0 → OnLoadListResult: mode(1)+hasNotice(1)+[noticeFields if set]+totalCount(4)+pageCount(4)+entries.
		// Atlas BBSThreadList.Encode: WriteByte(0x06)+hasNotice+[noticeFields]+totalCount(4)+[page of entries].
		// ⚠️ Tool-limitation: conditional notice block + loop body → flat analyzer FP. Wire ✓ on bytes.
		return []candidate{{name: "BBSThreadList", pkg: "guild", dir: csvpkg.DirClientbound}}
	case "CUIGuildBBS::OnGuildBBSPacket#BBSThread":
		// (Decode1-6)=1 → OnViewEntryResult: mode(1)+id(4)+charId(4)+date(8buf)+title(str)+text(str)+emoticon(4)+replyCount(4)+replies.
		// Atlas BBSThread.Encode: WriteByte(0x07)+id(4)+charId(4)+int64(date)+title+text+emoticon(4)+replyCount(4)+replies. ✓
		// ⚠️ Tool-limitation: DecodeBuf(8) date vs WriteInt64 — same 8 bytes but different op classification.
		return []candidate{{name: "BBSThread", pkg: "guild", dir: csvpkg.DirClientbound}}

	// --- Social: guild (serverbound) ---
	// CSV: GUILD_OPERATION (serverbound, opcode 0x07E/126 in GMS v95) → dispatcher reads op byte.
	// Multiple CField functions share this opcode.
	case "CUIFadeYesNo::OnButtonClicked":
		// Op-byte dispatcher for GUILD_OPERATION serverbound.
		// ⚠️ OP-FAMILY-guild-serverbound: op-byte family deferred to _pending.md.
		return []candidate{{name: "Operation", pkg: "guild", dir: csvpkg.DirServerbound}}
	case "CField::InputGuildName":
		// Sub-op: RequestCreate — Encode1(op) + EncodeStr(name). Atlas RequestCreate writes: WriteAsciiString(name). ✓ (op consumed by dispatcher)
		return []candidate{{name: "RequestCreate", pkg: "guild", dir: csvpkg.DirServerbound, prefixName: "Operation", prefixPkg: "guild"}}
	case "CField::SendCreateGuildAgreeMsg":
		// Sub-op: AgreementResponse — Encode1(op) + Encode1(agreed). Atlas AgreementResponse writes: WriteInt(unk)+WriteBool(agreed). ❌ wire mismatch — extra Encode4 unk.
		return []candidate{{name: "AgreementResponse", pkg: "guild", dir: csvpkg.DirServerbound, prefixName: "Operation", prefixPkg: "guild"}}
	case "CField::SendSetGuildMarkMsg":
		// Sub-op: SetEmblem — Encode1(op) + Encode2(logoBg) + Encode1(logoBgColor) + Encode2(logo) + Encode1(logoColor). Atlas SetEmblem writes same fields. ✓
		return []candidate{{name: "SetEmblem", pkg: "guild", dir: csvpkg.DirServerbound, prefixName: "Operation", prefixPkg: "guild"}}
	case "CField::SendInviteGuildMsg":
		// Sub-op: InviteRequest — Encode1(op) + EncodeStr(target). Atlas InviteRequest writes: WriteAsciiString(target). ✓ (op consumed by dispatcher)
		return []candidate{{name: "InviteRequest", pkg: "guild", dir: csvpkg.DirServerbound, prefixName: "Operation", prefixPkg: "guild"}}
	case "CField::SendWithdrawGuildMsg":
		// Sub-op: Withdraw — Encode1(op) + Encode4(charId) + EncodeStr(name). Atlas Withdraw writes: WriteInt(cid)+WriteAsciiString(name). ✓
		return []candidate{{name: "Withdraw", pkg: "guild", dir: csvpkg.DirServerbound, prefixName: "Operation", prefixPkg: "guild"}}
	case "CField::SendKickGuildMsg":
		// Sub-op: Kick — Encode1(op) + Encode4(charId) + EncodeStr(name). Atlas Kick writes: WriteInt(cid)+WriteAsciiString(name). ✓
		return []candidate{{name: "Kick", pkg: "guild", dir: csvpkg.DirServerbound, prefixName: "Operation", prefixPkg: "guild"}}
	case "CField::SendSetGuildNoticeMsg":
		// Sub-op: SetNotice — Encode1(op) + EncodeStr(notice). Atlas SetNotice writes: WriteAsciiString(notice). ✓
		return []candidate{{name: "SetNotice", pkg: "guild", dir: csvpkg.DirServerbound, prefixName: "Operation", prefixPkg: "guild"}}
	case "CTabGuildAlliance::OnGradeChange":
		// Sub-op: SetMemberTitle — Encode1(op) + Encode4(targetId) + Encode1(newTitle). Atlas SetMemberTitle writes: WriteInt(targetId)+WriteByte(newTitle). ✓
		return []candidate{{name: "SetMemberTitle", pkg: "guild", dir: csvpkg.DirServerbound, prefixName: "Operation", prefixPkg: "guild"}}
	case "CWvsContext::SendSetGuildTitleNames":
		// Sub-op: SetTitleNames — Encode1(op) + 5×EncodeStr(title). Atlas SetTitleNames writes: 5×WriteAsciiString. ✓
		return []candidate{{name: "SetTitleNames", pkg: "guild", dir: csvpkg.DirServerbound, prefixName: "Operation", prefixPkg: "guild"}}
	case "CWvsContext::OnGuildResult#AgreementResponse":
		// CLIENTBOUND guild-creation agreement broadcast to party members:
		// [mode, partyId, leaderName, guildName]. This is the REQUEST shown in the
		// agree dialog, NOT the serverbound member reply (CField::SendCreateGuildAgreeMsg
		// → the separate serverbound AgreementResponse). Atlas's clientbound
		// RequestAgreement writes exactly [mode, partyId, leaderName, guildName], and
		// the mode byte is the struct's own first field (no Operation wrapper).
		return []candidate{{name: "RequestAgreement", pkg: "guild", dir: csvpkg.DirClientbound}}
	case "CWvsContext::SendGuildJoinMsg":
		// Synthetic entry for Join serverbound (guild join after invitation accepted).
		// Atlas Join writes: WriteInt(guildId)+WriteInt(characterId). ✓
		return []candidate{{name: "Join", pkg: "guild", dir: csvpkg.DirServerbound, prefixName: "Operation", prefixPkg: "guild"}}

	// CSV: DENY_GUILD_REQUEST (serverbound, opcode 0x07F/127 in GMS v95).
	// Client sends decline code + inviter name when rejecting guild invite.
	// Atlas InviteReject writes: WriteByte(unk)+WriteAsciiString(from). ✓
	case "CFadeWnd::SendCloseMessage#DenyGuildRequest":
		// DENY_GUILD_REQUEST — atlas InviteReject (serverbound/invite_reject.go).
		return []candidate{{name: "InviteReject", pkg: "guild", dir: csvpkg.DirServerbound}}

	// --- Social: guild BBS (serverbound) ---
	// CSV: BBS_OPERATION (serverbound, opcode 0x09B/155 in GMS v95) → dispatcher reads op byte.
	case "CUIGuildBBS::SendLoadListRequest":
		// BBS list request: Encode1(op) + Encode4(startIndex). Atlas BBSListThreads writes: WriteInt(startIndex). ✓ (op consumed by BBS dispatcher)
		return []candidate{{name: "BBSListThreads", pkg: "guild", dir: csvpkg.DirServerbound, prefixName: "BBS", prefixPkg: "guild"}}
	case "CUIGuildBBS::SendViewEntryRequest":
		// BBS view entry: Encode1(op) + Encode4(threadId). Atlas BBSDisplayThread writes: WriteInt(threadId). ✓
		return []candidate{{name: "BBSDisplayThread", pkg: "guild", dir: csvpkg.DirServerbound, prefixName: "BBS", prefixPkg: "guild"}}
	case "CUIGuildBBS::OnCommentDelete":
		// BBS delete reply: Encode1(op) + Encode4(threadId) + Encode4(replyId). Atlas BBSDeleteReply writes: WriteInt(threadId)+WriteInt(replyId). ✓
		return []candidate{{name: "BBSDeleteReply", pkg: "guild", dir: csvpkg.DirServerbound, prefixName: "BBS", prefixPkg: "guild"}}
	case "CUIGuildBBS::OnRegister":
		// BBS create/edit: Encode1(op) + Encode1(modify) + [if modify: Encode4(threadId)] + Encode1(notice) + EncodeStr(title) + EncodeStr(msg) + Encode4(emoticon).
		// Atlas BBSCreateOrEditThread writes same fields. ✓
		return []candidate{{name: "BBSCreateOrEditThread", pkg: "guild", dir: csvpkg.DirServerbound, prefixName: "BBS", prefixPkg: "guild"}}
	case "CUIGuildBBS::OnComment":
		// BBS reply thread: Encode1(op) + Encode4(threadId) + EncodeStr(message). Atlas BBSReplyThread writes: WriteInt(threadId)+WriteAsciiString(message). ✓
		return []candidate{{name: "BBSReplyThread", pkg: "guild", dir: csvpkg.DirServerbound, prefixName: "BBS", prefixPkg: "guild"}}
	case "CUIGuildBBS::OnDelete":
		// BBS delete thread: Encode1(op) + Encode4(threadId). Atlas BBSDeleteThread writes: WriteInt(threadId). ✓
		return []candidate{{name: "BBSDeleteThread", pkg: "guild", dir: csvpkg.DirServerbound, prefixName: "BBS", prefixPkg: "guild"}}
	// --- storage sub-domain (task-067) ---
	// Clientbound CTrunkDlg::OnPacket is a mode-dispatched writer; use synthetic
	// #-suffix FNames (one per atlas wire shape) to disambiguate.
	case "CTrunkDlg::OnPacket#Show":
		return []candidate{{name: "Show", dir: csvpkg.DirClientbound, pkg: "storage"}}
	case "CTrunkDlg::OnPacket#UpdateAssets":
		return []candidate{{name: "UpdateAssets", dir: csvpkg.DirClientbound, pkg: "storage"}}
	case "CTrunkDlg::OnPacket#UpdateMeso":
		return []candidate{{name: "UpdateMeso", dir: csvpkg.DirClientbound, pkg: "storage"}}
	case "CTrunkDlg::OnPacket#ErrorSimple":
		return []candidate{{name: "ErrorSimple", dir: csvpkg.DirClientbound, pkg: "storage"}}
	case "CTrunkDlg::OnPacket#ErrorMessage":
		return []candidate{{name: "ErrorMessage", dir: csvpkg.DirClientbound, pkg: "storage"}}
	// Serverbound CTrunkDlg senders.
	case "CTrunkDlg::SendGetItemRequest":
		return []candidate{{name: "OperationRetrieveAsset", dir: csvpkg.DirServerbound, pkg: "storage"}}
	case "CTrunkDlg::SendPutItemRequest":
		return []candidate{{name: "OperationStoreAsset", dir: csvpkg.DirServerbound, pkg: "storage"}}
	case "CTrunkDlg::SendGetMoneyRequest":
		return []candidate{{name: "OperationMeso", dir: csvpkg.DirServerbound, pkg: "storage"}}
	case "CTrunkDlg::OnPacket#Operation":
		return []candidate{{name: "Operation", dir: csvpkg.DirServerbound, pkg: "storage"}}
	// --- inventory sub-domain (task-067) ---
	// Clientbound CWvsContext::OnInventoryOperation is a mode-dispatched reader;
	// use synthetic #-suffix FNames (one per atlas wire shape) to disambiguate.
	case "CWvsContext::OnInventoryOperation#QuantityUpdate":
		return []candidate{{name: "QuantityUpdate", dir: csvpkg.DirClientbound, pkg: "inventory"}}
	case "CWvsContext::OnInventoryOperation#ChangeMove":
		return []candidate{{name: "ChangeMove", dir: csvpkg.DirClientbound, pkg: "inventory"}}
	case "CWvsContext::OnInventoryOperation#Remove":
		return []candidate{{name: "Remove", dir: csvpkg.DirClientbound, pkg: "inventory"}}
	case "CWvsContext::OnInventoryOperation#Add":
		return []candidate{{name: "Add", dir: csvpkg.DirClientbound, pkg: "inventory"}}
	case "CWvsContext::OnInventoryOperation#ChangeBatch":
		return []candidate{{name: "ChangeBatch", dir: csvpkg.DirClientbound, pkg: "inventory"}}
	case "CWvsContext::OnGatherItemResult":
		return []candidate{{name: "CompartmentMerge", dir: csvpkg.DirClientbound, pkg: "inventory"}}
	case "CWvsContext::OnSortItemResult":
		return []candidate{{name: "CompartmentSort", dir: csvpkg.DirClientbound, pkg: "inventory"}}
	// Serverbound CWvsContext senders.
	case "CWvsContext::SendChangeSlotPositionRequest":
		return []candidate{{name: "Move", dir: csvpkg.DirServerbound, pkg: "inventory"}}
	case "CWvsContext::SendGatherItemRequest":
		return []candidate{{name: "CompartmentMergeRequest", dir: csvpkg.DirServerbound, pkg: "inventory"}}
	case "CWvsContext::SendSortItemRequest":
		return []candidate{{name: "CompartmentSortRequest", dir: csvpkg.DirServerbound, pkg: "inventory"}}
	case "CWvsContext::SendStatChangeItemUseRequest":
		return []candidate{{name: "ItemUse", dir: csvpkg.DirServerbound, pkg: "inventory"}}
	case "CWvsContext::SendUpgradeItemUseRequest":
		return []candidate{{name: "ScrollUse", dir: csvpkg.DirServerbound, pkg: "inventory"}}
	// --- interaction sub-domain (task-067) ---
	// NOTE: the interaction serverbound dispatcher struct is also named `Operation`
	// (collides with storage's CTrunkDlg `Operation` under the flat report layout;
	// storage wins the dedup). The interaction dispatcher is documented in
	// docs/packets/ida-exports/_pending.md -> "OP-FAMILY-interaction" instead of a
	// separate report. Sub-op senders below each map 1:1 to an atlas wire shape.
	case "CMiniRoomBaseDlg::CheckAndSendChat":
		return []candidate{{name: "OperationChat", dir: csvpkg.DirServerbound, pkg: "interaction"}}
	case "CField::SendInviteTradingRoomMsg":
		return []candidate{{name: "OperationInvite", dir: csvpkg.DirServerbound, pkg: "interaction"}}
	case "CField::AddBlackList":
		return []candidate{{name: "OperationFieldAddToBlackList", dir: csvpkg.DirServerbound, pkg: "interaction"}}
	case "CField::DeleteBlackList":
		return []candidate{{name: "OperationFieldRemoveFromBlackList", dir: csvpkg.DirServerbound, pkg: "interaction"}}
	case "CPersonalShopDlg::OnClickBanButton":
		return []candidate{{name: "OperationPersonalStoreAddToBlackList", dir: csvpkg.DirServerbound, pkg: "interaction"}}
	case "CPersonalShopDlg::DeliverBlackList":
		return []candidate{{name: "OperationPersonalStoreSetBlackList", dir: csvpkg.DirServerbound, pkg: "interaction"}}
	case "CTradingRoomDlg::PutItem":
		return []candidate{{name: "OperationTradePutItem", dir: csvpkg.DirServerbound, pkg: "interaction"}}
	case "CTradingRoomDlg::PutMoney":
		return []candidate{{name: "OperationTradeAddMeso", dir: csvpkg.DirServerbound, pkg: "interaction"}}
	case "CTradingRoomDlg::Trade":
		return []candidate{{name: "OperationTradeConfirm", dir: csvpkg.DirServerbound, pkg: "interaction"}}
	case "CCashTradingRoomDlg::Trade":
		return []candidate{{name: "OperationTransaction", dir: csvpkg.DirServerbound, pkg: "interaction"}}
	case "CPersonalShopDlg::PutItem":
		return []candidate{{name: "OperationPersonalStorePutItem", dir: csvpkg.DirServerbound, pkg: "interaction"}}
	case "CPersonalShopDlg::BuyItem":
		return []candidate{{name: "OperationPersonalStoreBuy", dir: csvpkg.DirServerbound, pkg: "interaction"}}
	case "CPersonalShopDlg::MoveItemToInventory":
		return []candidate{{name: "OperationPersonalStoreRemoveItem", dir: csvpkg.DirServerbound, pkg: "interaction"}}
	case "CMemoryGameDlg::SendTurnUpCard":
		return []candidate{{name: "OperationMemoryGameFlipCard", dir: csvpkg.DirServerbound, pkg: "interaction"}}
	case "CMemoryGameDlg::OnTieRequest":
		return []candidate{{name: "OperationMemoryGameTieAnswer", dir: csvpkg.DirServerbound, pkg: "interaction"}}
	case "COmokDlg::PutStoneChecker":
		return []candidate{{name: "OperationMemoryGameMoveStone", dir: csvpkg.DirServerbound, pkg: "interaction"}}
	case "COmokDlg::OnRetreatRequest":
		return []candidate{{name: "OperationMemoryGameRetreatAnswer", dir: csvpkg.DirServerbound, pkg: "interaction"}}
	// Entrusted-merchant sub-ops (share CPersonalShopDlg senders w/ different op-bytes).
	case "CEntrustedShopDlg::AddBlackList":
		return []candidate{{name: "OperationMerchantAddToBlackList", dir: csvpkg.DirServerbound, pkg: "interaction"}}
	case "CEntrustedShopDlg::DeleteBlackList":
		return []candidate{{name: "OperationMerchantRemoveFromBlackList", dir: csvpkg.DirServerbound, pkg: "interaction"}}
	case "CPersonalShopDlg::PutItem#Merchant":
		return []candidate{{name: "OperationMerchantPutItem", dir: csvpkg.DirServerbound, pkg: "interaction"}}
	case "CPersonalShopDlg::BuyItem#Merchant":
		return []candidate{{name: "OperationMerchantBuy", dir: csvpkg.DirServerbound, pkg: "interaction"}}
	case "CPersonalShopDlg::MoveItemToInventory#Merchant":
		return []candidate{{name: "OperationMerchantRemoveItem", dir: csvpkg.DirServerbound, pkg: "interaction"}}
	// Clientbound CMiniRoomBaseDlg::OnPacketBase is a mode-dispatched reader;
	// synthetic #-suffix FNames disambiguate the per-mode atlas wire shapes.
	case "CMiniRoomBaseDlg::OnPacketBase#Invite":
		return []candidate{{name: "InteractionInvite", dir: csvpkg.DirClientbound, pkg: "interaction"}}
	case "CMiniRoomBaseDlg::OnPacketBase#InviteResult":
		return []candidate{{name: "InteractionInviteResult", dir: csvpkg.DirClientbound, pkg: "interaction"}}
	case "CMiniRoomBaseDlg::OnPacketBase#Enter":
		return []candidate{{name: "InteractionEnter", dir: csvpkg.DirClientbound, pkg: "interaction"}}
	case "CMiniRoomBaseDlg::OnPacketBase#EnterResultSuccess":
		return []candidate{{name: "InteractionEnterResultSuccess", dir: csvpkg.DirClientbound, pkg: "interaction"}}
	case "CMiniRoomBaseDlg::OnPacketBase#EnterResultError":
		return []candidate{{name: "InteractionEnterResultError", dir: csvpkg.DirClientbound, pkg: "interaction"}}
	case "CMiniRoomBaseDlg::OnPacketBase#Chat":
		return []candidate{{name: "InteractionChat", dir: csvpkg.DirClientbound, pkg: "interaction"}}
	case "CMiniRoomBaseDlg::OnPacketBase#Leave":
		return []candidate{{name: "InteractionLeave", dir: csvpkg.DirClientbound, pkg: "interaction"}}
	case "CEntrustedShopDlg::OnRefresh#UpdateMerchant":
		// UPDATE_MERCHANT (mode 25) is the hired-merchant shop refresh. The
		// dispatcher's default case virtual-dispatches into the concrete dialog;
		// for the hired merchant that is CEntrustedShopDlg::OnRefresh, which reads
		// Decode4(meso=m_nMoney) then chains into CPersonalShopDlg::OnRefresh
		// (count byte + per-item perBundle/quantity/price/asset). v95 0x51cc30,
		// v83 0x518852, v84 0x5218ca, v87 0x53b2fc — all IDA-verified to read the
		// meso prefix before the personal-shop item loop.
		return []candidate{{name: "InteractionUpdateMerchant", dir: csvpkg.DirClientbound, pkg: "interaction"}}
	// --- cash sub-domain (task-067, Phase 1d) ---
	// Clientbound: QueryResult routes through CCashShop::OnQueryCashResult (opcode 0x17F),
	// a SEPARATE dispatcher from OnCashItemResult.
	case "CCashShop::OnQueryCashResult":
		return []candidate{{name: "QueryResult", dir: csvpkg.DirClientbound, pkg: "cash"}}
	// Clientbound CCashShop::OnCashItemResult is a mode-dispatched reader (op-bytes 0x54-0xBC);
	// synthetic #-suffix FNames map each CashShopOperation result struct to its OnCashItemRes* sub-handler.
	case "CCashShop::OnCashItemResult#CashShopInventory":
		return []candidate{{name: "CashShopInventory", dir: csvpkg.DirClientbound, pkg: "cash"}}
	case "CCashShop::OnCashItemResult#WishList":
		return []candidate{{name: "WishList", dir: csvpkg.DirClientbound, pkg: "cash"}}
	case "CCashShop::OnCashItemResult#InventoryCapacitySuccess":
		return []candidate{{name: "InventoryCapacitySuccess", dir: csvpkg.DirClientbound, pkg: "cash"}}
	case "CCashShop::OnCashItemResult#InventoryCapacityFailed":
		return []candidate{{name: "InventoryCapacityFailed", dir: csvpkg.DirClientbound, pkg: "cash"}}
	case "CCashShop::OnCashItemResult#OperationError":
		return []candidate{{name: "OperationError", dir: csvpkg.DirClientbound, pkg: "cash"}}
	case "CCashShop::OnCashItemResult#CashShopPurchaseSuccess":
		return []candidate{{name: "CashShopPurchaseSuccess", dir: csvpkg.DirClientbound, pkg: "cash"}}
	case "CCashShop::OnCashItemResult#CashItemMovedToCashInventory":
		// case 0x79 MOVE_S_TO_L_DONE (OnCashItemResMoveStoLDone): mode + 55-byte GW_CashItemInfo.
		return []candidate{{name: "CashItemMovedToCashInventory", dir: csvpkg.DirClientbound, pkg: "cash"}}
	case "CCashShop::OnCashItemResult#CashItemMovedToInventory":
		// case 0x77 MOVE_L_TO_S_DONE (OnCashItemResMoveLtoSDone): mode + Decode2(slot) + GW_ItemSlotBase (model.Asset).
		return []candidate{{name: "CashItemMovedToInventory", dir: csvpkg.DirClientbound, pkg: "cash"}}
	// Serverbound CCashShop senders (op-byte owned by the ShopOperation dispatcher; bodies below).
	case "CCashShop::TrySendQueryCashRequest":
		return []candidate{{name: "CheckWallet", dir: csvpkg.DirServerbound, pkg: "cash"}}
	case "CCashShop::OnBuy":
		return []candidate{{name: "ShopOperationBuy", dir: csvpkg.DirServerbound, pkg: "cash"}}
	case "CCashShop::OnBuyNormal":
		return []candidate{{name: "ShopOperationBuyNormal", dir: csvpkg.DirServerbound, pkg: "cash"}}
	case "CCashShop::OnBuyPackage":
		return []candidate{{name: "ShopOperationBuyPackage", dir: csvpkg.DirServerbound, pkg: "cash"}}
	case "CCashShop::OnBuyCouple":
		return []candidate{{name: "ShopOperationBuyCouple", dir: csvpkg.DirServerbound, pkg: "cash"}}
	case "CCashShop::OnBuyFriendship":
		return []candidate{{name: "ShopOperationBuyFriendship", dir: csvpkg.DirServerbound, pkg: "cash"}}
	case "CCashShop::SendBuyNameChangeItemPacket":
		return []candidate{{name: "ShopOperationBuyNameChange", dir: csvpkg.DirServerbound, pkg: "cash"}}
	case "CCashShop::SendBuyTransferWorldItemPacket":
		return []candidate{{name: "ShopOperationBuyWorldTransfer", dir: csvpkg.DirServerbound, pkg: "cash"}}
	case "CCashShop::OnEnableEquipSlotExt":
		return []candidate{{name: "ShopOperationEnableEquipSlot", dir: csvpkg.DirServerbound, pkg: "cash"}}
	case "CCashShop::RequestCashPurchaseRecord":
		return []candidate{{name: "ShopOperationGetPurchaseRecord", dir: csvpkg.DirServerbound, pkg: "cash"}}
	case "CCashShop::SendGiftsPacket":
		return []candidate{{name: "ShopOperationGift", dir: csvpkg.DirServerbound, pkg: "cash"}}
	case "CCashShop::OnIncCharacterSlotCount":
		return []candidate{{name: "ShopOperationIncreaseCharacterSlot", dir: csvpkg.DirServerbound, pkg: "cash"}}
	case "CCashShop::OnBuySlotInc":
		return []candidate{{name: "ShopOperationIncreaseInventory", dir: csvpkg.DirServerbound, pkg: "cash"}}
	case "CCashShop::OnIncTrunkCount":
		return []candidate{{name: "ShopOperationIncreaseStorage", dir: csvpkg.DirServerbound, pkg: "cash"}}
	case "CCashShop::OnMoveCashItemLtoS":
		return []candidate{{name: "ShopOperationMoveFromCashInventory", dir: csvpkg.DirServerbound, pkg: "cash"}}
	case "CCashShop::OnMoveCashItemStoL":
		return []candidate{{name: "ShopOperationMoveToCashInventory", dir: csvpkg.DirServerbound, pkg: "cash"}}
	case "CCashShop::OnRebateLockerItem":
		return []candidate{{name: "ShopOperationRebateLockerItem", dir: csvpkg.DirServerbound, pkg: "cash"}}
	case "CCashShop::OnSetWish":
		return []candidate{{name: "ShopOperationSetWishlist", dir: csvpkg.DirServerbound, pkg: "cash"}}

	// --- World: portal (serverbound) ---
	// CSV: CHANGE_MAP_SPECIAL (opcode 0x70/112 in GMS v95). Two FNames share this
	// opcode (CUserLocal::HandleUpKeyDown for up-key portals, CUserLocal::CheckPortal_Collision
	// for type-9 script portals). Both build the identical packet:
	// Encode1(fieldKey) + EncodeStr(sName) + Encode2(x) + Encode2(y). We route the
	// collision-based script portal path, which matches atlas portal/serverbound/script.go.
	// Verified against v95 IDA CheckPortal_Collision@0x919a10 (build site @0x919b07) and
	// HandleUpKeyDown@0x919e50 (build site @0x91a04b).
	case "CUserLocal::CheckPortal_Collision":
		return []candidate{{name: "Script", pkg: "portal", dir: csvpkg.DirServerbound}}

	// --- World: field (serverbound) ---
	// CSV: CHANGE_MAP (opcode 0x29/41 in GMS v95). The client field-transfer
	// request built by CField::SendTransferFieldRequest@0x5345c0. Wire (per IDA):
	// Encode1(fieldKey) + Encode4(targetId) + EncodeStr(portalName) +
	// [Encode2(x)+Encode2(y) when sPortal!=NULL] + Encode1(unused=0) +
	// Encode1(premium) + Encode1(chase=s_bChase) + [Encode4(targetX)+Encode4(targetY)
	// when s_bChase]. Matches atlas field/serverbound/change.go Change.Encode.
	case "CField::SendTransferFieldRequest":
		return []candidate{{name: "Change", pkg: "field", dir: csvpkg.DirServerbound}}

	// --- World: field (clientbound) ---
	// Affected-area (mist) + kite (the flying-kite field object, called
	// "MessageBox" client-side). FNames + addresses verified against the
	// canonical CSV (docs/packets/MapleStory Ops - ClientBound.csv) and live
	// GMS v95 IDA. Report files become Field<Struct>.{md,json}.
	case "CAffectedAreaPool::OnAffectedAreaCreated":
		// CSV: SPAWN_MIST. Atlas struct is the v83 layout (AffectedAreaCreated);
		// the v95 client decodes a structurally different packet — see
		// FieldAffectedAreaCreated report + _pending.md (v83-vs-v95 divergence).
		return []candidate{{name: "AffectedAreaCreated", pkg: "field", dir: csvpkg.DirClientbound}}
	case "CAffectedAreaPool::OnAffectedAreaRemoved":
		// CSV: REMOVE_MIST. Single Decode4 (mist id) — matches atlas.
		return []candidate{{name: "AffectedAreaRemoved", pkg: "field", dir: csvpkg.DirClientbound}}
	case "CMessageBoxPool::OnMessageBoxEnterField":
		// CSV: SPAWN_KITE. Decode4 id + Decode4 itemId + 2×DecodeStr + 2×Decode2.
		return []candidate{{name: "KiteSpawn", pkg: "field", dir: csvpkg.DirClientbound}}
	case "CMessageBoxPool::OnMessageBoxLeaveField":
		// CSV: REMOVE_KITE. Decode1 animation flag + Decode4 id.
		return []candidate{{name: "KiteDestroy", pkg: "field", dir: csvpkg.DirClientbound}}
	case "CMessageBoxPool::OnCreateFailed":
		// CSV: CANNOT_SPAWN_KITE. Empty body — client reads nothing.
		return []candidate{{name: "KiteError", pkg: "field", dir: csvpkg.DirClientbound}}
	case "CStage::OnSetField":
		// CSV: SET_FIELD (GMS v95 opcode 0x8D/141). Field-entry packet carrying the
		// full CharacterData blob. ENVELOPE-ONLY audit — CharacterData inner shape
		// audited under the character domain (task-028); represented in the IDA
		// export by a single DecodeBuf boundary marker. Struct is SetField
		// (set_field.go) → report FieldSetField.
		return []candidate{{name: "SetField", pkg: "field", dir: csvpkg.DirClientbound}}
	case "CStage::OnSetField#WarpToMap":
		// Same CStage::OnSetField handler, bCharacterData=0 (else branch) — an
		// in-game warp without full re-init. Struct is WarpToMap (warp_to_map.go)
		// → report FieldWarpToMap. Synthetic #-suffixed FName so it claims its own
		// (field, WarpToMap) slot without colliding with the SetField entry.
		return []candidate{{name: "WarpToMap", pkg: "field", dir: csvpkg.DirClientbound}}
	case "CField_ContiMove::OnContiState":
		// CSV: CONTI_STATE (GMS v95 opcode 0xA5/165). Field transport/ship boarding
		// state effect. Struct is Transport (transport.go) → report FieldTransport.
		return []candidate{{name: "Transport", pkg: "field", dir: csvpkg.DirClientbound}}

	// --- World: field (clientbound, effects) ---
	// FIELD_EFFECT (CSV opcode 0x09A/154 in GMS v95) → CField::OnFieldEffect@0x53b790
	// dispatches on a leading effect-type byte (mode). Each field-effect sub-type is a
	// separate atlas struct in field/clientbound/effect.go; modelled via #-suffixed
	// synthetic IDA entries (one per sub-type). Each atlas struct writes the mode byte
	// as its first field, so the synthetic export entry leads with the Decode1 mode.
	case "CField::OnFieldEffect#Summon":
		// case 0: Decode1(mode=0) + Decode1(effect) + Decode4(x) + Decode4(y).
		return []candidate{{name: "EffectSummon", pkg: "field", dir: csvpkg.DirClientbound}}
	case "CField::OnFieldEffect#Tremble":
		// case 1: Decode1(mode=1) + Decode1(bHeavyNShortTremble) + Decode4(delay).
		return []candidate{{name: "EffectTremble", pkg: "field", dir: csvpkg.DirClientbound}}
	case "CField::OnFieldEffect#String":
		// cases 2/3/4/6 (object/screen/sound/BGM): Decode1(mode) + DecodeStr(name).
		return []candidate{{name: "EffectString", pkg: "field", dir: csvpkg.DirClientbound}}
	case "CField::OnFieldEffect#BossHp":
		// case 5: Decode1(mode=5) + Decode4(monsterId) + Decode4(currentHp) +
		// Decode4(maxHp) + Decode1(tagColor) + Decode1(tagBackgroundColor).
		return []candidate{{name: "EffectBossHp", pkg: "field", dir: csvpkg.DirClientbound}}
	case "CField::OnFieldEffect#RewardRullet":
		// case 7: Decode1(mode=7) + Decode4(jobIdx) + Decode4(partIdx) + Decode4(levIdx).
		return []candidate{{name: "EffectRewardRullet", pkg: "field", dir: csvpkg.DirClientbound}}

	// BLOW_WEATHER (CSV opcode 0x09E/158 in GMS v95) → CField::OnBlowWeather@0x5468f0.
	// BAD-FORM single struct (EffectWeather) whose mode byte (!active) is set at
	// construction (NewFieldEffectWeatherStart / End). Analyzer produces one flat
	// verdict; the conditional message string (start-only) is a tool limitation —
	// per-mode table appended manually to the report (NOT refactored).
	case "CField::OnBlowWeather":
		return []candidate{{name: "EffectWeather", pkg: "field", dir: csvpkg.DirClientbound}}

	// CLOCK (CSV opcode 0x0A3/163 in GMS v95) → CField::OnClock@0x531510 dispatches on
	// a leading clockType byte (0/1/2/3/0x64). BAD-FORM single struct (Clock) with the
	// mode set at construction; the Encode switch is mode-keyed. Analyzer produces one
	// flat verdict; per-mode table appended manually to the report (NOT refactored).
	case "CField::OnClock":
		return []candidate{{name: "Clock", pkg: "field", dir: csvpkg.DirClientbound}}

	// CField clientbound proof batch (task-096, cluster 2). Version-invariant
	// layouts derived from IDA (addresses pinned per version in the test markers).
	case "CField::OnTransferFieldReqIgnored":
		return []candidate{{name: "BlockedMap", pkg: "field", dir: csvpkg.DirClientbound}}
	case "CField::OnQuiz":
		return []candidate{{name: "OxQuiz", pkg: "field", dir: csvpkg.DirClientbound}}
	case "CField::OnDestroyClock":
		return []candidate{{name: "StopClock", pkg: "field", dir: csvpkg.DirClientbound}}
	case "CField::OnSetObjectState":
		return []candidate{{name: "SetObjectState", pkg: "field", dir: csvpkg.DirClientbound}}
	case "CField::OnFieldObstacleOnOffStatus":
		return []candidate{{name: "FieldObstacleOnOffList", pkg: "field", dir: csvpkg.DirClientbound}}

	// CField clientbound cluster 3 proof batch (task-096, recipe R-CB).
	// Version-invariant layouts derived from IDA (addresses pinned per version
	// in the test markers). MtsOperation2's fname is CITC:: (not CField::).
	case "CField::OnZakumTimer":
		return []candidate{{name: "ZakumShrine", pkg: "field", dir: csvpkg.DirClientbound}}
	case "CField::OnHontailTimer":
		return []candidate{{name: "HorntailCave", pkg: "field", dir: csvpkg.DirClientbound}}
	case "CField::OnWarnMessage":
		return []candidate{{name: "AriantResult", pkg: "field", dir: csvpkg.DirClientbound}}
	case "CITC::OnQueryCashResult":
		return []candidate{{name: "MtsOperation2", pkg: "field", dir: csvpkg.DirClientbound}}

	// MTS_OPERATION (task-096, recipe OP-MODE-PREFIX). CITC::OnNormalItemResult
	// reads Decode1(mode) and switch-dispatches to one of 35 cash-shop/MTS
	// sub-handlers (0x15..0x3E); only the leading mode byte is on the codec's
	// wire contract. The full dispatcher entry CITC::OnNormalItemResult carries
	// the 35 Delegate arms (left intact); a synthetic #Mode export entry whose
	// calls = the single Decode1 is appended per version so report-gen audits
	// only the mode byte. baseFName strips #Mode so the registry op
	// MTS_OPERATION (fname CITC::OnNormalItemResult) links. jms VERSION-ABSENT
	// (no CITC in jms). field/clientbound/mts_operation.go MtsOperation.
	case "CITC::OnNormalItemResult#Mode":
		return []candidate{{name: "MtsOperation", pkg: "field", dir: csvpkg.DirClientbound}}

	// MTS_OPERATION per-mode body arms (task-096 graduation, FIELD_EFFECT model).
	// Each #-suffixed synthetic FName maps a body-shape group of CITC sub-handlers
	// to a per-mode body codec in field/clientbound/mts_operation_body.go. The
	// codec writes the leading dispatcher mode byte THEN the full arm body (the
	// read order the matching CITC sub-handler performs on CInPacket) — replacing
	// the mode-byte-only MtsOperation false-pass for the covered arms.
	//
	//   Each notice-only ("Empty-shape") arm now has its OWN discrete per-mode
	//   struct (MtsResult<Mode>) that fixes its own mode byte and writes exactly
	//   that byte (the sub-handler reads NOTHING after the dispatcher Decode1(mode)
	//   — StringPool notice only; the trailing m_bITCRequestSent=0 store is a member
	//   write, not a wire read). The shared MtsResultEmpty struct was retired
	//   (task-096 discrete-per-mode rule). All 19 Empty arms are decompile-confirmed
	//   Empty-shape in v83/v84/v87/v95; jms VERSION-ABSENT (no CITC).
	//   #Reason -> MtsResultReason (sub-handlers that read a single Decode1
	//              fail-reason byte). Covers: 0x16 GetITCListFailed,
	//              0x20 SaleCurrentItemToWishFailed; iteration 4
	//              0x18 GetSearchITCListFailed, 0x22 GetUserPurchaseItemFailed,
	//              0x24 GetUserSaleItemFailed.
	//
	// Body shapes are version-stable (gms_v83/v84/v87/v95 IDA-confirmed identical;
	// jms VERSION-ABSENT — no CITC). The remaining arms (list/item-blob, two-int,
	// search/purchase/sale) land in later task-096 iterations.
	case "CITC::OnNormalItemResult#RegisterSaleEntryDone":
		return []candidate{{name: "MtsResultRegisterSaleEntryDone", pkg: "field", dir: csvpkg.DirClientbound}}
	case "CITC::OnNormalItemResult#SaleCurrentItemToWishDone":
		return []candidate{{name: "MtsResultSaleCurrentItemToWishDone", pkg: "field", dir: csvpkg.DirClientbound}}
	case "CITC::OnNormalItemResult#CancelSaleItemDone":
		return []candidate{{name: "MtsResultCancelSaleItemDone", pkg: "field", dir: csvpkg.DirClientbound}}
	case "CITC::OnNormalItemResult#SetZzimDone":
		return []candidate{{name: "MtsResultSetZzimDone", pkg: "field", dir: csvpkg.DirClientbound}}
	case "CITC::OnNormalItemResult#SetZzimFailed":
		return []candidate{{name: "MtsResultSetZzimFailed", pkg: "field", dir: csvpkg.DirClientbound}}
	case "CITC::OnNormalItemResult#DeleteZzimDone":
		return []candidate{{name: "MtsResultDeleteZzimDone", pkg: "field", dir: csvpkg.DirClientbound}}
	case "CITC::OnNormalItemResult#DeleteZzimFailed":
		return []candidate{{name: "MtsResultDeleteZzimFailed", pkg: "field", dir: csvpkg.DirClientbound}}
	case "CITC::OnNormalItemResult#LoadWishSaleListFailed":
		return []candidate{{name: "MtsResultLoadWishSaleListFailed", pkg: "field", dir: csvpkg.DirClientbound}}
	case "CITC::OnNormalItemResult#BuyWishDone":
		return []candidate{{name: "MtsResultBuyWishDone", pkg: "field", dir: csvpkg.DirClientbound}}
	case "CITC::OnNormalItemResult#BuyWishFailed":
		return []candidate{{name: "MtsResultBuyWishFailed", pkg: "field", dir: csvpkg.DirClientbound}}
	case "CITC::OnNormalItemResult#CancelWishDone":
		return []candidate{{name: "MtsResultCancelWishDone", pkg: "field", dir: csvpkg.DirClientbound}}
	case "CITC::OnNormalItemResult#CancelWishFailed":
		return []candidate{{name: "MtsResultCancelWishFailed", pkg: "field", dir: csvpkg.DirClientbound}}
	case "CITC::OnNormalItemResult#BuyItemDone":
		return []candidate{{name: "MtsResultBuyItemDone", pkg: "field", dir: csvpkg.DirClientbound}}
	case "CITC::OnNormalItemResult#BuyItemFailed":
		return []candidate{{name: "MtsResultBuyItemFailed", pkg: "field", dir: csvpkg.DirClientbound}}
	case "CITC::OnNormalItemResult#BuyZzimItemDone":
		return []candidate{{name: "MtsResultBuyZzimItemDone", pkg: "field", dir: csvpkg.DirClientbound}}
	case "CITC::OnNormalItemResult#BuyZzimItemFailed":
		return []candidate{{name: "MtsResultBuyZzimItemFailed", pkg: "field", dir: csvpkg.DirClientbound}}
	case "CITC::OnNormalItemResult#RegisterWishItemDone":
		return []candidate{{name: "MtsResultRegisterWishItemDone", pkg: "field", dir: csvpkg.DirClientbound}}
	case "CITC::OnNormalItemResult#RegisterWishItemFailed":
		return []candidate{{name: "MtsResultRegisterWishItemFailed", pkg: "field", dir: csvpkg.DirClientbound}}
	case "CITC::OnNormalItemResult#BidAuctionFailed":
		return []candidate{{name: "MtsResultBidAuctionFailed", pkg: "field", dir: csvpkg.DirClientbound}}
	case "CITC::OnNormalItemResult#Reason":
		return []candidate{{name: "MtsResultReason", pkg: "field", dir: csvpkg.DirClientbound}}

	//   #TwoInts -> MtsResultTwoInts (sub-handlers that read exactly Decode4 then
	//              Decode4 after the dispatcher mode byte). Covers, iteration 5:
	//              0x27 MoveITCPurchaseItemLtoSDone (tab+1, selectedNo),
	//              0x3D NotifyCancelWishResult (count d, count x). The downstream
	//              use differs but the wire read order is identical Decode4×2.
	//   #RegisterSaleEntryFailed -> MtsResultRegisterSaleEntryFailed
	//              (0x1E; Decode1(reason) then, ONLY when reason==0x48, a trailing
	//              Decode2 short). The conditional tail makes it its own shape.
	//   #SuccessBidInfo -> MtsResultSuccessBidInfo (0x3E; Decode1(soldFlag) +
	//              Decode4(itemId) then, ONLY when itemId>0, Decode4(price) +
	//              DecodeBuffer(8) FILETIME contract date).
	//
	// All decompile-confirmed version-stable in gms_v83/v84/v87/v95 (iteration 5).
	case "CITC::OnNormalItemResult#TwoInts":
		return []candidate{{name: "MtsResultTwoInts", pkg: "field", dir: csvpkg.DirClientbound}}
	case "CITC::OnNormalItemResult#RegisterSaleEntryFailed":
		return []candidate{{name: "MtsResultRegisterSaleEntryFailed", pkg: "field", dir: csvpkg.DirClientbound}}
	case "CITC::OnNormalItemResult#SuccessBidInfo":
		return []candidate{{name: "MtsResultSuccessBidInfo", pkg: "field", dir: csvpkg.DirClientbound}}

	// MTS_OPERATION list/item-blob arms (task-096 iteration 6, FINAL — completes
	// the family). Each embeds one or more ITCITEM entries; an ITCITEM is a
	// GW_ItemSlotBase blob (model.Asset codec) + an MTS trailer (meso/contract/
	// bid metadata). Read order decompile-confirmed version-stable across
	// gms_v83/v84/v87/v95 (the loop count, item-blob, and any leading/trailing
	// scalars are identical; only the dispatcher mode bytes and sub-handler
	// addresses shift). jms VERSION-ABSENT (no CITC). See
	// field/clientbound/mts_operation_list.go.
	//
	//   #GetItcListDone            -> MtsResultGetItcListDone (0x15): Decode4
	//              catItemCnt, Decode4 pageItemCnt (loop), Decode4 category,
	//              Decode4 subCategory, Decode4 page, Decode1 sortType,
	//              Decode1 sortColumn, pageItemCnt × ITCITEM, Decode1 requestSent.
	//   #GetSearchItcListDone      -> MtsResultGetSearchItcListDone (0x17): same
	//              leading 5×Decode4 (catItemCnt, pageItemCnt, category,
	//              subCategory, page), pageItemCnt × ITCITEM; NO sort bytes, NO
	//              trailing requestSent byte.
	//   #GetUserPurchaseItemDone   -> MtsResultGetUserPurchaseItemDone (0x21):
	//              Decode4 totalCount, totalCount × ITCITEM, Decode4 limitedCount,
	//              Decode1 requestSent.
	//   #GetUserSaleItemDone       -> MtsResultGetUserSaleItemDone (0x23): Decode4
	//              totalCount, totalCount × ITCITEM. No trailing fields.
	//   #LoadWishSaleListDone      -> MtsResultLoadWishSaleListDone (0x2D): Decode4
	//              totalCount, totalCount × ITCITEM. No trailing fields.
	case "CITC::OnNormalItemResult#GetItcListDone":
		return []candidate{{name: "MtsResultGetItcListDone", pkg: "field", dir: csvpkg.DirClientbound}}
	case "CITC::OnNormalItemResult#GetSearchItcListDone":
		return []candidate{{name: "MtsResultGetSearchItcListDone", pkg: "field", dir: csvpkg.DirClientbound}}
	case "CITC::OnNormalItemResult#GetUserPurchaseItemDone":
		return []candidate{{name: "MtsResultGetUserPurchaseItemDone", pkg: "field", dir: csvpkg.DirClientbound}}
	case "CITC::OnNormalItemResult#GetUserSaleItemDone":
		return []candidate{{name: "MtsResultGetUserSaleItemDone", pkg: "field", dir: csvpkg.DirClientbound}}
	case "CITC::OnNormalItemResult#LoadWishSaleListDone":
		return []candidate{{name: "MtsResultLoadWishSaleListDone", pkg: "field", dir: csvpkg.DirClientbound}}

	// FOOTHOLD_INFO clientbound (task-096, recipe R-CB). CField::OnFootHoldInfo
	// is version-divergent: v87 reads a single entry (name, mode, [mode==2:
	// 8×int32 + 2×byte]); v95/jms read Decode4(count) then per entry name, mode,
	// idCount, idCount×int32, [mode==2: 7×int32 + 2×byte]. GMS<87 VERSION-ABSENT.
	// The CMapLoadable::FootHoldStateChange/FootHoldMove delegates are map-apply
	// application logic (no wire bytes), stripped from the exports per §10.
	// field/clientbound/foothold_info.go FootholdInfo.
	case "CField::OnFootHoldInfo":
		return []candidate{{name: "FootholdInfo", pkg: "field", dir: csvpkg.DirClientbound}}

	// FOOTHOLD_INFO serverbound (task-096, recipe R-SB). CField::OnRequestFootHoldInfo
	// is the client's REPLY to the server foothold-info request: it builds
	// COutPacket(270 v95 / 0xED jms) appending one entry per dynamic object with
	// NO count prefix — Encode4(nCurState) + Encode4(nCurX)+Encode4(nCurY)+
	// Encode1(revV)+Encode1(revH) (or 4 zeros). The server decodes the stream to
	// exhaustion. Exists only in GMS v95 (@0x52ddd0) + jms (@0x576cd2);
	// VERSION-ABSENT in GMS v83/v84/v87. field/serverbound/request_foothold_info.go.
	case "CField::OnRequestFootHoldInfo":
		return []candidate{{name: "RequestFootholdInfo", pkg: "field", dir: csvpkg.DirServerbound}}

	// CField clientbound minigame family (task-096, recipe R-CB). Version-invariant
	// wire layouts derived from IDA (addresses pinned per version in the test markers).
	// CField_ContiMove::OnContiMove and CField_AriantArena::OnUserScore carry a
	// post-read switch-dispatch / Delegate chain that is application logic, not a
	// wire read; those delegate entries are stripped from the exports (and re-pinned)
	// so Resolve yields only the wire fields.
	case "CField_SnowBall::OnSnowBallState":
		return []candidate{{name: "SnowballState", pkg: "field", dir: csvpkg.DirClientbound}}
	case "CField_SnowBall::OnSnowBallHit":
		return []candidate{{name: "SnowballHit", pkg: "field", dir: csvpkg.DirClientbound}}
	case "CField_SnowBall::OnSnowBallMsg":
		return []candidate{{name: "SnowballMessage", pkg: "field", dir: csvpkg.DirClientbound}}
	// LEFT_KNOCK_BACK (task-096 R-CB). EMPTY body — the handler reads no bytes
	// (SetImpact(0x12C,1) only). field/clientbound/snowball_touch.go SnowballTouch.
	case "CField_SnowBall::OnSnowBallTouch":
		return []candidate{{name: "SnowballTouch", pkg: "field", dir: csvpkg.DirClientbound}}
	// IDA_0X09C / OnStalkResult (task-096 R-CB). Minimap stalkee-list update: a
	// count-prefixed loop; the export flattens one insert iteration (count + charId
	// + flag + name + x + y). The InsertStalkee/RemoveStalkee/_Release Delegates are
	// UI logic. field/clientbound/stalk_result.go StalkResult. v84 VERSION-ABSENT.
	case "CField::OnStalkResult":
		return []candidate{{name: "StalkResult", pkg: "field", dir: csvpkg.DirClientbound}}
	// ADMIN_RESULT (task-096 R-CB). Mode-demux flattened like SPOUSE_CHAT; the
	// post-mode flat read order differs per version (the codec branches on tenant
	// version). field/clientbound/admin_result.go AdminResult.
	case "CField::OnAdminResult":
		return []candidate{{name: "AdminResult", pkg: "field", dir: csvpkg.DirClientbound}}
	case "CField_Coconut::OnCoconutHit":
		return []candidate{{name: "CoconutHit", pkg: "field", dir: csvpkg.DirClientbound}}
	case "CField_Coconut::OnCoconutScore":
		return []candidate{{name: "CoconutScore", pkg: "field", dir: csvpkg.DirClientbound}}
	case "CField_GuildBoss::OnHealerMove":
		return []candidate{{name: "GuildBossHealerMove", pkg: "field", dir: csvpkg.DirClientbound}}
	case "CField_GuildBoss::OnPulleyStateChange":
		return []candidate{{name: "GuildBossPulleyStateChange", pkg: "field", dir: csvpkg.DirClientbound}}
	case "CField_AriantArena::OnUserScore":
		return []candidate{{name: "AriantArenaUserScore", pkg: "field", dir: csvpkg.DirClientbound}}
	case "CField_AriantArena::OnShowResult":
		return []candidate{{name: "AriantArenaShowResult", pkg: "field", dir: csvpkg.DirClientbound}}
	case "CField_Battlefield::OnScoreUpdate":
		return []candidate{{name: "SheepRanchInfo", pkg: "field", dir: csvpkg.DirClientbound}}
	case "CField_Battlefield::OnTeamChanged":
		return []candidate{{name: "SheepRanchClothes", pkg: "field", dir: csvpkg.DirClientbound}}
	case "CField_ContiMove::OnContiMove":
		return []candidate{{name: "ContiMove", pkg: "field", dir: csvpkg.DirClientbound}}
	case "CField_Massacre::OnMassacreIncGauge":
		return []candidate{{name: "PyramidGauge", pkg: "field", dir: csvpkg.DirClientbound}}
	case "CField_MassacreResult::OnMassacreResult":
		return []candidate{{name: "PyramidScore", pkg: "field", dir: csvpkg.DirClientbound}}

	// CField serverbound minigame/door send-sites (task-096, recipe R-SB). These
	// are the client BasicActionAttack / Update / TryEnterTownPortal send-site
	// FNames (#suffix tags appended in the export so each maps to a distinct op,
	// since several share a base send-site). The codecs live under
	// field/serverbound; addresses are pinned per version in the test markers.
	case "CField_SnowBall::BasicActionAttack#Snowball":
		return []candidate{{name: "Snowball", pkg: "field", dir: csvpkg.DirServerbound}}
	case "CField_SnowBall::Update#LeftKnockback":
		return []candidate{{name: "LeftKnockback", pkg: "field", dir: csvpkg.DirServerbound}}
	case "CField_Coconut::BasicActionAttack#Coconut":
		return []candidate{{name: "Coconut", pkg: "field", dir: csvpkg.DirServerbound}}
	case "CField_GuildBoss::BasicActionAttack#GuildBoss":
		return []candidate{{name: "GuildBoss", pkg: "field", dir: csvpkg.DirServerbound}}
	case "CField::TryEnterTownPortal#UseDoor":
		return []candidate{{name: "UseDoor", pkg: "field", dir: csvpkg.DirServerbound}}

	// CField admin/slash serverbound family (task-096, recipe R-SB). The client
	// /-command parser CField::SendChatMsgSlash builds several distinct COutPacket
	// opcodes — one per command class — so each maps to a distinct op via a
	// #suffix export entry (the base CField::SendChatMsgSlash entry is untouched).
	// Layouts IDA-derived per version (v95 authoritative); addresses pinned per
	// version in the test markers. SLIDE_REQUEST is v95+jms only; SUE_CHARACTER is
	// jms-absent. SUE_CHARACTER version-branches its leading field (v83/v84/v87
	// int32 char id, v95 sub-command string).
	case "CField::SendChatMsgSlash#AdminChat":
		return []candidate{{name: "AdminChat", pkg: "field", dir: csvpkg.DirServerbound}}
	case "CField::SendChatMsgSlash#AdminCommand":
		return []candidate{{name: "AdminCommand", pkg: "field", dir: csvpkg.DirServerbound}}
	case "CField::SendChatMsgSlash#AdminLog":
		return []candidate{{name: "AdminLog", pkg: "field", dir: csvpkg.DirServerbound}}
	case "CField::SendChatMsgSlash#MatchTable":
		return []candidate{{name: "MatchTable", pkg: "field", dir: csvpkg.DirServerbound}}
	case "CField::SendChatMsgSlash#SlideRequest":
		return []candidate{{name: "SlideRequest", pkg: "field", dir: csvpkg.DirServerbound}}
	case "CField::SendChatMsgSlash#SueCharacter":
		return []candidate{{name: "SueCharacter", pkg: "field", dir: csvpkg.DirServerbound}}

	// CField_Tournament clientbound family (task-096, recipe R-CB). Version-invariant
	// layouts derived from IDA (addresses pinned per version in the test markers).
	// OnTournamentSetPrize carries a trailing post-read Delegate (sub_XXXXXX in the
	// v83/v87/jms exports) that is application logic, not a wire read; those delegate
	// entries are stripped from the exports (and re-pinned) so Resolve yields only the
	// wire fields. OnTournamentMatchTable and OnPacket are empty-body no-op stubs.
	case "CField_Tournament::OnTournament":
		return []candidate{{name: "Tournament", pkg: "field", dir: csvpkg.DirClientbound}}
	case "CField_Tournament::OnTournamentMatchTable":
		return []candidate{{name: "TournamentMatchTable", pkg: "field", dir: csvpkg.DirClientbound}}
	case "CField_Tournament::OnTournamentSetPrize":
		return []candidate{{name: "TournamentSetPrize", pkg: "field", dir: csvpkg.DirClientbound}}
	case "CField_Tournament::OnTournamentUEW":
		return []candidate{{name: "TournamentUew", pkg: "field", dir: csvpkg.DirClientbound}}
	case "CField_Tournament::OnPacket":
		return []candidate{{name: "TournamentCharacters", pkg: "field", dir: csvpkg.DirClientbound}}

	// CField_Wedding clientbound family (task-096, recipe R-CB). Version-invariant
	// layouts derived from IDA (addresses pinned per version in the test markers),
	// except WeddingProgress which is version-branched: jms drops the leading step
	// byte (groomId + brideId only). The OnWeddingProgress fname is also used by
	// serverbound WEDDING action/talk ops (different direction); only the clientbound
	// candidate is declared here. OnWeddingCeremonyEnd is an empty-body no-op stub.
	case "CField_Wedding::OnWeddingProgress":
		return []candidate{{name: "WeddingProgress", pkg: "field", dir: csvpkg.DirClientbound}}
	case "CField_Wedding::OnWeddingCeremonyEnd":
		return []candidate{{name: "WeddingCeremonyEnd", pkg: "field", dir: csvpkg.DirClientbound}}
	// Serverbound wedding send-sites (task-096, recipe R-SB). OnWeddingProgress
	// both reads the clientbound progress packet and emits these serverbound
	// replies; the #Action / #Talk export suffixes disambiguate them from the
	// base clientbound case above. jms-absent (no jms registry rows / markers).
	case "CField_Wedding::OnWeddingProgress#Action":
		return []candidate{{name: "WeddingAction", pkg: "field", dir: csvpkg.DirServerbound}}
	case "CField_Wedding::OnWeddingProgress#Talk":
		return []candidate{{name: "WeddingTalk", pkg: "field", dir: csvpkg.DirServerbound}}

	// CField witch-tower / item-upgrade clientbound family (task-096). The
	// OnScoreUpdate handler is shared by two ops that differ by version: it backs
	// WITCH_TOWER_SCORE_UPDATE on v83/v84/v87/jms and ARIANT_SCORE on v95 (where
	// v95 routes WITCH_TOWER_SCORE_UPDATE to OnChaosZakumTimer instead). Both
	// clientbound candidates are returned; the matrix resolves them per op-identity
	// via the registry op->fname mapping. OnItemUpgrade is an empty-body vtable
	// forwarder backing VICIOUS_HAMMER (absent from the jms registry).
	case "CField_Witchtower::OnScoreUpdate":
		return []candidate{
			{name: "WitchTowerScoreUpdate", pkg: "field", dir: csvpkg.DirClientbound},
			{name: "AriantScore", pkg: "field", dir: csvpkg.DirClientbound},
		}
	case "CField::OnChaosZakumTimer":
		return []candidate{{name: "WitchTowerScoreUpdate", pkg: "field", dir: csvpkg.DirClientbound}}
	case "CField::OnItemUpgrade":
		return []candidate{{name: "ViciousHammer", pkg: "field", dir: csvpkg.DirClientbound}}

	// CField clientbound cluster 2, remaining 9 ops (task-096). Version-invariant
	// layouts derived from IDA (addresses pinned per version in the test markers).
	case "CField::OnTransferChannelReqIgnored":
		return []candidate{{name: "BlockedServer", pkg: "field", dir: csvpkg.DirClientbound}}
	case "CField::OnFieldSpecificData":
		return []candidate{{name: "ForcedMapEquip", pkg: "field", dir: csvpkg.DirClientbound}}
	case "CField::OnSummonItemInavailable":
		return []candidate{{name: "SummonItemUnavailable", pkg: "field", dir: csvpkg.DirClientbound}}
	case "CField::OnFieldObstacleOnOff":
		return []candidate{{name: "FieldObstacleOnOff", pkg: "field", dir: csvpkg.DirClientbound}}
	case "CField::OnFieldObstacleAllReset":
		return []candidate{{name: "FieldObstacleAllReset", pkg: "field", dir: csvpkg.DirClientbound}}
	case "CField::OnSetQuestClear":
		return []candidate{{name: "SetQuestClear", pkg: "field", dir: csvpkg.DirClientbound}}
	case "CField::OnSetQuestTime":
		return []candidate{{name: "SetQuestTime", pkg: "field", dir: csvpkg.DirClientbound}}
	case "CField::OnDesc":
		return []candidate{{name: "GmEventInstructions", pkg: "field", dir: csvpkg.DirClientbound}}
	case "CField::OnPlayJukeBox":
		return []candidate{{name: "PlayJukebox", pkg: "field", dir: csvpkg.DirClientbound}}

	// --- World: npc (clientbound) ---
	// Non-conversation NPC packets. FNames + addresses verified against the
	// canonical CSV (docs/packets/MapleStory Ops - ClientBound.csv) and live
	// GMS v95 IDA. Report files become Npc<Struct>.{md,json}.
	case "CNpc::OnMove":
		// CSV: NPC_ACTION (GMS v95 opcode 0x13A/314) dispatched via
		// CNpcPool::OnNpcPacket@0x679260 which Decode4(npcId) BEFORE calling
		// CNpc::OnMove. OnMove@0x678060 then reads Decode1(action) + Decode1(chatIdx),
		// and (if m_pTemplate->bMove, a client template flag) a CMovePath movement
		// body. Atlas Action writes Int(objectId — dispatcher prefix) + Byte(unk) +
		// Byte(unk2) + optional movement. Dispatcher-prefix pattern; the movement
		// presence is server-controlled (hasMovement) and template-gated client-side.
		return []candidate{{name: "Action", pkg: "npc", dir: csvpkg.DirClientbound}}
	case "CUserLocal::OnTutorMsg#Message":
		// CSV: TALK_GUIDE (GMS v95 opcode 0x157/343) → CUserLocal::OnTutorMsg@0x916f60.
		// bByMessage==0 (false) arm: DecodeStr(message) + Decode4(width) + Decode4(duration)
		// → CTutor::OnMessage(string,int,int). Atlas GuideTalkMessage. ✓ fixed: leading
		// bool corrected from true→false to match the client's message branch.
		return []candidate{{name: "GuideTalkMessage", pkg: "npc", dir: csvpkg.DirClientbound}}
	case "CUserLocal::OnTutorMsg#Idx":
		// Same OnTutorMsg handler, bByMessage!=0 (true) arm: Decode4(hintId) +
		// Decode4(duration) → CTutor::OnMessage(int,int), NO string. Atlas GuideTalkIdx.
		// ✓ fixed: leading bool corrected from false→true to match the client's index branch.
		return []candidate{{name: "GuideTalkIdx", pkg: "npc", dir: csvpkg.DirClientbound}}
	case "CShopDlg::SetShopDlg":
		// CSV: OPEN_NPC_SHOP (GMS v95 opcode 0x12F/303) → CShopDlg::OnPacket@0x6eb7d0
		// (nType==364) → CShopDlg::SetShopDlg@0x6eab00. Decode4(npcTemplateId) +
		// Decode2(count) + count×{Decode4 itemId + Decode4 mesoPrice + Decode1 discount +
		// Decode4 tokenItemId + Decode4 tokenPrice + Decode4 period + Decode4 levelLimit +
		// (itemId/10000∈{207,233} ? DecodeBuffer(8) unitPrice : Decode2 quantity) +
		// Decode2 slotMax}. Atlas ShopList loop body matches per-item; analyzer flattens
		// the loop → ⚠️ tool-limitation (manually verified — see NpcShopList ## Loop bounds).
		return []candidate{{name: "ShopList", pkg: "npc", dir: csvpkg.DirClientbound}}
	case "CShopDlg::OnPacket#Simple":
		// CSV: CONFIRM_SHOP_TRANSACTION (GMS v95 opcode 0x12F? no — 0x130/304) →
		// CShopDlg::OnPacket@0x6eb7d0 (nType==365) switch(Decode1 mode). Most modes
		// (0,1,2,3,5,8,9,0xA,0xD,0x10,0x11,0x12) read no further fields (mode byte only,
		// then a StringPool Notice). Atlas ShopOperationSimple writes Byte(mode). ✓
		// ⚠️ OP-FAMILY-npc-shop-operation: full mode enum deferred to _pending.md.
		return []candidate{{name: "ShopOperationSimple", pkg: "npc", dir: csvpkg.DirClientbound}}
	case "CShopDlg::OnPacket#LevelRequirement":
		// Same handler, cases 0xE/0xF (over/under level requirement): Decode4(level).
		// Atlas ShopOperationLevelRequirement writes Byte(mode) + Int(levelLimit). ✓
		return []candidate{{name: "ShopOperationLevelRequirement", pkg: "npc", dir: csvpkg.DirClientbound}}
	case "CShopDlg::OnPacket#GenericError":
		// Same handler, case 0x13: Decode1(hasReason) + (hasReason ? DecodeStr(reason)).
		// Atlas ShopOperationGenericError writes Byte(mode) + Bool(hasReason) +
		// optional AsciiString(reason). ✓
		return []candidate{{name: "ShopOperationGenericError", pkg: "npc", dir: csvpkg.DirClientbound}}
	// shop_operation_body.go has no exported struct (pure helper functions); no candidate entry.
	case "CNpcPool::OnNpcEnterField":
		// CSV: SPAWN_NPC (GMS v95 opcode 0x12F/303). OnNpcEnterField@0x679680 reads
		// Decode4(npcId) + Decode4(templateId), then CNpc::Init@0x676770 reads
		// Decode2(x) + Decode2(cy) + Decode1(moveAction/f) + Decode2(fh) + Decode2(rx0) +
		// Decode2(rx1) + Decode1(enabled). Atlas Spawn writes Int(id) + Int(template) +
		// Int16(x) + Int16(cy) + Byte(f) + Short(fh) + Int16(rx0) + Int16(rx1) + Byte(1). ✓
		return []candidate{{name: "Spawn", pkg: "npc", dir: csvpkg.DirClientbound}}
	case "CNpcPool::OnNpcChangeController":
		// CSV: SPAWN_NPC_REQUEST_CONTROLLER (GMS v95 opcode 0x131/305).
		// OnNpcChangeController@0x679730 reads Decode1(localFlag) + Decode4(npcId); when
		// localFlag→SetLocalNpc@0x679440 reads Decode4(templateId) then CNpc::Init reads
		// the same x/cy/f/fh/rx0/rx1/enabled tail. Atlas SpawnRequestController writes
		// Byte(1) + Int(id) + Int(template) + Int16(x) + Int16(cy) + Byte(f) + Short(fh) +
		// Int16(rx0) + Int16(rx1) + Bool(miniMap → maps to CNpc::Init enabled). ✓
		return []candidate{{name: "SpawnRequestController", pkg: "npc", dir: csvpkg.DirClientbound}}

	// --- World: npc (clientbound, conversation) ---
	// SCRIPT_MESSAGE / NPC_TALK (GMS v95 opcode 363/0x16B). The clientbound NPC
	// dialog packet. CScriptMan::OnPacket@0x6de360 dispatches nType==363 to
	// CScriptMan::OnScriptMessage@0x6de0f0, which reads a common header
	// (speakerType byte + npcTemplateId int + msgType byte + param byte) then
	// switch(msgType) to 14 per-dialog-type handlers (cases 0,1,2,3,4,5,6,7,8,9,
	// 10,11,13,14,15). Atlas models this as one NpcConversation wrapper struct
	// (header + opaque detail byte array) plus 14 separate *ConversationDetail
	// structs (each its own Encode = the per-type body). We route the wrapper and
	// each detail individually so each gets a per-dialog-type verdict. Each detail
	// is modeled as a #-suffixed synthetic IDA entry whose Decode ops cover ONLY
	// that case's body (the wrapper covers the header + secondary + opaque body).
	case "CScriptMan::OnScriptMessage":
		// Wrapper: common header envelope (Decode1 speakerType + Decode4 npcTemplate
		// + Decode1 msgType + Decode1 param + guarded Decode4 secondary + opaque body).
		// Per-dialog-type bodies audited in the NpcSay*/NpcAsk* reports below.
		return []candidate{{name: "NpcConversation", pkg: "npc", dir: csvpkg.DirClientbound}}
	case "CScriptMan::OnSay#Say":
		// msgType 0. OnSay@0x6dc110 body: DecodeStr(message)+Decode1(prev)+Decode1(next). ✓
		return []candidate{{name: "SayConversationDetail", pkg: "npc", dir: csvpkg.DirClientbound}}
	case "CScriptMan::OnSayImage#SayImage":
		// msgType 1. OnSayImage@0x6dc310: Decode1(count)+loop DecodeStr(image).
		// ✓ fixed: image-count corrected from WriteInt→WriteByte to match Decode1@0x6dc3d9.
		return []candidate{{name: "SayImageConversationDetail", pkg: "npc", dir: csvpkg.DirClientbound}}
	case "CScriptMan::OnAskYesNo#AskYesNo":
		// msgType 2 (AskYesNo) and 13 (AskYesNoQuest) share OnAskYesNo@0x6dc5a0:
		// DecodeStr(message). Atlas has one struct for both. ✓
		return []candidate{{name: "AskYesNoConversationDetail", pkg: "npc", dir: csvpkg.DirClientbound}}
	case "CScriptMan::OnAskText#AskText":
		// msgType 3. OnAskText@0x6dc790: DecodeStr(msg)+DecodeStr(def)+Decode2(min)+Decode2(max). ✓
		return []candidate{{name: "AskTextConversationDetail", pkg: "npc", dir: csvpkg.DirClientbound}}
	case "CScriptMan::OnAskNumber#AskNumber":
		// msgType 4. OnAskNumber@0x6dcc00: DecodeStr(msg)+Decode4(def)+Decode4(min)+Decode4(max). ✓
		return []candidate{{name: "AskNumberConversationDetail", pkg: "npc", dir: csvpkg.DirClientbound}}
	case "CScriptMan::OnAskMenu#AskMenu":
		// msgType 5. OnAskMenu@0x6dce00: DecodeStr(message). ✓
		return []candidate{{name: "AskMenuConversationDetail", pkg: "npc", dir: csvpkg.DirClientbound}}
	case "CScriptMan::OnAskQuiz#AskQuiz":
		// msgType 6. OnAskQuiz@0x6dbaf0 → CWvsContext::OnInitialQuiz@0x9ffad0:
		// Decode1(flag); flag==0 → DecodeStr×3 (title/problem/hint)+Decode4×3 (min/max/time).
		// Atlas Bool(Fail) + guarded body; flag==0/Fail==false same polarity. ✓
		return []candidate{{name: "AskQuizConversationDetail", pkg: "npc", dir: csvpkg.DirClientbound}}
	case "CScriptMan::OnAskSpeedQuiz#AskSpeedQuiz":
		// msgType 7. OnAskSpeedQuiz@0x6dbb10 → CWvsContext::OnInitialSpeedQuiz@0x9f1d50:
		// Decode1(flag); flag==0 → Decode4×5 (type/answer/correct/remain/time). ✓
		return []candidate{{name: "AskSpeedQuizConversationDetail", pkg: "npc", dir: csvpkg.DirClientbound}}
	case "CScriptMan::OnAskAvatar#AskAvatar":
		// msgType 8. OnAskAvatar@0x6dcff0: DecodeStr(msg)+Decode1(count)+loop Decode4(style). ✓
		return []candidate{{name: "AskAvatarConversationDetail", pkg: "npc", dir: csvpkg.DirClientbound}}
	case "CScriptMan::OnAskMembershopAvatar#AskMemberShopAvatar":
		// msgType 9. OnAskMembershopAvatar@0x6dd340: DecodeStr(msg)+Decode1(count)+loop Decode4(candidate).
		// ✓ fixed: candidate-count corrected from WriteInt→WriteByte to match Decode1@0x6dd394.
		return []candidate{{name: "AskMemberShopAvatarConversationDetail", pkg: "npc", dir: csvpkg.DirClientbound}}
	case "CScriptMan::OnAskPet#AskPet":
		// msgType 10. OnAskPet@0x6dd6e0: DecodeStr(msg)+Decode1(count)+loop{DecodeBuffer(8)+Decode1}.
		// cashItemSN 8-byte modeled as Decode8 for width parity with atlas WriteLong. ✓
		return []candidate{{name: "AskPetConversationDetail", pkg: "npc", dir: csvpkg.DirClientbound}}
	case "CScriptMan::OnAskPetAll#AskPetAll":
		// msgType 11. OnAskPetAll@0x6ddbe0: DecodeStr(msg)+Decode1(count)+Decode1(exceptionExist)+
		// loop{DecodeBuffer(8)+Decode1}. ✓
		return []candidate{{name: "AskPetAllConversationDetail", pkg: "npc", dir: csvpkg.DirClientbound}}
	case "CScriptMan::OnAskBoxText#AskBoxText":
		// msgType 14. OnAskBoxText@0x6dc9c0: DecodeStr(msg)+DecodeStr(def)+Decode2(col)+Decode2(line). ✓
		return []candidate{{name: "AskBoxTextConversationDetail", pkg: "npc", dir: csvpkg.DirClientbound}}
	case "CScriptMan::OnAskSlideMenu#AskSlideMenu":
		// msgType 15. OnAskSlideMenu@0x6dbe50: Decode4(slideDlgType) → CSlideMenuDlgEX::SetSlideMenuDlg@0x7156f0
		// Decode4(menuType)+DecodeStr(message). Atlas (GMS>83) Int(Unknown)+Int(MenuType)+AsciiString(Message);
		// leading Int(Unknown) maps to slideDlgType; v95 major>83 so the guard fires. ✓
		return []candidate{{name: "AskSlideMenuConversationDetail", pkg: "npc", dir: csvpkg.DirClientbound}}

	// --- World: npc (serverbound) ---
	// Client-built request packets. SERVERBOUND: the client's Send*/dialog-reply
	// build site writes opcode + body; there is NO characterId dispatcher prefix
	// (the client writes npcId/op-byte as the first field itself, verified in
	// CUserLocal::TalkToNpc@0x9321f0 and CNpc::GenerateMovePath@0x671590). Report
	// files become Npc<Struct>.{md,json}.
	case "CNpc::GenerateMovePath":
		// CSV: NPC_ACTION (GMS v95 opcode 241/0xF1). CNpc::GenerateMovePath@0x671590
		// builds COutPacket(241) + Encode4(npcId) + Encode1(nAction) + Encode1(nChatIdx)
		// + (m_pTemplate->bMove ? CMovePath::Flush movement body). Atlas ActionRequest
		// writes Int(objectId) + Byte(unk=action) + Byte(unk2=chatIdx) + optional
		// WriteByteArray(movement). Movement presence is server-controlled (hasMovement)
		// and template-gated client-side. ✓ header match.
		return []candidate{{name: "ActionRequest", pkg: "npc", dir: csvpkg.DirServerbound}}
	case "CUserLocal::TalkToNpc":
		// CSV: NPC_TALK (GMS v95 opcode 63/0x3F). CUserLocal::TalkToNpc@0x9321f0
		// (non-quest branch) builds COutPacket(63) + Encode4(npcId) + Encode2(x) +
		// Encode2(y). Atlas StartConversation writes Int(oid) + Int16(x) + Int16(y). ✓
		return []candidate{{name: "StartConversation", pkg: "npc", dir: csvpkg.DirServerbound}}
	case "CScriptMan::OnSay#Reply":
		// CSV: NPC_TALK_MORE (GMS v95 opcode 65/0x41) generic continue reply, built
		// inside CScriptMan::OnSay@0x6dc110 after DoModal: COutPacket(65) + Encode1(msgType)
		// + Encode1(action). Atlas ContinueConversation writes Byte(lastMessageType) +
		// Byte(action). This is the dispatcher header; selection/text trailing fields are
		// separate structs. ✓
		return []candidate{{name: "ContinueConversation", pkg: "npc", dir: csvpkg.DirServerbound}}
	case "CScriptMan::OnAskMenu#Selection":
		// Same NPC_TALK_MORE opcode (65). The AskMenu reply (msgType 5) appends
		// Encode4(m_nSelect) when action==1 (CScriptMan::OnAskMenu@0x6dce00). AskAvatar
		// (msgType 8) appends a single byte (Encode1@0x6dd26e). Atlas
		// ContinueConversationSelection.Decode picks width at runtime (r.Available()>=4 →
		// Int32 wide, else Byte); Encode mirrors via the `wide` flag. Analyzer flattens
		// the branch; modeled as the wide (Encode4) path. ⚠️ runtime width guard — tool
		// limitation; both client widths covered by the wide/narrow branch.
		return []candidate{{name: "ContinueConversationSelection", pkg: "npc", dir: csvpkg.DirServerbound}}
	case "CScriptMan::OnAskText#Reply":
		// Same NPC_TALK_MORE opcode (65). The AskText reply (msgType 3) appends
		// EncodeStr(input) when action==1 (CScriptMan::OnAskText@0x6dc790). Atlas
		// ContinueConversationText writes AsciiString(text). ✓
		return []candidate{{name: "ContinueConversationText", pkg: "npc", dir: csvpkg.DirServerbound}}
	case "CShopDlg::OnPacket#ShopDispatch":
		// CSV: NPC_SHOP (GMS v95 opcode 66/0x42) op-byte dispatcher. The transaction
		// senders each build COutPacket(66) with a leading Encode1(op) discriminator
		// (0=BUY/1=SELL/2=RECHARGE) then the per-op body. Atlas Shop reads the op byte,
		// then the channel handler delegates to ShopBuy/ShopSell/ShopRecharge. Atlas Shop
		// writes Byte(op). ⚠️ OP-FAMILY-npc-shop-serverbound: op-byte values are runtime
		// config (operations map), not in the template — deferred to _pending.md.
		return []candidate{{name: "Shop", pkg: "npc", dir: csvpkg.DirServerbound}}
	case "CShopDlg::SendBuyRequest":
		// NPC_SHOP BUY body (op=0). CShopDlg::SendBuyRequest@0x6e9bb0 after the op byte:
		// Encode2(slot) + Encode4(itemId) + Encode2(quantity) + Encode4(discountPrice).
		// Atlas ShopBuy writes Short(slot) + Int(itemId) + Short(quantity) + Int(discountPrice). ✓
		return []candidate{{name: "ShopBuy", pkg: "npc", dir: csvpkg.DirServerbound, prefixSubOps: 1}}
	case "CShopDlg::SendSellRequest":
		// NPC_SHOP SELL body (op=1). CShopDlg::SendSellRequest@0x6e7260 after the op byte:
		// Encode2(slot/nPOS) + Encode4(itemId) + Encode2(quantity). Atlas ShopSell writes
		// Int16(slot) + Int(itemId) + Short(quantity). ✓
		return []candidate{{name: "ShopSell", pkg: "npc", dir: csvpkg.DirServerbound, prefixSubOps: 1}}
	case "CShopDlg::SendRechargeRequest":
		// NPC_SHOP RECHARGE body (op=2). CShopDlg::SendRechargeRequest@0x6e4e90 after the
		// op byte: Encode2(slot/nPos). Atlas ShopRecharge writes Short(slot). ✓
		return []candidate{{name: "ShopRecharge", pkg: "npc", dir: csvpkg.DirServerbound, prefixSubOps: 1}}
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

// repoRelAtlasFile returns a stable, repo-relative display path for an Atlas
// source file so a committed audit report never embeds a machine-specific
// absolute prefix (a developer home directory, a CI workspace). This is for
// DISPLAY only — the analyzer reads the file via the original (possibly
// absolute) path; only the value stored in the report is normalized, so the
// read path is never affected.
//
// A relative input — the documented `--atlas-packet ../../libs/atlas-packet`
// invocation or the `libs/atlas-packet` default — is already safe and is
// returned unchanged so existing reports do not churn. An absolute input is
// rewritten to start at the `libs/atlas-packet/` marker; if that marker is
// absent it falls back to the base name. Either way the result is never absolute.
func repoRelAtlasFile(p string) string {
	s := filepath.ToSlash(p)
	if !filepath.IsAbs(p) {
		return s
	}
	const marker = "libs/atlas-packet/"
	if i := strings.LastIndex(s, marker); i >= 0 {
		return s[i:]
	}
	return filepath.Base(s)
}

// exportCarriesPrefix reports whether the client export's leading field(s) match
// the family-operation wrapper's flattened shape — i.e. the baseline faithfully
// includes the sub-op byte the wrapper accounts for. It reuses diff.Diff on just
// the export's head so the width/representation tolerance is identical to the main
// comparison. False when the export is shorter than the wrapper or the head does
// not match (an incomplete baseline that omitted the sub-op): the caller then
// leaves Atlas body-only rather than manufacturing a one-field misalignment.
func exportCarriesPrefix(reg *atlaspacket.TypeRegistry, ctx atlaspacket.GuardContext, pcalls []atlaspacket.Call, exportCalls []idasrc.FieldCall) bool {
	flat := diff.FlattenWithRegistry(pcalls, ctx, reg)
	if len(flat) == 0 || len(exportCalls) < len(flat) {
		return false
	}
	head := idasrc.Fields{Calls: exportCalls[:len(flat)]}
	for _, r := range diff.Diff(flat, head) {
		if r.Verdict != diff.VerdictMatch {
			return false
		}
	}
	return true
}

// hasUnresolvedBranch reports whether any call (recursively, through loop and
// sub-struct bodies) sits under a guard the analyzer could NOT compile to a
// version predicate. guardFromIf tags those with a "<unparsed:...>" text (and an
// always-true eval). These are `if m.<field>` data-dependent branches OR
// version-derived locals (`b := <version expr>; if b`) the flatten doesn't
// trace — either way the writer's wire shape can't be resolved statically, so a
// flat positional diff cannot model it. Version guards (Region/MajorVersion)
// parse cleanly and never carry this marker.
func hasUnresolvedBranch(calls []atlaspacket.Call) bool {
	for _, c := range calls {
		if c.Guard != nil && strings.Contains(c.Guard.String(), "<unparsed:") {
			return true
		}
		if len(c.Body) > 0 && hasUnresolvedBranch(c.Body) {
			return true
		}
	}
	return false
}

// clientReadsConditional reports whether the client's verified read-order
// contains a conditional read — a field guarded by a runtime discriminator
// (e.g. "mode <= 1", "controlMode != 0", "destroyType == 4") rather than the
// loop-iteration guard ("loop X"). When the CLIENT branches on a runtime value,
// its read SHAPE is value-dependent while Atlas writes unconditionally, so a flat
// positional diff cannot faithfully compare them (the symmetric case to an
// Atlas-side data-dependent branch). Loop guards are excluded — those are handled
// by the diff's loop-body equivalence, not a flat-invalid branch.
func clientReadsConditional(f idasrc.Fields) bool {
	for _, c := range f.Calls {
		g := strings.TrimSpace(c.Guard)
		if g != "" && !strings.HasPrefix(g, "loop ") {
			return true
		}
	}
	return false
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
	return diff.WorstVerdict(rows)
}

func writeSummary(outDir string, summary []report.Packet) error {
	var b strings.Builder
	b.WriteString("# Audit summary\n\n")
	b.WriteString("> ❌/🔍 rows are dispositioned accepted-exclusions (export read-order truncation, opaque register-boundary types, version-absent/representation-equivalence) — see ../../ida-exports/_pending.md. Zero open actionable deferrals (task-080).\n\n")
	b.WriteString("| Packet | Verdict | Atlas file |\n|---|---|---|\n")
	for _, p := range summary {
		fmt.Fprintf(&b, "| [%s](%s.md) | %s | `%s` |\n", p.WriterName, p.WriterName, p.Verdict.Symbol(), p.AtlasFile)
	}
	return os.WriteFile(filepath.Join(outDir, "SUMMARY.md"), []byte(b.String()), 0o644)
}
